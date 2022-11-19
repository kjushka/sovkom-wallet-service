package database

import (
	"fmt"
	"time"
	"wallet-service/internal/config"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
)

func InitDB(cfg *config.Config) (*sqlx.DB, error) {
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser,
		cfg.DBPass,
		cfg.DBHost,
		cfg.DBPort,
		cfg.Database,
	)
	db, err := sqlx.Open("postgres", connStr)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't connect with database")
	}

	err = db.Ping()
	if err != nil {
		for err != nil {
			time.Sleep(time.Second * 2)
			err = db.Ping()
		}
	}

	return db, nil
}
