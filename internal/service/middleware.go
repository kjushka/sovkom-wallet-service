package service

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func initMiddlewares(r chi.Router, s Service) {
	r.Use(
		middleware.Logger,
		func(handler http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, PUT")
				w.Header().Set("Access-Control-Allow-Headers", "*")
				handler.ServeHTTP(w, r)
			})
		},
	)
}
