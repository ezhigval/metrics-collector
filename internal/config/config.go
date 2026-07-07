package config

import (
	"github.com/ezhigval/go-toolkit/config"
)

type Config struct {
	Port        string `env:"PORT" envDefault:"8089"`
	LogLevel    string `env:"LOG_LEVEL" envDefault:"info"`
	LogFormat   string `env:"LOG_FORMAT" envDefault:"json"`
	DatabaseURL string `env:"DATABASE_URL,required"`
	ServiceName string `env:"SERVICE_NAME" envDefault:"metrics-collector"`
	EnableOTel  bool   `env:"ENABLE_OTEL" envDefault:"true"`
}

func MustLoad() Config {
	return config.MustLoad[Config]()
}
