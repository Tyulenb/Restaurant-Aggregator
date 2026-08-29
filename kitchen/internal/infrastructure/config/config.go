package config

import (
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	ListenAddress          string        `env:"LISTEN_ADDRESS" envDefault:":8080"`
	DatabaseURL            string        `env:"DATABASE_URL,required"`
	PartnerAPIKey          string        `env:"PARTNER_API_KEY,required"`
	PartnerEstablishmentID string        `env:"PARTNER_ESTABLISHMENT_ID,required"`
	ReadTimeout            time.Duration `env:"READ_TIMEOUT" envDefault:"2s"`
	WriteTimeout           time.Duration `env:"WRITE_TIMEOUT" envDefault:"10s"`
	ShutdownGrace          time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"5s"`
	IdleTimeout            time.Duration `env:"HTTP_IDLE_TIMEOUT" envDefault:"60s"`
}

func NewConfig() (*Config, error) {
	cfg := new(Config)
	err := env.Parse(cfg)
	return cfg, err
}
