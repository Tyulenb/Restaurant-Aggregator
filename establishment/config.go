package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type config struct {
	baseURL          string
	apiKey           string
	pollInterval     time.Duration
	transitionDelay  time.Duration
	retryInterval    time.Duration
	rejectionPercent int
	requestTimeout   time.Duration
}

func loadConfig() (config, error) {
	cfg := config{
		baseURL:          env("KITCHEN_BASE_URL", "http://kitchen:8080"),
		apiKey:           env("PARTNER_API_KEY", "development-api-key"),
		pollInterval:     2 * time.Second,
		transitionDelay:  3 * time.Second,
		retryInterval:    2 * time.Second,
		rejectionPercent: 20,
		requestTimeout:   10 * time.Second,
	}

	var err error
	if cfg.pollInterval, err = durationEnv("POLL_INTERVAL", cfg.pollInterval); err != nil {
		return config{}, err
	}
	if cfg.transitionDelay, err = durationEnv("STATUS_DELAY", cfg.transitionDelay); err != nil {
		return config{}, err
	}
	if cfg.retryInterval, err = durationEnv("RETRY_INTERVAL", cfg.retryInterval); err != nil {
		return config{}, err
	}
	if cfg.requestTimeout, err = durationEnv("REQUEST_TIMEOUT", cfg.requestTimeout); err != nil {
		return config{}, err
	}
	if raw := os.Getenv("REJECTION_PERCENT"); raw != "" {
		cfg.rejectionPercent, err = strconv.Atoi(raw)
		if err != nil || cfg.rejectionPercent < 0 || cfg.rejectionPercent > 100 {
			return config{}, fmt.Errorf("REJECTION_PERCENT must be an integer between 0 and 100")
		}
	}

	return cfg, nil
}

func env(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func durationEnv(name string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", name)
	}
	return value, nil
}
