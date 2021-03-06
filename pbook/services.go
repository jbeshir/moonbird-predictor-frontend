package pbook

import (
	"context"
	"github.com/jbeshir/moonbird-auth-frontend/data"
	"github.com/jbeshir/predictionbook-extractor/predictions"
)

type CacheStore interface {
	Get(ctx context.Context, key string, v interface{}) error
	Set(ctx context.Context, key string, v interface{}) error
	Delete(ctx context.Context, key string) error
}

type PersistentStore interface {
	Get(ctx context.Context, kind, key string, v interface{}) ([]data.Property, error)
	Set(ctx context.Context, kind, key string, properties []data.Property, v interface{}) error
}

type PredictionSource interface {
	RetrievePredictionListPage(context.Context, int64) ([]*predictions.PredictionSummary, *predictions.PredictionListPageInfo, error)
	AllPredictionResponses(context.Context, []*predictions.PredictionSummary) ([]*predictions.PredictionSummary, []*predictions.PredictionResponse, error)
}
