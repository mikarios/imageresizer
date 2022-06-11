package queues

import (
	"context"
	"errors"

	"github.com/streadway/amqp"

	"github.com/mikarios/golib/logger"
	"github.com/mikarios/golib/queue"

	"github.com/mikarios/imageresizer/internal/services/config"
)

var errImageQueue = errors.New("got error from rabbitMQ image queue")

func declareExchange(q *queue.RabbitMQ, exchangeName string) error {
	return q.
		Exchange().
		Name(exchangeName).
		Kind("direct").
		Durable(true).
		Declare()
}

func declareQueue(q *queue.RabbitMQ, queueName string) (amqp.Queue, error) {
	return q.Queue().
		Name(queueName).
		Durable(true).
		Declare()
}

func bindQueue(q *queue.RabbitMQ, queueName, key, exchangeName string) error {
	return q.Ch.QueueBind(queueName, key, exchangeName, false, nil)
}

func panicOnError(ch <-chan *amqp.Error, err error) {
	cfg := config.GetInstance()
	ctx := context.Background()

	for errCh := range ch {
		if cfg.DEV {
			logger.Error(ctx, err, errCh)
		} else {
			logger.Panic(context.Background(), err, errCh)
		}
	}
}
