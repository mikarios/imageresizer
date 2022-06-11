package queues

import (
	"encoding/json"

	"github.com/streadway/amqp"

	"github.com/mikarios/golib/queue"

	"github.com/mikarios/imageresizer/internal/services/config"
)

const (
	imageExchangeName = "imageExchange"
	imageQueueName    = "imageQueue"
	imageKey          = "imageKey"
)

type ImageQ struct {
	imageQueue *queue.RabbitMQ
}

func ImagePublisher(cfg *config.RabbitMQConfig) (*ImageQ, error) {
	r, err := queue.NewQueue(&queue.RabbitMQConf{URL: cfg.URL})
	if err != nil {
		return nil, err
	}

	if err := declareExchange(r, imageExchangeName); err != nil {
		return nil, err
	}

	if _, err := declareQueue(r, imageQueueName); err != nil {
		return nil, err
	}

	return &ImageQ{imageQueue: r}, nil
}

func ImageConsumer(cfg *config.RabbitMQConfig) (*ImageQ, error) {
	r, err := queue.NewQueue(&queue.RabbitMQConf{URL: cfg.URL})
	if err != nil {
		return nil, err
	}

	if err := declareExchange(r, imageExchangeName); err != nil {
		return nil, err
	}

	if _, err := declareQueue(r, imageQueueName); err != nil {
		return nil, err
	}

	if err := bindQueue(r, imageQueueName, imageKey, imageExchangeName); err != nil {
		return nil, err
	}

	ch := make(chan *amqp.Error)

	go panicOnError(ch, errImageQueue)

	r.Ch.NotifyClose(ch)

	return &ImageQ{imageQueue: r}, nil
}

func (i *ImageQ) Publish(job *Job) error {
	b, err := json.Marshal(job.ImageJob)
	if err != nil {
		return err
	}

	ch := make(chan *amqp.Error)

	go panicOnError(ch, errImageQueue)

	i.imageQueue.Ch.NotifyClose(ch)

	return i.imageQueue.Ch.Publish(
		imageExchangeName,
		imageKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         b,
		},
	)
}

func (i *ImageQ) Consume() (<-chan amqp.Delivery, error) {
	return i.imageQueue.Ch.Consume(
		imageQueueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
}

func (i *ImageQ) Close() {
	_ = i.imageQueue.Ch.Close()
	_ = i.imageQueue.Conn.Close()
}
