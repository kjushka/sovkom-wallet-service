package http_service

import (
	"github.com/go-chi/chi/v5"
)

func initRoutes(r chi.Router, s Service) {
	r.Route("/currency", func(r chi.Router) {
		r.Get("/available", s.GetAvailableCurrencies)
		r.Post("/change-ban", s.ChangeCurrencyBanStatus)

		r.Get("/current-rate", s.GetCurrentCurrencyRate)
		r.Get("/time-series", s.GetTimelineCurrencyRate)
	})
}
