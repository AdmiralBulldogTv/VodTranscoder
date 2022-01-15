package rmq

import (
	"context"

	"github.com/AdmiralBulldogTv/VodTranscoder/src/instance"
	"github.com/streadway/amqp"
)

type RmqInst struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func New(ctx context.Context, opts SetupOptions) (instance.RMQ, error) {
	conn, err := amqp.Dial(opts.URI)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	_, err = ch.QueueDeclare(opts.TranscoderTaskQueueName, true, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	_, err = ch.QueueDeclare(opts.ApiTaskQueueName, true, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	return &RmqInst{
		conn: conn,
		ch:   ch,
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
	ApiTaskQueueName        string
}
