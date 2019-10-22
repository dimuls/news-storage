package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Boostport/migration"
	"github.com/Boostport/migration/driver/postgres"
	"github.com/dimuls/news-storage/entity"
	"github.com/gobuffalo/packr"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

func initStorage(t *testing.T) *Storage {
	s, err := NewStorage(os.Getenv("TEST_POSTGRES_URI"))
	if err != nil {
		t.Fatal("failed to create store: " + err.Error())
	}

	packrSource := &migration.PackrMigrationSource{
		Box: packr.NewBox(migrationsPath),
	}

	d, err := postgres.New(s.uri)
	if err != nil {
		t.Fatal("failed to create migration driver: " + err.Error())
	}

	_, err = migration.Migrate(d, packrSource, migration.Up, 0)
	if err != nil {
		t.Fatal("failed to migrate: " + err.Error())
	}

	return s
}

func cleanStorage(t *testing.T, s *Storage) {
	packrSource := &migration.PackrMigrationSource{
		Box: packr.NewBox(migrationsPath),
	}

	d, err := postgres.New(s.uri)
	if err != nil {
		t.Fatal("failed to create driver: " + err.Error())
	}

	_, err = migration.Migrate(d, packrSource, migration.Down, 0)
	if err != nil {
		t.Fatal("failed to migrate: " + err.Error())
	}
}

func TestStorage_News_notFound(t *testing.T) {
	s := initStorage(t)
	defer cleanStorage(t, s)

	_, err := s.db.Exec(`
		INSERT INTO news (id, header, date)
		VALUES (234, 'header-2', NOW())
	`)
	if !assert.NoError(t, err) {
		return
	}

	_, err = s.News(context.TODO(), 123)
	assert.Equal(t, entity.ErrNewsNotFound, err)
}

func TestStorage_News_success(t *testing.T) {
	s := initStorage(t)
	defer cleanStorage(t, s)

	testTime, _ := time.Parse("2006-01-02 15:04:05",
		"2006-01-02 15:04:05")

	wantN := entity.News{
		ID:     123,
		Header: "header",
		Date:   testTime,
	}

	_, err := s.db.Exec(`
		INSERT INTO news (id, header, date)
		VALUES ($1, $2, $3), (234, 'header-2', NOW())
	`, wantN.ID, wantN.Header, wantN.Date)
	if !assert.NoError(t, err) {
		return
	}

	gotN, err := s.News(context.TODO(), 123)
	if !assert.NoError(t, err) {
		return
	}

	if !assert.True(t, cmp.Equal(wantN, gotN)) {
		t.Log(cmp.Diff(wantN, gotN))
	}
}
