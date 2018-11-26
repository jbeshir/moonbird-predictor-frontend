package pbook

import (
	"context"
	"github.com/jbeshir/predictionbook-extractor/predictions"
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

type PredictionSource interface {
	RetrievePredictionListPage(context.Context, int64) ([]*predictions.PredictionSummary, *predictions.PredictionListPageInfo, error)
	AllPredictionResponses(context.Context, []*predictions.PredictionSummary) ([]*predictions.PredictionResponse, error)
}
