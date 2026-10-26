package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

const minJWTSecretLength = 32

type Config struct {
	HTTPAddr string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	RedisAddr     string
	RedisPassword string
	RedisDB       int

	JWTSecret string
	JWTTTL    time.Duration

	CacheTTL time.Duration

	RateLimitRPS   float64
	RateLimitBurst int
}

func Load() (Config, error) {
	var errs []error
	cfg := Config{
		HTTPAddr: envOr("HTTP_ADDR", ":8080"),

		DBHost:     requiredEnv("DB_HOST", &errs),
		DBPort:     envOr("DB_PORT", "3306"),
		DBUser:     requiredEnv("DB_USER", &errs),
		DBPassword: requiredEnv("DB_PASSWORD", &errs),
		DBName:     requiredEnv("DB_NAME", &errs),

		RedisAddr:     requiredEnv("REDIS_ADDR", &errs),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		RedisDB:       parsedEnvOr("REDIS_DB", "0", strconv.Atoi, &errs),

		JWTSecret: requiredEnv("JWT_SECRET", &errs),
		JWTTTL:    parsedEnvOr("JWT_TTL", "24h", time.ParseDuration, &errs),

		CacheTTL: parsedEnvOr("CACHE_TTL", "5m", time.ParseDuration, &errs),

		RateLimitRPS:   parsedEnvOr("RATE_LIMIT_RPS", "20", parseFloat, &errs),
		RateLimitBurst: parsedEnvOr("RATE_LIMIT_BURST", "40", strconv.Atoi, &errs),
	}
	if cfg.JWTSecret != "" && len(cfg.JWTSecret) < minJWTSecretLength {
		errs = append(errs, fmt.Errorf("JWT_SECRET must be at least %d characters", minJWTSecretLength))
	}
	return cfg, errors.Join(errs...)
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func requiredEnv(key string, errs *[]error) string {
	value := os.Getenv(key)
	if value == "" {
		*errs = append(*errs, fmt.Errorf("%s is required", key))
	}
	return value
}

func parsedEnvOr[T any](key, fallback string, parse func(string) (T, error), errs *[]error) T {
	value, err := parse(envOr(key, fallback))
	if err != nil {
		*errs = append(*errs, fmt.Errorf("%s: %w", key, err))
	}
	return value
}

func parseFloat(value string) (float64, error) {
	return strconv.ParseFloat(value, 64)
}
