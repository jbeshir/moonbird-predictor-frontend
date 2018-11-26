package pbook

import (
	"context"
	"errors"
	"github.com/jbeshir/moonbird-predictor-frontend/data"
	"github.com/jbeshir/predictionbook-extractor/predictions"
	"reflect"
	"testing"
)

func TestLister_GetExamples_FromCache(t *testing.T) {
	ps := newTestPersistentStore(t)
	cs := newTestCacheStore(t)
	s := newTestPredictionSource(t)
	lister := &Lister{
		PredictionSource: s,
		CacheStore:       cs,
		PersistentStore:  ps,
	}

	example := data.ExamplePrediction{
		PredictionSummary: &predictions.PredictionSummary{
			Id: 7,
		},
	}

	cs.GetFunc = func(ctx context.Context, key string, v interface{}) error {
		if key != cacheExamplesKey {
			t.Errorf("Reading from wrong cache key; expected %s, was %s", cacheExamplesKey, key)
		}

		examples, valid := v.(*data.ExamplePredictions)
		if !valid {
			t.Errorf("Cache reading wrong type; expected *data.ExamplePredictions")
		} else {
			*examples = []data.ExamplePrediction{example}
		}

		return nil
	}

	c := context.Background()
	examples, err := lister.GetExamples(c)
	if err != nil {
		t.Errorf("Unexpected error from lister: %s", err)
	}
	if len(examples) != 1 {
		t.Errorf("Unexpected examples length, should have been %d, was %d", 1, len(examples))
	} else {
		if examples[0].Id != 7 {
			t.Errorf("Incorrect example data, prediction ID should have been %d, was %d", 7, examples[0].Id)
		}
	}
}

func TestLister_GetExamples_FromStore(t *testing.T) {
	ps := newTestPersistentStore(t)
	cs := newTestCacheStore(t)
	s := newTestPredictionSource(t)
	lister := &Lister{
		PredictionSource: s,
		CacheStore:       cs,
		PersistentStore:  ps,
	}

	example := data.ExamplePrediction{
		PredictionSummary: &predictions.PredictionSummary{
			Id: 7,
		},
	}
	testExamples := data.ExamplePredictions([]data.ExamplePrediction{example})

	setCallCount := 0
	cs.GetFunc = func(ctx context.Context, key string, v interface{}) error {
		return errors.New("nope")
	}
	cs.SetFunc = func(ctx context.Context, key string, v interface{}) error {
		if key != cacheExamplesKey {
			t.Errorf("Setting wrong cache key; expected %s, was %s", cacheExamplesKey, key)
		}

		examples, valid := v.(*data.ExamplePredictions)
		if !valid {
			t.Errorf("Cache setting wrong type; expected *data.ExamplePredictions")
		} else {
			if !reflect.DeepEqual(*examples, testExamples) {
				t.Error("Cache setting wrong value; doesn't match generated example set")
			}
		}

		setCallCount++
		return nil
	}
	ps.GetOpaqueFunc = func(ctx context.Context, kind, key string, v interface{}) error {
		if kind != storeExamplesKind {
			t.Errorf("Reading from wrong store kind; expected %s, was %s", storeExamplesKind, kind)
		}
		if key != storeExamplesKey {
			t.Errorf("Reading from wrong store key; expected %s, was %s", storeExamplesKey, key)
		}

		examples, valid := v.(*data.ExamplePredictions)
		if !valid {
			t.Errorf("Store reading wrong type; expected *data.ExamplePredictions")
		} else {
			*examples = testExamples
		}

		return nil
	}

	c := context.Background()
	examples, err := lister.GetExamples(c)
	if err != nil {
		t.Errorf("Unexpected error from lister: %s", err)
	}
	if len(examples) != 1 {
		t.Errorf("Unexpected examples length, should have been %d, was %d", 1, len(examples))
	} else {
		if examples[0].Id != 7 {
			t.Errorf("Incorrect example data, prediction ID should have been %d, was %d", 7, examples[0].Id)
		}
	}
}

func TestLister_GetExamples_FromStoreErr(t *testing.T) {
	ps := newTestPersistentStore(t)
	cs := newTestCacheStore(t)
	s := newTestPredictionSource(t)
	lister := &Lister{
		PredictionSource: s,
		CacheStore:       cs,
		PersistentStore:  ps,
	}

	testErr := errors.New("bluh")
	cs.GetFunc = func(ctx context.Context, key string, v interface{}) error {
		return errors.New("nope")
	}
	ps.GetOpaqueFunc = func(ctx context.Context, kind, key string, v interface{}) error {
		return testErr
	}

	c := context.Background()
	examples, err := lister.GetExamples(c)
	if examples != nil {
		t.Error("Expected nil result from lister, got non-nil result")
	}
	if err == nil {
		t.Error("Expected error from lister, got nil error")
	}
}
