package controllers

import (
	"context"
	"github.com/jbeshir/moonbird-predictor-frontend/data"
	"testing"
	"time"
)

func newTestExamplesLister(t *testing.T) *testExamplesLister {
	return &testExamplesLister{
		GetExamplesFunc: func(ctx context.Context) (data.ExamplePredictions, error) {
			t.Error("GetExamplesFunc should not be called")
			return nil, nil
		},
		UpdateExamplesFunc: func(ctx context.Context) (data.ExamplePredictions, error) {
			t.Error("UpdateExamplesFunc should not be called")
			return nil, nil
		},
	}
}

type testExamplesLister struct {
	GetExamplesFunc    func(ctx context.Context) (data.ExamplePredictions, error)
	UpdateExamplesFunc func(ctx context.Context) (data.ExamplePredictions, error)
}

func (l *testExamplesLister) GetExamples(ctx context.Context) (data.ExamplePredictions, error) {
	return l.GetExamplesFunc(ctx)
}

func (l *testExamplesLister) UpdateExamples(ctx context.Context) (data.ExamplePredictions, error) {
	return l.UpdateExamplesFunc(ctx)
}

func newTestPredictionMaker(t *testing.T) *testPredictionMaker {
	return &testPredictionMaker{
		PredictFunc: func(ctx context.Context, predictions []float64) (p float64, err error) {
			t.Error("Predict should not be called")
			return 0, nil
		},
	}
}

type testPredictionMaker struct {
	PredictFunc func(ctx context.Context, predictions []float64) (p float64, err error)
}

func (pm *testPredictionMaker) Predict(ctx context.Context, predictions []float64) (p float64, err error) {
	return pm.PredictFunc(ctx, predictions)
}

func newTestModelTrainer(t *testing.T) *testModelTrainer {
	return &testModelTrainer{
		RetrainFunc: func(ctx context.Context, now time.Time) error {
			t.Error("RetrainFunc should not be called")
			return nil
		},
	}
}

type testModelTrainer struct {
	RetrainFunc func(ctx context.Context, now time.Time) error
}

func (tr *testModelTrainer) Retrain(ctx context.Context, now time.Time) error {
	return tr.RetrainFunc(ctx, now)
}
