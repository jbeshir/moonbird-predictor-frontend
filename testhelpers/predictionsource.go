package testhelpers

import (
	"context"
	"github.com/jbeshir/predictionbook-extractor/predictions"
	"testing"
	"time"
)

type PredictionSource struct {
	RetrievePredictionListPageFunc func(context.Context, int64) ([]*predictions.PredictionSummary, *predictions.PredictionListPageInfo, error)
	AllPredictionsSinceFunc        func(context context.Context, t time.Time) ([]*predictions.PredictionSummary, error)
	AllPredictionResponsesFunc     func(context.Context, []*predictions.PredictionSummary) ([]*predictions.PredictionSummary, []*predictions.PredictionResponse, error)
}

func NewPredictionSource(t *testing.T) *PredictionSource {
	return &PredictionSource{
		RetrievePredictionListPageFunc: func(i context.Context, i2 int64) ([]*predictions.PredictionSummary, *predictions.PredictionListPageInfo, error) {
			t.Error("RetrievePredictionListPage should not be called")
			return nil, nil, nil
		},
		AllPredictionsSinceFunc: func(context context.Context, _ time.Time) ([]*predictions.PredictionSummary, error) {
			t.Error("AllPredictionsSince should not be called")
			return nil, nil
		},
		AllPredictionResponsesFunc: func(i context.Context, summaries []*predictions.PredictionSummary) ([]*predictions.PredictionSummary, []*predictions.PredictionResponse, error) {
			t.Error("AllPredictionResponses should not be called")
			return nil, nil, nil
		},
	}
}

func (ps *PredictionSource) RetrievePredictionListPage(ctx context.Context, i int64) ([]*predictions.PredictionSummary, *predictions.PredictionListPageInfo, error) {
	return ps.RetrievePredictionListPageFunc(ctx, i)
}

func (ps *PredictionSource) AllPredictionsSince(context context.Context, t time.Time) ([]*predictions.PredictionSummary, error) {
	return ps.AllPredictionsSinceFunc(context, t)
}

func (ps *PredictionSource) AllPredictionResponses(ctx context.Context, summaries []*predictions.PredictionSummary) ([]*predictions.PredictionSummary, []*predictions.PredictionResponse, error) {
	return ps.AllPredictionResponsesFunc(ctx, summaries)
}
