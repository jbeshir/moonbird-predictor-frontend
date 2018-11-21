package pbook

import (
	"context"
	"github.com/jbeshir/moonbird-predictor-frontend/data"
	"github.com/jbeshir/predictionbook-extractor/predictions"
	"google.golang.org/appengine/datastore"
)

const cacheExamplesKey = "examples"
const datastorePbLatestKey = "pb_latest"

type Lister struct {
	Fetcher predictions.HtmlFetcher
	CacheStorage CacheStorage
}

func (l *Lister) GetExamples(ctx context.Context) (*data.ExamplePredictions, error) {

	latest := new(data.ExamplePredictions)
	err := l.CacheStorage.Get(ctx, cacheExamplesKey, &latest)
	if err == nil {
		return latest, nil
	}
	err = nil

	k := datastore.NewKey(ctx, "PredictionBookLatest", datastorePbLatestKey, 0, nil)
	err = datastore.Get(ctx, k, latest)
	if err != nil {
		return nil, err
	}

	l.CacheStorage.Set(ctx, cacheExamplesKey, &latest)

	return latest, nil
}

func (l *Lister) UpdateExamples(ctx context.Context) (*data.ExamplePredictions, error) {

	s := predictions.NewSource(l.Fetcher, "https://predictionbook.com")
	summaries, _, err := s.RetrievePredictionListPage(ctx, 1)
	if err != nil {
		return nil, err
	}

	responses, err := s.AllPredictionResponses(ctx, summaries)
	if err != nil {
		return nil, err
	}

	latest := new(data.ExamplePredictions)
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

	l.CacheStorage.Delete(ctx, cacheExamplesKey)

	return latest, nil
}

func predictionBookCompact(latest *data.ExamplePredictions) {
	// We don't use any of this data for predicting, so discard it to shrink the size of our example data.
	for i := range latest.Responses {
		latest.Responses[i].User = ""
		latest.Responses[i].Comment = ""
	}
}