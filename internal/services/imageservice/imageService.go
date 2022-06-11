package imageservice

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"reflect"
	"runtime"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/streadway/amqp"

	"github.com/mikarios/golib/logger"

	"github.com/mikarios/imageresizer/internal/imagehelper"
	"github.com/mikarios/imageresizer/internal/services/cdnservice"
	"github.com/mikarios/imageresizer/internal/services/config"
	"github.com/mikarios/imageresizer/pkg/dtos/imagedto"
)

var (
	once         sync.Once
	jobChan      chan *imagedto.ImageProcessJob
	imageJobChan chan *imageJob
	noOfWorkers  int
	finishedChan chan interface{}
)

type imageJob struct {
	*imagehelper.ImageJob
	errorChan chan []error
}

// AddImageJob is used to add a new job to the channel. Since this is the only function needed A GetInstance which would
// return the jobChan is not needed.
func AddImageJob(job *imagedto.ImageProcessJob) {
	jobChan <- job
}

func Init() <-chan *imagedto.ImageProcessJob {
	once.Do(func() {
		cfg := config.GetInstance()

		maxProcesses := runtime.GOMAXPROCS(0)
		noOfWorkers = 2 * maxProcesses

		if cfg.ImageConfig.NumberOfWorkers > 0 {
			noOfWorkers = cfg.ImageConfig.NumberOfWorkers
		}

		logger.Debug(context.Background(), fmt.Sprintf("found %v threads, spawning %v workers", maxProcesses, noOfWorkers))

		jobChan = make(chan *imagedto.ImageProcessJob)
		imageJobChan = make(chan *imageJob)
		finishedChan = make(chan interface{})

		for i := 0; i < noOfWorkers; i++ {
			go spawnWorker(cfg)
		}

		go listenForJobs()
	})

	return jobChan
}

func Destroy() {
	close(jobChan)

	for i := 0; i < noOfWorkers; i++ {
		<-finishedChan
	}
}

func listenForJobs() {
	cfg := config.GetInstance()
	cdn := cdnservice.GetInstance()
	ctx := context.Background()

	for job := range jobChan {
		var err error

		logger.Debug(ctx, "received new job for shop ID", job.Data.ShopID)

		now := time.Now()
		errorChannel := make(chan []error)
		collectedErrors := make([]error, 0)
		listOfFiles := make(map[string]interface{})

		if job.Data.ShopID != 0 {
			start := time.Now()
			baseImagePath := imagehelper.ImageSubPath("", &job.Data.ShopID, "", nil, nil, nil, nil, "")

			listOfFiles, err = cdn.ListFilesToMap("", path.Join(cfg.CDN.ImagesFolder, baseImagePath))
			if err != nil {
				listOfFiles = nil
			}

			logger.Debug(ctx, fmt.Sprintf("LIST: %v finished. Took: %v", baseImagePath, time.Since(start)))
		}

		go func() {
			for _, imgJob := range job.Data.Images {
				newImageJob := imageJob{
					ImageJob: &imagehelper.ImageJob{
						ImageStruct:    imgJob,
						ShopID:         job.Data.ShopID,
						ImageExtension: job.Data.ImageExtension,
						ImagesOnCdn:    &listOfFiles,
					},
					errorChan: errorChannel,
				}

				imageJobChan <- &newImageJob
			}

			deleteImageJob := imageJob{
				ImageJob: &imagehelper.ImageJob{
					DeleteImages: job.Data.DeleteImages,
				},
				errorChan: errorChannel,
			}
			imageJobChan <- &deleteImageJob
		}()

		for i := 0; i < len(job.Data.Images)+1; i++ {
			if imageErrors := <-errorChannel; len(imageErrors) > 0 {
				collectedErrors = append(collectedErrors, imageErrors...)
			}
		}

		close(errorChannel)
		logger.Debug(ctx, fmt.Sprintf("job for shop ID: %v finished. Took: %v", job.Data.ShopID, time.Since(now)))

		if len(collectedErrors) > 0 {
			for _, err = range collectedErrors {
				logger.Error(ctx, err, "unable to process job", job.Data.ShopID)
			}

			// imageProcessErrorsNotification(ctx, collectedErrors, job)
			// If job is added from API call do not respond to queue
			if reflect.DeepEqual(job.QueueJob, amqp.Delivery{}) {
				continue
			}

			go func() {
				time.Sleep(time.Minute)

				_ = job.QueueJob.Nack(false, !job.QueueJob.Redelivered)
			}()
		} else {
			// If job is added from API call do not respond to queue
			if reflect.DeepEqual(job.QueueJob, amqp.Delivery{}) {
				continue
			}

			_ = job.QueueJob.Ack(false)
		}
	}

	close(imageJobChan)
}

