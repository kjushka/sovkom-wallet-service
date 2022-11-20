package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v9"
	"github.com/pkg/errors"
	"time"
	"wallet-service/internal/config"
	"wallet-service/internal/currency_helpers"
)

type Cache interface {
	GetAvailableCurrencies(ctx context.Context) ([]currency_helpers.CurrencyWithBanStatus, error)
	SetAvailableCurrencies(ctx context.Context, availableCurrencies []currency_helpers.CurrencyWithBanStatus) error
	CleanCacheForAvailableCurrencies(ctx context.Context) error

	GetCurrencyLastRate(
		ctx context.Context,
		currencyCodeBase currency_helpers.CurrencyCode,
		currencyCodeSecond currency_helpers.CurrencyCode,
	) (*currency_helpers.CurrencyRate, error)
	SetCurrencyLastRate(ctx context.Context, currencyRates *currency_helpers.CurrencyRates) error

	GetTimestampRate(
		ctx context.Context,
		currencyCodeBase currency_helpers.CurrencyCode,
		currencyCodeSecond currency_helpers.CurrencyCode,
	) (*currency_helpers.CurrencyTimelineRate, error)
	SaveTimestampRate(ctx context.Context, rate *currency_helpers.CurrencyTimelineRate) error
}

func InitCache(cfg *config.Config) (Cache, error) {
	rdb := &Redis{
		rds: redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("redis:%s", cfg.CachePort),
			Password: "",
			DB:       0,
		}),
	}

	_, err := rdb.rds.Ping(context.Background()).Result()
	if err != nil {
		return nil, errors.Wrap(err, "error in ping redis")
	}

	return rdb, nil
}

type Redis struct {
	rds *redis.Client
}

func (r *Redis) GetAvailableCurrencies(ctx context.Context) ([]currency_helpers.CurrencyWithBanStatus, error) {
	jsonData, err := r.rds.Get(ctx, currency_helpers.AvailableCurrencies).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}

		return nil, errors.Wrap(err, "error in getting available currencies")
	}

	var result []currency_helpers.CurrencyWithBanStatus
	err = json.Unmarshal([]byte(jsonData), &result)
	if err != nil {
		return nil, errors.Wrap(err, "parse currency currency availability data")
	}

	return result, nil
}

func (r *Redis) SetAvailableCurrencies(ctx context.Context, availableCurrencies []currency_helpers.CurrencyWithBanStatus) error {
	data, err := json.Marshal(availableCurrencies)
	if err != nil {
		return errors.Wrap(err, "error in marshal data for redis")
	}
	saved, err := r.rds.Set(ctx, currency_helpers.AvailableCurrencies, string(data), time.Hour*24).Result()
	if err != nil {
		return errors.Wrap(err, "save available currencies")
	}

	if saved == "" {
		return errors.New("save no info")
	}

	return nil
}

func (r *Redis) CleanCacheForAvailableCurrencies(ctx context.Context) error {
	count, err := r.rds.Del(ctx, currency_helpers.AvailableCurrencies).Result()
	if err != nil {
		return errors.Wrap(err, "del available currencies")
	}

	if count == 0 {
		return errors.New("deleted no info")
	}

	return err
}

func (r *Redis) GetCurrencyLastRate(
	ctx context.Context,
	currencyCodeBase currency_helpers.CurrencyCode,
	currencyCodeSecond currency_helpers.CurrencyCode,
) (*currency_helpers.CurrencyRate, error) {
	jsonData, err := r.rds.HGet(ctx, currency_helpers.CurrentTimeRateCollection, currencyCodeBase.String()).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}

		return nil, errors.Wrap(err, "get currency last rate error")
	}

	var result currency_helpers.CurrencyRates
	err = json.Unmarshal([]byte(jsonData), &result)
	if err != nil {
		return nil, errors.Wrap(err, "parse currency rate data")
	}

	return result.ToResultRate(currencyCodeSecond), nil
}

func (r *Redis) SetCurrencyLastRate(ctx context.Context, currencyRates *currency_helpers.CurrencyRates) error {
	data, err := json.Marshal(currencyRates)
	if err != nil {
		return errors.Wrap(err, "error in marshal data for redis")
	}
	count, err := r.rds.HSet(
		ctx, currency_helpers.CurrentTimeRateCollection, currencyRates.Base.String(), string(data),
	).Result()
	if err != nil {
		return errors.Wrap(err, "save currency last rate")
	}

	if count == 0 {
		return errors.New("save no info")
	}

	return nil
}

func (r *Redis) GetTimestampRate(
	ctx context.Context,
	currencyCodeBase currency_helpers.CurrencyCode,
	currencyCodeSecond currency_helpers.CurrencyCode,
) (*currency_helpers.CurrencyTimelineRate, error) {
	jsonData, err := r.rds.HGet(
		ctx,
		currency_helpers.TimeCollection,
		fmt.Sprintf("%s:%s", currencyCodeBase.String(), currencyCodeSecond.String()),
	).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}

		return nil, errors.Wrap(err, "get timestamp rate error")
	}

	var result currency_helpers.CurrencyTimelineRate
	err = json.Unmarshal([]byte(jsonData), &result)
	if err != nil {
		return nil, errors.Wrap(err, "parse timestamp rate data")
	}

	return &result, nil
}

func (r *Redis) SaveTimestampRate(ctx context.Context, rate *currency_helpers.CurrencyTimelineRate) error {
	data, err := json.Marshal(rate)
	if err != nil {
		return errors.Wrap(err, "error in marshal data for redis")
	}
	count, err := r.rds.HSet(
		ctx,
		currency_helpers.TimeCollection,
		fmt.Sprintf("%s:%s", rate.Base.String(), rate.Second.String()),
		string(data),
	).Result()
	if err != nil {
		return errors.Wrap(err, "save currency last rate")
	}

	if count == 0 {
		return errors.New("save no info")
	}

	return nil
}
