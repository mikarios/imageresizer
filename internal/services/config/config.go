package config

// Config holds the main config for all servers.
type Config struct {
	DEV            bool `servers:"imageresizer" envconfig:"DEV" required:"true"`
	LOG            LogConfig
	HTTP           HTTPConfig
	CDN            CDNConfig
	RabbitMQConfig RabbitMQConfig
	ImageConfig    ImageConfig
	LambdaConfig   LambdaConfig
}

type LambdaConfig struct {
	Function string `servers:"imageresizer" optional:"true" envconfig:"LAMBDA_FUNC"`
	ID       string `servers:"imageresizer" optional:"true" envconfig:"LAMBDA_ID"`
	Secret   string `servers:"imageresizer" optional:"true" envconfig:"LAMBDA_SECRET"`
	Token    string `servers:"imageresizer" optional:"true" envconfig:"LAMBDA_TOKEN"`
	Region   string `servers:"imageresizer" optional:"true" envconfig:"LAMBDA_REGION"`
}

type ImageConfig struct {
	NumberOfWorkers int    `servers:"imageresizer" optional:"true" envconfig:"IMG_WORKERS_NUMBER"`
	ImageServerUser string `servers:"imageresizer" envconfig:"IMG_USERNAME"`
	ImageServerPass string `servers:"imageresizer" envconfig:"IMG_PASSWORD"`
}

type CDNConfig struct {
	Key          string `servers:"imageresizer" envconfig:"CDN_KEY"`
	Secret       string `servers:"imageresizer" envconfig:"CDN_SECRET"`
	Endpoint     string `servers:"imageresizer" envconfig:"CDN_ENDPOINT"`
	Bucket       string `servers:"imageresizer" envconfig:"CDN_BUCKET"`
	Region       string `servers:"imageresizer" envconfig:"CDN_REGION"`
	ImagesFolder string `servers:"imageresizer" envconfig:"CDN_IMAGES_FOLDER"`
}

type LogConfig struct {
	Format string `servers:"imageresizer" optional:"true" envconfig:"LOG_FORMAT"`
	Level  string `servers:"imageresizer" optional:"true" envconfig:"LOG_LEVEL"`
	Trace  bool   `servers:"imageresizer" optional:"true" envconfig:"LOG_TRACE"`
}

type HTTPConfig struct {
	IP       string `servers:"imageresizer" envconfig:"HTTP_IP"`
	Port     string `servers:"imageresizer" envconfig:"HTTP_PORT"`
	PortTLS  string `servers:"imageresizer" optional:"true" envconfig:"HTTPS_PORT"`
	CertFile string `servers:"imageresizer" optional:"true" envconfig:"HTTPS_CERT"`
	KeyFile  string `servers:"imageresizer" optional:"true" envconfig:"HTTPS_KEY"`
}

type RabbitMQConfig struct {
	URL string `servers:"imageresizer" envconfig:"RABBITMQ_URL"`
}
