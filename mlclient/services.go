package mlclient

import (
	"context"
	"net/http"
)

type CacheStorage interface {
	Get(ctx context.Context, key string, v interface{}) error
	Set(ctx context.Context, key string, v interface{}) error
}

type HttpClientMaker interface {
	MakeClient(ctx context.Context) (*http.Client, error)
}
