package service

import (
	"github.com/jmoiron/sqlx"
	"net/http"
	"wallet-service/internal/cache"
	"wallet-service/internal/config"

	"github.com/go-chi/chi/v5"
)

func InitRouter(db *sqlx.DB, redisCache cache.Cache, cfg *config.Config) http.Handler {
	s := NewService(db, redisCache, cfg)

	r := chi.NewRouter()
	initMiddlewares(r, s)
	initRoutes(r, s)

	return r
}
