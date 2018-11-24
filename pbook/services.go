package pbook

import (
	"context"
)

type CacheStore interface {
	Get(ctx context.Context, key string, v interface{}) error
	Set(ctx context.Context, key string, v interface{}) error
	Delete(ctx context.Context, key string) error
}

type PersistentStore interface {
	GetOpaque(ctx context.Context, kind, key string, v interface{}) error
	SetOpaque(ctx context.Context, kind, key string, v interface{}) error
}
