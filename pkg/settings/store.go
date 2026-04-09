package settings

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("setting not found")

type Record struct {
	Key   string
	Value []byte
	Order int
}

type Store interface {
	GetByKey(ctx context.Context, key string) (*Record, error)
	UpdateOrCreate(ctx context.Context, record *Record) error
}

var store Store

func InitStore(s Store) {
	store = s
}
