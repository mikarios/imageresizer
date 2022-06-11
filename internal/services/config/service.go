package config

import (
	"context"
	"sync"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"

	"github.com/mikarios/golib/logger"

	"github.com/mikarios/imageresizer/internal/constants"
)

var (
	once           sync.Once
	instance       *Config
	serverToConfig = map[constants.ServerType]string{
		constants.ServerTypes.ImageResizer: "./internal/services/config/daemonConfigTemplates/imageresizer.env",
	}
)

func GetInstance() *Config {
	if instance == nil {
		panic("not initialised")
	}

	return instance
}

func Init(prefix string, serverType constants.ServerType) *Config {
	once.Do(func() {
		instance = &Config{}
		if err := envconfig.Process(prefix, instance); err == nil && !instance.DEV {
			validateEnvironment(instance, serverType)
		} else {
			_ = godotenv.Load(serverToConfig[serverType])

			if err := envconfig.Process(prefix, instance); err != nil {
				logger.Panic(context.Background(), err)
			}
		}
	})

	return instance
}
