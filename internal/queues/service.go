package queues

import (
	"github.com/streadway/amqp"

	"github.com/mikarios/imageresizer/pkg/dtos/imagedto"
)

type Job struct {
	ImageJob *imagedto.ImageProcessJobData
}

type Q interface {
	Publish(job *Job) error
	Consume() (<-chan amqp.Delivery, error)
	Close()
}
