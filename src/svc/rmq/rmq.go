package rmq

import (
	"context"
	"time"

	"github.com/AdmiralBulldogTv/VodTranscoder/src/instance"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

type RmqInst struct {
	conn      *amqp.Connection
	ch        *amqp.Channel
	noopQueue string
}

func New(ctx context.Context, opts SetupOptions) (instance.RMQ, error) {
	conn, err := amqp.DialConfig(opts.URI, amqp.Config{
		Heartbeat: time.Second * 5,
	})
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	_, err = ch.QueueDeclare(opts.NoopQueue, false, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	_, err = ch.QueueDeclare(opts.TranscoderTaskQueueName, true, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	return &RmqInst{
		conn:      conn,
		ch:        ch,
		noopQueue: opts.NoopQueue,
	}, nil
}

func (r *RmqInst) Publish(queueName string, msg amqp.Publishing) error {
	return r.ch.Publish("", queueName, false, false, msg)
}

func (r *RmqInst) Consume(queueName string, consumer string) (*amqp.Channel, <-chan amqp.Delivery, error) {
	ch, err := r.conn.Channel()
	if err != nil {
		return nil, nil, err
	}

	go func() {
		tick := time.NewTicker(time.Second * 5)
		defer tick.Stop()
		defer ch.Close()
		for range tick.C {
			if err := ch.Publish("", r.noopQueue, false, false, amqp.Publishing{
				Headers: amqp.Table{
					"x-message-ttl": int32(60000),
				},
				Body: []byte{'b', 'a', 't', 'c', 'h', 'e', 's', 't'},
			}); err != nil {
				logrus.Error("failed to publish to noop queue: ", err)
				return
			}
		}
	}()

	msg, err := ch.Consume(queueName, consumer, false, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return nil, nil, err
	}

	return ch, msg, nil
}

type SetupOptions struct {
	URI                     string
	TranscoderTaskQueueName string
	NoopQueue               string
}
