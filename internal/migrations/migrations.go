package migrations

import (
	"wallet-service/internal/config"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

func Migrate(db *sqlx.DB, cfg *config.Config) error {
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{
		DatabaseName: cfg.Database,
	})
	if err != nil {
		return errors.Wrap(err, "error to define driver")
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://./migrations",
		"postgres", driver,
	)
	if err != nil {
		return errors.Wrap(err, "error in create migration client")
	}
	err = m.Up()
	if err != nil {
		downErr := m.Down()
		if downErr != nil {
			return errors.Wrap(err, "error in up and down migration")
		}
		return errors.Wrap(err, "error in up migration")
	}

	return nil
}
