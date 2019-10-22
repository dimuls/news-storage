package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Boostport/migration"
	"github.com/Boostport/migration/driver/postgres"
	"github.com/dimuls/news-storage/entity"
	"github.com/gobuffalo/packr"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type Storage struct {
	uri string
	db  *sqlx.DB
}

func NewStorage(postgresURI string) (*Storage, error) {
	db, err := sqlx.Connect("postgres", postgresURI)
	if err != nil {
		return nil, errors.New("failed to connect: " + err.Error())
	}

	ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, errors.New("failed to ping: " + err.Error())
	}

	return &Storage{
		uri: postgresURI,
		db:  db,
	}, nil
}

//go:generate packr

const migrationsPath = "./migrations"

func (s *Storage) Migrate() error {
	packrSource := &migration.PackrMigrationSource{
		Box: packr.NewBox(migrationsPath),
	}

	d, err := postgres.New(s.uri)
	if err != nil {
		return errors.New("failed to create migration driver: " + err.Error())
	}

	_, err = migration.Migrate(d, packrSource, migration.Up, 0)
	if err != nil {
		return errors.New("failed to migrate: " + err.Error())
	}

	return nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) News(ctx context.Context, id int64) (n entity.News, err error) {
	err = s.db.QueryRowxContext(ctx, `
		SELECT * FROM news WHERE id = $1;
	`, id).StructScan(&n)
	if err == sql.ErrNoRows {
		err = entity.ErrNewsNotFound
	}
	return
}
