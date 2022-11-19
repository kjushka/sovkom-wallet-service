package config

import (
	"os"
	"time"

	"github.com/pkg/errors"
)

type Config struct {
	DBHost, DBPort, Database, DBUser, DBPass string
	DBTimeout                                time.Duration
	CachePort                                string
	CacheTimeout                             time.Duration
	ExchangerAPIURL                          string
	ExchangerAPITimeout                      time.Duration
}

func InitConfig() (*Config, error) {
	pgHost, ok := os.LookupEnv("PG_HOST")
	if !ok {
		return nil, errors.New("PG_ADDR not found")
	}
	pgPort, ok := os.LookupEnv("PG_PORT")
	if !ok {
		return nil, errors.New("PG_PORT not found")
	}
	database, ok := os.LookupEnv("PG_WALLET_DATABASE")
	if !ok {
		return nil, errors.New("PG_WALLET_DATABASE not found")
	}
	pgUser, ok := os.LookupEnv("PG_USER")
	if !ok {
		return nil, errors.New("PG_USER not found")
	}
	pgPass, ok := os.LookupEnv("PG_PASS")
	if !ok {
		return nil, errors.New("PG_PASS not found")
	}
	pgTimeoutStr, ok := os.LookupEnv("PG_TIMEOUT")
	if !ok {
		return nil, errors.New("PG_TIMEOUT not found")
	}
	pgTimeout, err := time.ParseDuration(pgTimeoutStr)
	if err != nil {
		return nil, errors.Wrap(err, "parse pgsql timeout")
	}

	redisPort, ok := os.LookupEnv("REDIS_PORT")
	if !ok {
		return nil, errors.New("REDIS_PORT not found")
	}
	redisTimeoutStr, ok := os.LookupEnv("REDIS_TIMEOUT")
	if !ok {
		return nil, errors.New("REDIS_TIMEOUT not found")
	}
	redisTimeout, err := time.ParseDuration(redisTimeoutStr)
	if err != nil {
		return nil, errors.Wrap(err, "parse redis timeout")
	}

	cbrApiUrl, ok := os.LookupEnv("CBR_API_URL")
	if !ok {
		return nil, errors.New("CBR_API_URL not found")
	}
	cbrApiTimeoutStr, ok := os.LookupEnv("CBR_API_TIMEOUT")
	if !ok {
		return nil, errors.New("CBR_API_TIMEOUT not found")
	}
	cbrApiTimeout, err := time.ParseDuration(cbrApiTimeoutStr)
	if err != nil {
		return nil, errors.Wrap(err, "parse cbr api timeout")
	}

	config := &Config{
		DBHost:              pgHost,
		DBPort:              pgPort,
		DBUser:              pgUser,
		DBPass:              pgPass,
		Database:            database,
		DBTimeout:           pgTimeout,
		CachePort:           redisPort,
		CacheTimeout:        redisTimeout,
		ExchangerAPIURL:     cbrApiUrl,
		ExchangerAPITimeout: cbrApiTimeout,
	}
	return config, nil
}
