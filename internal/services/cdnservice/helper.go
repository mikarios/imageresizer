package cdnservice

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/mikarios/golib/logger"

	"github.com/mikarios/imageresizer/internal/exceptions"
	"github.com/mikarios/imageresizer/internal/services/config"
)

var ErrDeletingImages = errors.New("could not delete images")

func (cdn *CdnStruct) Delete(bucket string, imagePaths []string) error {
	if bucket == "" {
		bucket = *cdn.defaultBucket
	}

	objectIdentifiers := make([]*s3.ObjectIdentifier, 0)

	for _, imagePath := range imagePaths {
		imagePathCheck := strings.TrimSuffix(imagePath, "/")

		if imagePathCheck == config.GetInstance().CDN.ImagesFolder || imagePathCheck == "" {
			return fmt.Errorf("%w: you should not delete root path! %s", exceptions.ErrNotImplemented, imagePath)
		}

		files, err := cdn.ListFiles(bucket, imagePath)
		if err != nil {
			return err
		}

		for j := range files {
			f := files[j]

			objectIdentifiers = append(objectIdentifiers, &s3.ObjectIdentifier{Key: &f})
		}
	}

	deleteErrors := make([]string, 0)

	for i, end := 0, 1000; i < len(objectIdentifiers); i, end = i+1000, end+1000 {
		if end > len(objectIdentifiers) {
			end = len(objectIdentifiers)
		}

		chunk := objectIdentifiers[i:end]

		input := &s3.DeleteObjectsInput{
			Bucket: &bucket,
			Delete: &s3.Delete{
				Objects: chunk,
				Quiet:   nil,
			},
		}

		if _, err := cdn.s3.DeleteObjects(input); err != nil {
			deleteErrors = append(deleteErrors, err.Error())
		}
	}

	if len(deleteErrors) > 0 {
		return fmt.Errorf("%w: %s", ErrDeletingImages, strings.Join(deleteErrors, " | "))
	}

	return nil
}

// FileExists checks for the existence of a file. Returns false if there is a problem OR if more than one
// item are found in cdn. If bucket is not set then the default one is used.
func (cdn *CdnStruct) FileExists(bucket, filePath string) bool {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(filePath),
	}

	if bucket == "" {
		input.Bucket = cdn.defaultBucket
	}

	objects, err := cdn.s3.ListObjectsV2(input)
	if err != nil {
		logger.Error(context.Background(), err, "could not list objects from cdn", input)
		return false
	}

	if len(objects.Contents) > 1 {
		logger.Warning(context.Background(), errCdnMoreThanOne, input, objects.Contents)
		return false
	}

	return len(objects.Contents) == 1
}

func (cdn *CdnStruct) ListFiles(bucket, filePath string) ([]string, error) {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(filePath),
	}

	if bucket == "" {
		input.Bucket = cdn.defaultBucket
	}

	res := make([]string, 0)
	objects := &s3.ListObjectsV2Output{NextContinuationToken: aws.String("")}

	var err error

	for objects.NextContinuationToken != nil {
		if *objects.NextContinuationToken != "" {
			input.ContinuationToken = objects.NextContinuationToken
		}

		objects, err = cdn.s3.ListObjectsV2(input)
		if err != nil {
			logger.Error(context.Background(), err, "could not list objects from cdn", input)
			return res, err
		}

		for _, content := range objects.Contents {
			res = append(res, *content.Key)
		}
	}

	return res, nil
}

func (cdn *CdnStruct) ListFilesToMap(bucket, filePath string) (map[string]interface{}, error) {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(filePath),
	}

	if bucket == "" {
		input.Bucket = cdn.defaultBucket
	}

	res := make(map[string]interface{})
	objects := &s3.ListObjectsV2Output{NextContinuationToken: aws.String("")}

	var err error

	for objects.NextContinuationToken != nil {
		if *objects.NextContinuationToken != "" {
			input.ContinuationToken = objects.NextContinuationToken
		}

		objects, err = cdn.s3.ListObjectsV2(input)
		if err != nil {
			logger.Error(context.Background(), err, "could not list objects from cdn", input)
			return res, err
		}

		for _, content := range objects.Contents {
			res[*content.Key] = nil
		}
	}

	return res, nil
}

// StoreFile stores the given data to the path provided. If bucket is not set then the default one is used.
func (cdn *CdnStruct) StoreFile(bucket, filePath string, file io.ReadSeeker, contentType string) error {
	object := s3.PutObjectInput{
		ACL:    aws.String(s3.ObjectCannedACLPublicRead),
		Body:   file,
		Bucket: aws.String(bucket),
		Key:    aws.String(filePath),
	}

	if contentType != "" {
		object.ContentType = aws.String(contentType)
	}

	if bucket == "" {
		object.Bucket = cdn.defaultBucket
	}

	_, err := cdn.s3.PutObject(&object)

	return err
}
