package config

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/mikarios/golib/logger"

	"github.com/mikarios/imageresizer/internal/constants"
	"github.com/mikarios/imageresizer/internal/exceptions"
)

const (
	optionalTagName  = "optional"
	serversTagName   = "servers"
	envConfigTagName = "envconfig"
)

func validateEnvironment(instance *Config, serverType constants.ServerType) {
	instanceType := reflect.TypeOf(instance)

	missingValues := make([]string, 0)

	for i := 0; i < instanceType.NumField(); i++ {
		field := instanceType.Field(i)
		if field.Type.Kind() == reflect.Struct {
			validateStructVariable(field.Type, serverType, &missingValues)
			continue
		}

		validateField(&field, serverType, &missingValues)
	}

	if len(missingValues) > 0 {
		logger.Panic(
			context.Background(),
			exceptions.ErrIncompleteEnvironment,
			fmt.Sprintf("missing non-optional variables %v", strings.Join(missingValues, ", ")),
		)
	}
}

func validateStructVariable(configType reflect.Type, serverType constants.ServerType, missingValues *[]string) {
	for i := 0; i < configType.NumField(); i++ {
		field := configType.Field(i)

		if field.Type.Kind() == reflect.Struct {
			validateStructVariable(field.Type, serverType, missingValues)
			continue
		}

		validateField(&field, serverType, missingValues)
	}
}

func validateField(field *reflect.StructField, serverType constants.ServerType, missingValues *[]string) {
	isOptional := field.Tag.Get(optionalTagName)
	serversValue := field.Tag.Get(serversTagName)

	isForServer := strings.Contains(serversValue, string(serverType))
	if !isForServer {
		return
	}

	if isOptional == "true" {
		return
	}

	fieldName := field.Tag.Get(envConfigTagName)

	envValue := os.Getenv(fieldName)
	if envValue == "" {
		*missingValues = append(*missingValues, fieldName)
	}
}
