package transcoder

import (
	"context"
	"sync"

	"github.com/AdmiralBulldogTv/VodTranscoder/src/global"
	jsoniter "github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func New(gCtx global.Context) <-chan struct{} {
	done := make(chan struct{})

	jobsCh := make(chan *Job, gCtx.Config().Transcode.MaxJobs)
	for i := 0; i < gCtx.Config().Transcode.MaxJobs; i++ {
		jobsCh <- &Job{}
	}

	wg := sync.WaitGroup{}
	ch, msgQueue, err := gCtx.Inst().RMQ.Consume(gCtx.Config().RMQ.TranscoderTaskQueue, gCtx.Config().Pod.Name)
	if err != nil {
		logrus.Fatal("failed to consume rmq: ", err)
	}
	closeCh := ch.NotifyClose(make(chan *amqp.Error))
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		defer ch.Close()
		for {
			select {
			case <-closeCh:
				logrus.Warn("RMQ connection closed, reopening...")
				ch, msgQueue, err = gCtx.Inst().RMQ.Consume(gCtx.Config().RMQ.TranscoderTaskQueue, gCtx.Config().Pod.Name)
				if err != nil {
					logrus.Fatal("failed to consume rmq: ", err)
				}
				closeCh = ch.NotifyClose(make(chan *amqp.Error))
				cancel()
				ctx, cancel = context.WithCancel(context.Background())
			case rawJob := <-msgQueue:
				select {
				case job := <-jobsCh:
					if len(rawJob.Body) == 0 {
						_ = rawJob.Nack(false, false)
						continue
					}

					if err := json.Unmarshal(rawJob.Body, &job.VodTranscodeJob); err != nil {
						logrus.Errorf("bad json structure in job queue: %s: %s", err.Error(), rawJob.Body)
						err := rawJob.Nack(false, false)
						if err != nil {
							logrus.Error("failed to ack: ", err)
						}
						continue
					}
					wg.Add(1)
					go func() {
						var err error
						if job.Process(gCtx, ctx) {
							err = rawJob.Nack(false, true)
						} else {
							err = rawJob.Ack(false)
						}
						if err != nil {
							logrus.Error("failed to ack: ", err)
						}
						wg.Done()
						jobsCh <- job
					}()
				case <-gCtx.Done():
					return
				default:
					if len(rawJob.Body) == 0 {
						_ = rawJob.Nack(false, false)
						continue
					}
					if err := rawJob.Nack(false, true); err != nil {
						logrus.Error("failed to nack message: ", err)
					}
				}
			case <-gCtx.Done():
				return
			}
		}
	}()

	go func() {
		<-gCtx.Done()
		wg.Wait()
		close(done)
	}()

	return done
}
