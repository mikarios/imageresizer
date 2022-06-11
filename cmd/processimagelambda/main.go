package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"

	"github.com/mikarios/imageresizer/internal/imagehelper"
)

var errImageProcess = errors.New("image process error")

func main() {
	lambda.Start(Handler)
}

func Handler(ctx context.Context, job *imagehelper.ImageJob) error {
	collected := imagehelper.ProcessJobImage(ctx, job)

	if len(collected) > 0 {
		s := make([]string, len(collected))

		for i, e := range collected {
			s[i] = e.Error()
		}

		return imageProcessError(strings.Join(s, "|"))
	}

	return nil
}

func imageProcessError(msg string) error {
	return fmt.Errorf("%w: %s", errImageProcess, msg)
}
