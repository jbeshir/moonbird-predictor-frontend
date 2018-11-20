package controllers

import (
	"context"
	"github.com/jbeshir/predictionbook-extractor/predictions"
	"net/http"
)

type ContextMaker interface {
	MakeContext(r *http.Request) (context.Context, error)
}

type PredictionMaker interface {
	Predict(ctx context.Context, predictions []float64) (p float64, err error)
}

type LatestPredictions struct {
	Summaries []predictions.PredictionSummary
	Responses []predictions.PredictionResponse
}

type LatestPredictionLister interface {
	GetLatest(ctx context.Context) (*LatestPredictions, error)
}
