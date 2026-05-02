package config

import (
	"os"
)

type Config struct {
	App  AppConfig
	HTTP HTTPConfig
	Log  LogConfig
}

type AppConfig struct {
	Name    string
	Version string
	Env     string
}

type HTTPConfig struct {
	Host string
	Port string
}

type LogConfig struct {
	Level  string
	Format string
}

func Load() *Config {
	return &Config{
		App: AppConfig{
			Name:    getEnv("APP_NAME", "ms-storage"),
			Version: getEnv("APP_VERSION", "1.0.0"),
			Env:     getEnv("APP_ENV", "development"),
		},
		HTTP: HTTPConfig{
			Host: getEnv("HTTP_HOST", "0.0.0.0"),
			Port: getEnv("HTTP_PORT", "8080"),
		},
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}
