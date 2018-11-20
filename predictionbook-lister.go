package main

import (
	"context"
	"github.com/jbeshir/moonbird-predictor-frontend/controllers"
	"github.com/jbeshir/predictionbook-extractor/predictions"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/memcache"
	"math"
)

const memcachePbLatestKey = "pb_latest"
const datastorePbLatestKey = "pb_latest"

type PredictionBookLister struct {
	Fetcher predictions.HtmlFetcher
}

func (_ *PredictionBookLister) GetExamples(ctx context.Context) (*controllers.ExamplePredictions, error) {

	latest := new(controllers.ExamplePredictions)
	_, err := memcache.JSON.Get(ctx, memcachePbLatestKey, &latest)
	if err == nil {
		predictionBookJsonPostprocess(latest)
		return latest, nil
	}
	err = nil

	k := datastore.NewKey(ctx, "PredictionBookLatest", datastorePbLatestKey, 0, nil)
	err = datastore.Get(ctx, k, latest)
	if err != nil {
		return nil, err
	}

	cacheItem := &memcache.Item{
		Key:    memcachePbLatestKey,
		Object: &latest,
	}
	predictionBookJsonPreprocess(latest)
	memcache.JSON.Set(ctx, cacheItem)
	predictionBookJsonPostprocess(latest)

	return latest, nil
}

func (l *PredictionBookLister) UpdateExamples(ctx context.Context) (*controllers.ExamplePredictions, error) {

	s := predictions.NewSource(l.Fetcher, "https://predictionbook.com")
	summaries, _, err := s.RetrievePredictionListPage(ctx, 1)
	if err != nil {
		return nil, err
	}

	responses, err := s.AllPredictionResponses(ctx, summaries)
	if err != nil {
		return nil, err
	}

	latest := new(controllers.ExamplePredictions)
	latest.Summaries = make([]predictions.PredictionSummary, len(summaries))
	for i := range summaries {
		latest.Summaries[i] = *summaries[i]
	}
	latest.Responses = make([]predictions.PredictionResponse, len(responses))
	for i := range responses {
		latest.Responses[i] = *responses[i]
	}
	predictionBookCompact(latest)

	k := datastore.NewKey(ctx, "PredictionBookLatest", datastorePbLatestKey, 0, nil)
	_, err = datastore.Put(ctx, k, latest)
	if err != nil {
		return nil, err
	}

	memcache.Delete(ctx, memcachePbLatestKey)

	return latest, nil
}

func predictionBookJsonPostprocess(latest *controllers.ExamplePredictions) {
	for i := range latest.Responses {
		if latest.Responses[i].Confidence == -1 {
			latest.Responses[i].Confidence = math.NaN()
		}
	}
}

func predictionBookJsonPreprocess(latest *controllers.ExamplePredictions) {
	for i := range latest.Responses {
		if math.IsNaN(latest.Responses[i].Confidence) {
			latest.Responses[i].Confidence = -1
		}
	}
}

func predictionBookCompact(latest *controllers.ExamplePredictions) {
	// We don't use any of this data for predicting, so discard it to shrink the entity size.
	for i := range latest.Responses {
		latest.Responses[i].User = ""
		latest.Responses[i].Comment = ""
	}
}
