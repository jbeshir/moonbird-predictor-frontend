package controllers

import (
	"context"
	"github.com/jbeshir/moonbird-predictor-frontend/data"
	"net/http"
	"testing"
)

func newTestContextMaker(t *testing.T) *testContextMaker {
	return &testContextMaker{
		MakeContextFunc: func(r *http.Request) (context.Context, error) {
			t.Error("MakeContextFunc should not be called")
			return nil, nil
		},
	}
}

type testContextMaker struct {
	MakeContextFunc func(r *http.Request) (context.Context, error)
}

func (cm *testContextMaker) MakeContext(r *http.Request) (context.Context, error) {
	return cm.MakeContextFunc(r)
}

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
