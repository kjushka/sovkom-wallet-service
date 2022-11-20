package currency_helpers

import (
	"fmt"
	"strings"
)

type Currency int

type CurrencyCode string

func (c CurrencyCode) String() string {
	return string(c)
}

func (c *CurrencyCode) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), "\"")
	*c = CurrencyCode(s)
	return
}

func (c CurrencyCode) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", c.String())), nil
}

func (c *CurrencyCode) IsSet() bool {
	return *c != ""
}

const (
	CurrentTimeRateCollection string = "rate:collection"
	AvailableCurrencies       string = "available"
	TimeCollection            string = "time:collection"
)

type CurrencyRatesResponse struct {
	Success bool `json:"success"`
	*CurrencyRates
}

type CurrencyRates struct {
	Base  CurrencyCode             `json:"base"`
	Rates map[CurrencyCode]float64 `json:"rates"`
	Date  CustomTime               `json:"date"`
}

func (cr CurrencyRates) ToResultRate(currencyCode CurrencyCode) *CurrencyRate {
	return &CurrencyRate{
		Base:   cr.Base,
		Second: currencyCode,
		Rate:   cr.Rates[currencyCode],
		Date:   cr.Date,
	}
}

type CurrencyTimelineRatesResponse struct {
	Success bool `json:"success"`
	*CurrencyTimelineRates
}

type CurrencyTimelineRates struct {
	Base      CurrencyCode                            `json:"base"`
	Rates     map[CustomTime]map[CurrencyCode]float64 `json:"rates"`
	StartDate CustomTime                              `json:"start_date"`
	EndDate   CustomTime                              `json:"end_date"`
}

type CurrencyTimelineRate struct {
	Base        CurrencyCode           `json:"base"`
	Second      CurrencyCode           `json:"second"`
	Rates       map[CustomTime]float64 `json:"rates"`
	Predictions map[CustomTime]float64 `json:"predictions,omitempty"`
	StartDate   CustomTime             `json:"startDate"`
	EndDate     CustomTime             `json:"endDate"`
}

func (cr CurrencyTimelineRates) ToResultTimelineRates(currencyCode CurrencyCode) *CurrencyTimelineRate {
	rates := make(map[CustomTime]float64, len(cr.Rates))
	for periodTime, rate := range cr.Rates {
		rates[periodTime] = rate[currencyCode]
	}

	return &CurrencyTimelineRate{
		Base:      cr.Base,
		Second:    currencyCode,
		Rates:     rates,
		StartDate: cr.StartDate,
		EndDate:   cr.EndDate,
	}
}

type CurrencyRate struct {
	Base   CurrencyCode `json:"base"`
	Second CurrencyCode `json:"second"`
	Rate   float64      `json:"rate"`
	Date   CustomTime   `json:"date"`
}

type CurrencyWithBanStatus struct {
	Currency CurrencyCode `json:"currency"`
	Banned   bool         `json:"banned"`
}
