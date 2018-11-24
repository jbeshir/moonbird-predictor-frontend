package pbook

import (
	"context"
	"github.com/jbeshir/predictionbook-extractor/predictions"
	"testing"
)

type testCacheStore struct {
	GetFunc func(ctx context.Context, key string, v interface{}) error
	SetFunc func(ctx context.Context, key string, v interface{}) error
	DeleteFunc func(ctx context.Context, key string) error
}

func newTestCacheStore(t *testing.T) *testCacheStore {
	return &testCacheStore{
		GetFunc: func(ctx context.Context, key string, v interface{}) error {
			t.Error("Get should not be called")
			return nil
		},
		SetFunc: func(ctx context.Context, key string, v interface{}) error {
			t.Error("Set should not be called")
			return nil
		},
		DeleteFunc: func(ctx context.Context, key string) error {
			t.Error("Delete should not be called")
			return nil
		},
	}
}

func (cs *testCacheStore) Get(ctx context.Context, key string, v interface{}) error {
	return cs.GetFunc(ctx, key, v)
}

func (cs *testCacheStore) Set(ctx context.Context, key string, v interface{}) error {
	return cs.SetFunc(ctx, key, v)
}

func (cs *testCacheStore) Delete(ctx context.Context, key string) error {
	return cs.DeleteFunc(ctx, key)
}

type testPersistentStore struct {
	GetOpaqueFunc func(ctx context.Context, kind, key string, v interface{}) error
	SetOpaqueFunc func(ctx context.Context, kind, key string, v interface{}) error
}

func newTestPersistentStore(t *testing.T) *testPersistentStore {
	return &testPersistentStore{
		GetOpaqueFunc: func(ctx context.Context, kind, key string, v interface{}) error {
			t.Error("GetOpaque should not be called")
			return nil
		},
		SetOpaqueFunc: func(ctx context.Context, kind, key string, v interface{}) error {
			t.Error("SetOpaque should not be called")
			return nil
		},
	}
}

func (ps *testPersistentStore) GetOpaque(ctx context.Context, kind, key string, v interface{}) error {
	return ps.GetOpaqueFunc(ctx, kind, key, v)
}

func (ps *testPersistentStore) SetOpaque(ctx context.Context, kind, key string, v interface{}) error {
	return ps.SetOpaqueFunc(ctx, kind, key, v)
}

type testPredictionSource struct {
	RetrievePredictionListPageFunc func(context.Context, int64) ([]*predictions.PredictionSummary, *predictions.PredictionListPageInfo, error)
	AllPredictionResponsesFunc func(context.Context, []*predictions.PredictionSummary) ([]*predictions.PredictionResponse, error)
}

func newTestPredictionSource(t *testing.T) *testPredictionSource {
	return &testPredictionSource{
		RetrievePredictionListPageFunc: func(i context.Context, i2 int64) ([]*predictions.PredictionSummary, *predictions.PredictionListPageInfo, error) {
			t.Error("RetrievePredictionListPage should not be called")
			return nil, nil, nil
		},
		AllPredictionResponsesFunc: func(i context.Context, summaries []*predictions.PredictionSummary) ([]*predictions.PredictionResponse, error) {
			t.Error("AllPredictionResponses should not be called")
			return nil, nil
		},
	}
}

func (ps *testPredictionSource) RetrievePredictionListPage(ctx context.Context, i int64) ([]*predictions.PredictionSummary, *predictions.PredictionListPageInfo, error) {
	return ps.RetrievePredictionListPageFunc(ctx, i)
}

func (ps *testPredictionSource) AllPredictionResponses(ctx context.Context, summaries []*predictions.PredictionSummary) ([]*predictions.PredictionResponse, error) {
	return ps.AllPredictionResponsesFunc(ctx, summaries)
}