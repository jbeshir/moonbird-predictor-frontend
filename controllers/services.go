package controllers

import (
	"context"
	"github.com/jbeshir/moonbird-predictor-frontend/data"
	"net/http"
	"time"
)

type ContextMaker interface {
	MakeContext(r *http.Request) (context.Context, error)
}

type PredictionMaker interface {
	Predict(ctx context.Context, predictions []float64) (p float64, err error)
}

type ExampleLister interface {
	GetExamples(ctx context.Context) (data.ExamplePredictions, error)
	UpdateExamples(ctx context.Context) (data.ExamplePredictions, error)
}

type ModelTrainer interface {
	Retrain(ctx context.Context, now time.Time) error
}
