package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Port           int           `envconfig:"PORT" default:"8080"`
	DatabaseURL    string        `envconfig:"DATABASE_URL" required:"true"`
	RequestTimeout time.Duration `envconfig:"REQUEST_TIMEOUT" default:"30s"`
	MaxConnections int           `envconfig:"MAX_CONNECTIONS" default:"100"`
	GoEnv          string        `envconfig:"GO_ENV" default:"development"`
	CorsOrigin     string        `envconfig:"CORS_ORIGIN" default:"http://localhost:3000"`

	Redis struct {
		Addr string `envconfig:"REDIS_ADDR" default:"localhost:6379"`
		DB   int    `envconfig:"REDIS_DB" default:"0"`
	}

	Mailer struct {
		Hostname string `envconfig:"MAILER_HOSTNAME" default:""`
		Port     int    `envconfig:"MAILER_PORT" default:"465"`
		Password string `envconfig:"MAILER_PASSWORD" default:""`
		Username string `envconfig:"MAILER_USERNAME" default:""`
		From     string `envconfig:"MAILER_FROM" default:""`
	}
}

func LoadEnv() (*Config, error) {
	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
