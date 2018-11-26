package pbook

import (
	"context"
	"errors"
	"github.com/jbeshir/moonbird-predictor-frontend/data"
	"github.com/jbeshir/predictionbook-extractor/predictions"
	"math"
	"reflect"
	"testing"
)

func TestLister_UpdateExamples(t *testing.T) {
	ps := newTestPersistentStore(t)
	cs := newTestCacheStore(t)
	s := newTestPredictionSource(t)
	lister := &Lister{
		PredictionSource: s,
		CacheStore:       cs,
		PersistentStore:  ps,
	}

	testSummaries := []*predictions.PredictionSummary{
		{Id: 7},
		{Id: 9},
	}
	s.RetrievePredictionListPageFunc = func(ctx context.Context, i int64) ([]*predictions.PredictionSummary, *predictions.PredictionListPageInfo, error) {
		if i != 1 {
			t.Errorf("Expected to retrieve page %d, actually retrieved %d", 1, i)
		}

		return testSummaries, nil, nil
	}

	testResponses := []*predictions.PredictionResponse{
		{
			Prediction: 7,
			Confidence: 0.6,
		},
		{
			Prediction: 7,
			Confidence: math.NaN(),
		},
		{
			Prediction: 9,
			Confidence: 0.3,
		},
		{
			Prediction: 9,
			Confidence: 0.1,
		},
	}
	s.AllPredictionResponsesFunc = func(ctx context.Context, summaries []*predictions.PredictionSummary) ([]*predictions.PredictionResponse, error) {
		if !reflect.DeepEqual(summaries, testSummaries) {
			t.Error("Expected to receive test summaries, received different summary slice")
		}
		return testResponses, nil
	}

	storeSetCallCount := 0
	var storeSetExamples data.ExamplePredictions
	ps.SetOpaqueFunc = func(ctx context.Context, kind, key string, v interface{}) error {
		if kind != storeExamplesKind {
			t.Errorf("Writing to wrong store kind; expected %s, was %s", storeExamplesKind, kind)
		}
		if key != storeExamplesKey {
			t.Errorf("Writing to wrong store key; expected %s, was %s", storeExamplesKey, key)
		}

		examples, valid := v.(*data.ExamplePredictions)
		if !valid {
			t.Errorf("Store writing wrong type; expected *data.ExamplePredictions")
		} else {
			if len(*examples) != 2 {
				t.Errorf("Store writing wrong number of examples; expected %d, was %d", 2, len(*examples))
			} else {
				if (*examples)[1].Id != 9 {
					t.Errorf("Store writing wrong second example ID; expected %d, was %d", 9, (*examples)[1].Id)
				}
				if len((*examples)[0].Assignments) != 1 {
					t.Errorf("Store writing wrong number of assignments for first example; expected %d, was %d", 1, len((*examples)[0].Assignments))
				} else {
					if (*examples)[0].Assignments[0] != 0.6 {
						t.Errorf("Store writing wrong first example assignment; expected %g, was %g", 0.6, (*examples)[0].Assignments[0])
					}
				}
				if len((*examples)[1].Assignments) != 2 {
					t.Errorf("Store writing wrong number of assignments for second example; expected %d, was %d", 2, len((*examples)[1].Assignments))
				}
			}
			storeSetExamples = *examples
		}

		storeSetCallCount++
		return nil
	}

	deleteCallCount := 0
	cs.DeleteFunc = func(ctx context.Context, key string) error {
		if key != cacheExamplesKey {
			t.Errorf("Deleting wrong cache key; expected %s, was %s", cacheExamplesKey, key)
		}

		deleteCallCount++
		return nil
	}

	c := context.Background()
	result, err := lister.UpdateExamples(c)
	if err != nil {
		t.Errorf("Unexpected error returned from lister: %s", err)
	}
	if !reflect.DeepEqual(result, storeSetExamples) {
		t.Error("Expected returned examples to match those set in the store")
	}

	if storeSetCallCount != 1 {
		t.Errorf("Expected SetOpaque to be called %d times, was called %d times", 1, storeSetCallCount)
	}
	if deleteCallCount != 1 {
		t.Errorf("Expected Delete to be called %d times, was called %d times", 1, deleteCallCount)
	}
}

func TestLister_UpdateExamples_SummariesErr(t *testing.T) {
	ps := newTestPersistentStore(t)
	cs := newTestCacheStore(t)
	s := newTestPredictionSource(t)
	lister := &Lister{
		PredictionSource: s,
		CacheStore:       cs,
		PersistentStore:  ps,
	}

	s.RetrievePredictionListPageFunc = func(ctx context.Context, i int64) ([]*predictions.PredictionSummary, *predictions.PredictionListPageInfo, error) {
		return nil, nil, errors.New("nope")
	}

	c := context.Background()
	result, err := lister.UpdateExamples(c)
	if result != nil {
		t.Error("Expected nil results from lister, got non-nil results")
	}
	if err == nil {
		t.Error("Expected error return from lister, got nil error")
	}
}

func TestLister_UpdateExamples_ResponsesErr(t *testing.T) {
	ps := newTestPersistentStore(t)
	cs := newTestCacheStore(t)
	s := newTestPredictionSource(t)
	lister := &Lister{
		PredictionSource: s,
		CacheStore:       cs,
		PersistentStore:  ps,
	}

	testSummaries := []*predictions.PredictionSummary{
		{Id: 7},
		{Id: 9},
	}
	s.RetrievePredictionListPageFunc = func(ctx context.Context, i int64) ([]*predictions.PredictionSummary, *predictions.PredictionListPageInfo, error) {
		return testSummaries, nil, nil
	}

	s.AllPredictionResponsesFunc = func(ctx context.Context, summaries []*predictions.PredictionSummary) ([]*predictions.PredictionResponse, error) {
		return nil, errors.New("nope")
	}

	c := context.Background()
	result, err := lister.UpdateExamples(c)
	if result != nil {
		t.Error("Expected nil results from lister, got non-nil results")
	}
	if err == nil {
		t.Error("Expected error return from lister, got nil error")
	}
}

func TestLister_UpdateExamples_StoreErr(t *testing.T) {
	ps := newTestPersistentStore(t)
	cs := newTestCacheStore(t)
	s := newTestPredictionSource(t)
	lister := &Lister{
		PredictionSource: s,
		CacheStore:       cs,
		PersistentStore:  ps,
	}

	testSummaries := []*predictions.PredictionSummary{
		{Id: 7},
		{Id: 9},
	}
	s.RetrievePredictionListPageFunc = func(ctx context.Context, i int64) ([]*predictions.PredictionSummary, *predictions.PredictionListPageInfo, error) {
		return testSummaries, nil, nil
	}

	testResponses := []*predictions.PredictionResponse{
		{
			Prediction: 7,
			Confidence: 0.6,
		},
		{
			Prediction: 7,
			Confidence: math.NaN(),
		},
		{
			Prediction: 9,
			Confidence: 0.3,
		},
		{
			Prediction: 9,
			Confidence: 0.1,
		},
	}
	s.AllPredictionResponsesFunc = func(ctx context.Context, summaries []*predictions.PredictionSummary) ([]*predictions.PredictionResponse, error) {
		return testResponses, nil
	}

	ps.SetOpaqueFunc = func(ctx context.Context, kind, key string, v interface{}) error {
		return errors.New("nope")
	}

	c := context.Background()
	result, err := lister.UpdateExamples(c)
	if result != nil {
		t.Error("Expected nil results from lister, got non-nil results")
	}
	if err == nil {
		t.Error("Expected error return from lister, got nil error")
	}
}
