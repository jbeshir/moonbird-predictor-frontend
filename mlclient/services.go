package mlclient

import (
	"context"
	"github.com/jbeshir/predictionbook-extractor/predictions"
	"net/http"
	"time"
)

type CacheStorage interface {
	Get(ctx context.Context, key string, v interface{}) error
	Set(ctx context.Context, key string, v interface{}) error
}

type PersistentStore interface {
	GetOpaque(ctx context.Context, kind, key string, v interface{}) error
	SetOpaque(ctx context.Context, kind, key string, v interface{}) error
	Transact(ctx context.Context, f func(ctx context.Context) error) error
}

type FileStore interface {
	Load(ctx context.Context, path string) ([]byte, error)
	Save(ctx context.Context, path string, content []byte) error
}

type PredictionSource interface {
	AllPredictionsSince(ctx context.Context, t time.Time) ([]*predictions.PredictionSummary, error)
	AllPredictionResponses(context.Context, []*predictions.PredictionSummary) ([]*predictions.PredictionSummary, []*predictions.PredictionResponse, error)
}

type HttpClientMaker interface {
	MakeClient(ctx context.Context) (*http.Client, error)
}
