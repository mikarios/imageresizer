package queueservice

import (
	"context"
	"sync"

	"github.com/streadway/amqp"

	"github.com/mikarios/golib/logger"

	"github.com/mikarios/imageresizer/internal/queues"
	"github.com/mikarios/imageresizer/internal/services/config"
	"github.com/mikarios/imageresizer/pkg/dtos/imagedto"
)

type Instance struct {
	imagePublisher *queues.ImageQ
	imageConsumer  *queues.ImageQ
}

var (
	once     sync.Once
	instance *Instance
)

func GetInstance() *Instance {
	if instance == nil {
		Init(false, false, false, false)
	}

	return instance
}

func Init(imageP, imageC, statsP, statsC bool) *Instance {
	var err error

	cfg := config.GetInstance()

	once.Do(func() {
		instance = &Instance{}
		if imageP {
			if instance.imagePublisher, err = queues.ImagePublisher(&cfg.RabbitMQConfig); err != nil {
				logger.Panic(context.Background(), err, "could not create publisher")
			}
		}

		if imageC {
			if instance.imageConsumer, err = queues.ImageConsumer(&cfg.RabbitMQConfig); err != nil {
				logger.Panic(context.Background(), err, "could not create consumer")
			}
		}
	})

	return instance
}

func Destroy() {
	if instance.imagePublisher != nil {
		instance.imagePublisher.Close()
	}

	if instance.imageConsumer != nil {
		instance.imageConsumer.Close()
	}
}

func (i *Instance) ImagePublish(job *imagedto.ImageProcessJobData) error {
	return i.imagePublisher.Publish(&queues.Job{ImageJob: job})
}

func (i *Instance) ImageConsume() (<-chan amqp.Delivery, error) {
	return i.imageConsumer.Consume()
}
