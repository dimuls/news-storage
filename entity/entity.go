package entity

import (
	"errors"
	"time"
)

type News struct {
	ID     int64     `db:"id" json:"id"`
	Header string    `db:"header" json:"header"`
	Date   time.Time `db:"date" json:"date"`
}

var ErrNewsNotFound = errors.New("news not found")
