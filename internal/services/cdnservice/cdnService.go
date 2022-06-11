package cdnservice

import (
	"errors"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

var (
	once              sync.Once
	instance          *CdnStruct
	errCdnMoreThanOne = errors.New("cdn returned more than one item")
)

type CdnStruct struct {
	s3            *s3.S3
	defaultBucket *string
}

func GetInstance() *CdnStruct {
	if instance == nil || instance.s3 == nil {
		panic("not initialised")
	}

	return instance
}

func Init(bucket, key, secret, endpoint, region string) *CdnStruct {
	once.Do(func() {
		instance = &CdnStruct{defaultBucket: aws.String(bucket)}
		s3Config := &aws.Config{
			Credentials: credentials.NewStaticCredentials(key, secret, ""),
			Endpoint:    aws.String("https://" + endpoint),
			Region:      aws.String(region),
		}

		newSession, err := session.NewSession(s3Config)
		if err != nil {
			panic(err)
		}

		instance.s3 = s3.New(newSession)
	})

	return instance
}
