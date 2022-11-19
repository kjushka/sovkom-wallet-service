package main

import (
	"log"
	"net/http"
	"os"
	"wallet-service/internal/cache"
	"wallet-service/internal/database"
	"wallet-service/internal/migrations"

	"github.com/pkg/errors"
	"wallet-service/internal/config"
	"wallet-service/internal/http_service"
)

func main() {
	cfg, err := config.InitConfig()
	if err != nil {
		log.Fatal(errors.Wrap(err, "error in config initiating"))
	}

	db, err := database.InitDB(cfg)
	if err != nil {
		log.Fatal(errors.Wrap(err, "error in create database conn"))
	}
	defer db.Close()

	err = migrations.Migrate(db, cfg)
	if err != nil {
		log.Fatal(errors.Wrap(err, "error in migrate process"))
	}

	redisCache, err := cache.InitCache(cfg)
	if err != nil {
		log.Fatal(errors.Wrap(err, "error in cache initiating"))
	}

	router := http_service.InitRouter(db, redisCache, cfg)

	log.Println("service starting...")
	err = http.ListenAndServe(":8080", router)
	if err != nil {
		log.Println(errors.Wrap(err, "error in running service"))
		os.Exit(0)
	}
}