func spawnWorker(cfg *config.Config) {
	defer func() {
		finishedChan <- struct{}{}
	}()

	ctx := context.Background()

	for job := range imageJobChan {
		if cfg.LambdaConfig.Function != "" {
			if err := callLambdaProcessJob(job.ImageJob, &cfg.LambdaConfig); err != nil {
				job.errorChan <- []error{err}
			} else {
				job.errorChan <- nil
			}
		} else {
			job.errorChan <- imagehelper.ProcessJobImage(ctx, job.ImageJob)
		}
	}
}

func callLambdaProcessJob(job *imagehelper.ImageJob, lambdaConfig *config.LambdaConfig) error {
	cdnConfig := config.GetInstance().CDN
	scale := make([]*int, 0)
	crop := make([]*imagedto.Dimensions, 0)
	minXMaxY := make([]*imagedto.Dimensions, 0)
	minYMaxX := make([]*imagedto.Dimensions, 0)

	for _, scaleDimension := range job.ScaleDimensionMax {
		imagePath := imagehelper.ImageSubPath("", &job.ShopID, job.ProductID, scaleDimension, nil, nil, nil, job.Name)
		if _, ok := (*job.ImagesOnCdn)[path.Join(cdnConfig.ImagesFolder, imagePath)]; !ok {
			scale = append(scale, scaleDimension)
		}
	}

	for _, cropDimension := range job.CropDimensions {
		imagePath := imagehelper.ImageSubPath("", &job.ShopID, job.ProductID, nil, cropDimension, nil, nil, job.Name)
		if _, ok := (*job.ImagesOnCdn)[path.Join(cdnConfig.ImagesFolder, imagePath)]; !ok {
			crop = append(crop, cropDimension)
		}
	}

	for _, v := range job.MinXMaxY {
		imagePath := imagehelper.ImageSubPath("", &job.ShopID, job.ProductID, nil, nil, v, nil, job.Name)
		if _, ok := (*job.ImagesOnCdn)[path.Join(cdnConfig.ImagesFolder, imagePath)]; !ok {
			minXMaxY = append(minXMaxY, v)
		}
	}

	for _, v := range job.MinYMaxX {
		imagePath := imagehelper.ImageSubPath("", &job.ShopID, job.ProductID, nil, nil, nil, v, job.Name)
		if _, ok := (*job.ImagesOnCdn)[path.Join(cdnConfig.ImagesFolder, imagePath)]; !ok {
			minYMaxX = append(minYMaxX, v)
		}
	}

	job.ScaleDimensionMax, job.CropDimensions, job.MinXMaxY, job.MinYMaxX = scale, crop, minXMaxY, minYMaxX

	if len(scale) == 0 && len(crop) == 0 && len(minXMaxY) == 0 && len(minXMaxY) == 0 {
		return nil
	}

	sess := session.Must(session.NewSessionWithOptions(session.Options{SharedConfigState: session.SharedConfigEnable}))
	client := lambda.New(
		sess,
		&aws.Config{
			Credentials: credentials.NewStaticCredentials(lambdaConfig.ID, lambdaConfig.Secret, lambdaConfig.Token),
			Region:      aws.String(lambdaConfig.Region),
		},
	)

	payload, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("error marshalling processimage lambda request: %w", err)
	}

	_, err = client.Invoke(&lambda.InvokeInput{FunctionName: &lambdaConfig.Function, Payload: payload})
	if err != nil {
		return fmt.Errorf("error calling processimage lambda: %w", err)
	}

	return nil
}
