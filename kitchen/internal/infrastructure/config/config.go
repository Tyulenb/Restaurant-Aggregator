package config

import (
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	ListenAddress string        `env:"LISTEN_ADDRESS" defaultEnv:":8080"`
	ReadTimeout   time.Duration `env:"READ_TIMEOUT" envDefault:"2s"`
	WriteTimeout  time.Duration `env:"WRITE_TIMEOUT" envDefault:"10s"`
	ShutdownGrace time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"5s"`
	IdleTimeout   time.Duration `env:"HTTP_IDLE_TIMEOUT" envDefault:"60s"`
}

func NewConfig() (*Config, error) {
	cfg := new(Config)
	err := env.Parse(cfg)
	return cfg, err
}
