package pbook

import (
	"context"
)

type CacheStorage interface {
	Get(ctx context.Context, key string, v interface{}) error
	Set(ctx context.Context, key string, v interface{}) error
	Delete(ctx context.Context, key string) error
}