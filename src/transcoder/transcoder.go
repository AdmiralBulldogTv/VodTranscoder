package transcoder

import (
	"sync"

	"github.com/AdmiralBulldogTv/VodTranscoder/src/global"
	jsoniter "github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
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

	go func() {
		defer ch.Close()
		for {
			select {
			case job := <-jobsCh:
				select {
				case rawJob := <-msgQueue:
					if err := json.Unmarshal(rawJob.Body, &job.VodTranscodeJob); err != nil {
						logrus.Errorf("bad json structure in job queue: %s: %s", err.Error(), rawJob.Body)
						_ = rawJob.Nack(false, false)
						continue
					}
					wg.Add(1)
					go func() {
						if job.Process(gCtx) {
							_ = rawJob.Nack(false, true)
						} else {
							_ = rawJob.Ack(false)
						}
						wg.Done()
						jobsCh <- job
					}()
				case <-gCtx.Done():
					return
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
