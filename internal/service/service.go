package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"time"
	"wallet-service/internal/cache"
	"wallet-service/internal/config"
	"wallet-service/internal/currency_helpers"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type Service interface {
	// routes
	GetAvailableCurrencies(w http.ResponseWriter, r *http.Request)
	ChangeCurrencyBanStatus(w http.ResponseWriter, r *http.Request)

	GetCurrentCurrencyRate(w http.ResponseWriter, r *http.Request)
	GetTimelineCurrencyRate(w http.ResponseWriter, r *http.Request)
}

func NewService(db *sqlx.DB, redisCache cache.Cache, cfg *config.Config) Service {
	return &HttpService{
		db:         db,
		redisCache: redisCache,
		cfg:        cfg,
	}
}

type HttpService struct {
	db         *sqlx.DB
	redisCache cache.Cache
	cfg        *config.Config
}

func (s *HttpService) GetAvailableCurrencies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	currencies := make([]currency_helpers.CurrencyCode, 0, len(currency_helpers.CodeToCurrency))
	for curr := range currency_helpers.CodeToCurrency {
		currencies = append(currencies, curr)
	}

	cacheCtx, cancel := context.WithTimeout(ctx, s.cfg.CacheTimeout)
	defer cancel()
	availableCurrencies, err := s.redisCache.GetAvailableCurrencies(cacheCtx)
	if err != nil {
		err = errors.Wrap(err, "error in get available currencies")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if availableCurrencies != nil {
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(availableCurrencies)
		if err != nil {
			err = errors.Wrap(err, "error in marshalling currencies")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		return
	}

	queryBase := `
		select cb.currency, cb.banned
		from currency_bans as cb
		where currency in (?);
	`
	query, params, err := sqlx.In(queryBase, currencies)
	if err != nil {
		err = errors.Wrap(err, "error in prepare query")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	query = s.db.Rebind(query)

	var curr2ban []currency_helpers.CurrencyWithBanStatus

	dbCtx, cancel := context.WithTimeout(ctx, s.cfg.DBTimeout)
	defer cancel()
	err = s.db.SelectContext(dbCtx, &curr2ban, query, params...)
	if err != nil {
		err = errors.Wrap(err, "error in getting currency to ban data")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cur2banMap := make(map[currency_helpers.CurrencyCode]currency_helpers.CurrencyWithBanStatus, len(curr2ban))
	for _, curr := range curr2ban {
		cur2banMap[curr.Currency] = curr
	}

	result := make([]currency_helpers.CurrencyWithBanStatus, 0, len(curr2ban))
	for curr, _ := range currency_helpers.CodeToCurrency {
		if v, ok := cur2banMap[curr]; ok {
			result = append(result, v)
		} else {
			result = append(result, currency_helpers.CurrencyWithBanStatus{
				Currency: curr,
				Banned:   false,
			})
		}
	}

	sort.SliceStable(result, func(i, j int) bool {
		si, sj := result[i], result[j]
		if si.Banned != sj.Banned {
			return si.Banned
		}
		return si.Currency <= sj.Currency
	})

	cacheCtx, cancel = context.WithTimeout(ctx, s.cfg.CacheTimeout)
	defer cancel()
	err = s.redisCache.SetAvailableCurrencies(cacheCtx, result)
	if err != nil {
		log.Printf("error in save available currencies: %s", err.Error())
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(result)
	if err != nil {
		err = errors.Wrap(err, "error in marshalling currencies")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	return
}

func (s *HttpService) ChangeCurrencyBanStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req := struct {
		Currency currency_helpers.CurrencyCode `json:"currency"`
		Banned   bool                          `json:"banned"`
	}{}
	defer r.Body.Close()
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		err = errors.Wrap(err, "error in unmarshalling request")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if _, ok := currency_helpers.CodeToCurrency[req.Currency]; !ok {
		err = errors.New("invalid currency")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := `
		insert into currency_bans (currency, banned) values ($1, $2) 
		on conflict (currency)
		do update set banned = $2 where excluded.currency = $1;
	`
	dbCtx, cancel := context.WithTimeout(ctx, s.cfg.DBTimeout)
	defer cancel()
	_, err = s.db.ExecContext(dbCtx, query, req.Currency, req.Banned)
	if err != nil {
		err = errors.Wrap(err, "error in update currency status")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cacheCtx, cancel := context.WithTimeout(ctx, s.cfg.CacheTimeout)
	defer cancel()
	err = s.redisCache.CleanCacheForAvailableCurrencies(cacheCtx)
	if err != nil {
		log.Printf("error in clean available currencies: %s", err.Error())
	}

	w.WriteHeader(http.StatusOK)
}

func (s *HttpService) GetCurrentCurrencyRate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	currencyCodeBase := currency_helpers.CurrencyCode(r.URL.Query().Get("base"))
	if _, ok := currency_helpers.CodeToCurrency[currencyCodeBase]; !ok {
		err := errors.New("invalid base currency code")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	currencyCodeSecond := currency_helpers.CurrencyCode(r.URL.Query().Get("second"))
	if _, ok := currency_helpers.CodeToCurrency[currencyCodeSecond]; !ok {
		err := errors.New("invalid second currency code")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cacheCtx, cancel := context.WithTimeout(ctx, s.cfg.CacheTimeout)
	defer cancel()
	currencyRate, err := s.redisCache.GetCurrencyLastRate(cacheCtx, currencyCodeBase, currencyCodeSecond)
	if err != nil {
		err = errors.Wrap(err, "error in get currency rate")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if currencyRate != nil {
		rYear, rMonth, rDay := currencyRate.Date.Date()
		nYear, nMonth, nDay := time.Now().Date()
		if rYear < nYear ||
			rYear == nYear && rMonth < nMonth ||
			rYear == nYear && rMonth == nMonth && rDay == nDay {
			w.Header().Set("Content-Type", "application/json")
			err = json.NewEncoder(w).Encode(currencyRate)
			if err != nil {
				err = errors.Wrap(err, "error in marshalling result")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	cbrCtx, cancel := context.WithTimeout(ctx, s.cfg.ExchangerAPITimeout)
	defer cancel()

	todayY, todayM, todayD := time.Now().Year(), time.Now().Month(), time.Now().Day()
	previousDay := time.Date(todayY, todayM, todayD-1, 0, 0, 0, 0, time.UTC)
	req, err := http.NewRequestWithContext(
		cbrCtx,
		http.MethodGet,
		fmt.Sprintf(
			"%s/%s?base=%s&places=4",
			s.cfg.ExchangerAPIURL,
			previousDay.Format("02.01.2006"),
			currencyCodeBase,
		),
		nil,
	)
	if err != nil {
		err = errors.Wrap(err, "error in prepare request")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	client := http.DefaultClient

	exchangerResp, err := client.Do(req)
	if err != nil {
		err = errors.Wrap(err, "error in get new data")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer exchangerResp.Body.Close()

	updatedCurrencyRates := &currency_helpers.CurrencyRatesResponse{}
	err = json.NewDecoder(exchangerResp.Body).Decode(updatedCurrencyRates)
	if err != nil {
		err = errors.Wrap(err, "internal error in read JSON data")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !updatedCurrencyRates.Success {
		err = errors.New("unsuccessful getting new rates")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, ok := updatedCurrencyRates.Rates[currencyCodeSecond]
	if !ok {
		err = errors.Errorf("cannot find rate for '%s'", currencyCodeSecond.String())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cacheCtx, cancel = context.WithTimeout(ctx, s.cfg.CacheTimeout)
	defer cancel()
	err = s.redisCache.SetCurrencyLastRate(ctx, updatedCurrencyRates.CurrencyRates)
	if err != nil {
		log.Printf("error in save new rate: %s", err.Error())
	}

	w.Header().Set("Content-Type", "application/json")
	ratesJson, err := json.Marshal(updatedCurrencyRates.CurrencyRates.ToResultRate(currencyCodeSecond))
	if err != nil {
		err = errors.New("marshal result error")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(ratesJson)
}

func (s *HttpService) GetTimelineCurrencyRate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var err error
	currencyCodeBase := currency_helpers.CurrencyCode(r.URL.Query().Get("base"))
	if _, ok := currency_helpers.CodeToCurrency[currencyCodeBase]; !ok {
		err = errors.New("invalid base currency code")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	currencyCodeSecond := currency_helpers.CurrencyCode(r.URL.Query().Get("second"))
	if _, ok := currency_helpers.CodeToCurrency[currencyCodeSecond]; !ok {
		err = errors.New("invalid second currency code")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var (
		startDate time.Time
		endDate   time.Time
	)

	startDateStr := r.URL.Query().Get("start")
	if startDate, err = time.Parse(currency_helpers.CustomTimeLayout, startDateStr); err != nil {
		err = errors.New("invalid start period date")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	endDateStr := r.URL.Query().Get("end")
	if endDate, err = time.Parse(currency_helpers.CustomTimeLayout, endDateStr); err != nil {
		err = errors.New("invalid end period date")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cacheCtx, cancel := context.WithTimeout(ctx, s.cfg.CacheTimeout)
	defer cancel()
	currencyRate, err := s.redisCache.GetTimestampRate(ctx, currencyCodeBase, currencyCodeSecond)
	if err != nil || currencyRate == nil {
		if err != nil {
			err = errors.Wrap(err, "error in get timestamp rate from cache")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		cbrCtx, cancel := context.WithTimeout(ctx, s.cfg.ExchangerAPITimeout)
		defer cancel()

		todayY, todayM, todayD := time.Now().Year(), time.Now().Month(), time.Now().Day()
		previousDay := time.Date(todayY, todayM, todayD-1, 0, 0, 0, 0, time.UTC)
		previousYear := time.Date(todayY-1, todayM, todayD, 0, 0, 0, 0, time.UTC)
		// какая-то странная бага, не работает today
		url := fmt.Sprintf(
			"%s/timeseries?start_date=%s&end_date=%s&base=%s&symbols=%s&places=4",
			s.cfg.ExchangerAPIURL,
			previousYear.Format(currency_helpers.CustomTimeLayout),
			previousDay.Format(currency_helpers.CustomTimeLayout),
			currencyCodeBase,
			currencyCodeSecond,
		)

		req, err := http.NewRequestWithContext(cbrCtx, http.MethodGet, url, nil)
		if err != nil {
			err = errors.Wrap(err, "error in prepare request")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		client := http.DefaultClient

		exchangerResp, err := client.Do(req)
		if err != nil {
			err = errors.Wrap(err, "error in get new data")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		defer exchangerResp.Body.Close()

		timelineCurrencyRates := &currency_helpers.CurrencyTimelineRatesResponse{}
		err = json.NewDecoder(exchangerResp.Body).Decode(timelineCurrencyRates)
		if err != nil {
			err = errors.Wrap(err, "internal error in read JSON data")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if !timelineCurrencyRates.Success {
			err = errors.New("unsuccessful getting new rates")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		currencyRate = timelineCurrencyRates.ToResultTimelineRates(currencyCodeSecond)

		predictorCtx, cancel := context.WithTimeout(ctx, time.Minute*10)
		defer cancel()

		dataForPredictions, err := json.Marshal(currencyRate.Rates)
		if err != nil {
			err = errors.Wrap(err, "error in prepare data for predictions")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		req, err = http.NewRequestWithContext(
			predictorCtx,
			http.MethodPost,
			fmt.Sprintf(
				"https://stbuddy.xyz/predict",
			),
			bytes.NewBuffer(dataForPredictions),
		)
		req.Header.Set("Content-Type", "application/json")
		if err != nil {
			err = errors.Wrap(err, "error in prepare request")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		predictorResp, err := client.Do(req)
		if err != nil {
			err = errors.Wrap(err, "error in get predictions")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		defer predictorResp.Body.Close()
		var predictedRates []float64
		b, _ := io.ReadAll(predictorResp.Body)
		err = json.NewDecoder(bytes.NewBuffer(b)).Decode(&predictedRates)
		if err != nil {
			err = errors.Wrap(err, "error in reading predictor response")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		predictions := make(map[currency_helpers.CustomTime]float64, len(predictedRates))

		i := 0
		for _, rate := range predictedRates {
			t := time.Date(todayY, todayM, todayD+i, 0, 0, 0, 0, time.UTC)
			i += 1
			predictions[currency_helpers.CustomTime{Time: t}] = rate
		}
		currencyRate.Predictions = predictions

		cacheCtx, cancel = context.WithTimeout(ctx, s.cfg.CacheTimeout)
		defer cancel()
		err = s.redisCache.SaveTimestampRate(cacheCtx, currencyRate)
		if err != nil {
			log.Printf("error in save timestamp rate: %s", err.Error())
		}
	}

	timelineRates := make(map[currency_helpers.CustomTime]float64)
	for t, rate := range currencyRate.Rates {
		if t.Equal(startDate) || t.After(startDate) && t.Before(endDate) {
			timelineRates[t] = rate
		}
	}
	result := currency_helpers.CurrencyTimelineRate{
		Base:        currencyRate.Base,
		Second:      currencyRate.Second,
		Rates:       timelineRates,
		Predictions: currencyRate.Predictions,
		StartDate:   currency_helpers.CustomTime{Time: startDate},
		EndDate:     currency_helpers.CustomTime{Time: endDate},
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(result)
	if err != nil {
		err = errors.Wrap(err, "error in prepare response date")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
