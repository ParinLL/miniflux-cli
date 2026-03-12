package config

import (
	"os"
	"strings"
)

const (
	DefaultBaseURL = "http://127.0.0.1:8080/v1/"
	EnvBaseURL     = "MINIFLUX_BASE_URL"
	EnvUsername    = "MINIFLUX_USERNAME"
	EnvPassword    = "MINIFLUX_PASSWORD"
	EnvToken       = "MINIFLUX_API_TOKEN"
)

type Config struct {
	BaseURL  string
	Username string
	Password string
	Token    string
}

func FromEnv() Config {
	return Config{
		BaseURL:  valueOrDefault(os.Getenv(EnvBaseURL), DefaultBaseURL),
		Username: os.Getenv(EnvUsername),
		Password: os.Getenv(EnvPassword),
		Token:    os.Getenv(EnvToken),
	}
}

func valueOrDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
