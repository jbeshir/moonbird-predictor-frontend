package pbook

import (
	"context"
	"github.com/jbeshir/moonbird-predictor-frontend/data"
	"github.com/jbeshir/predictionbook-extractor/predictions"
	"math"
)

const cacheExamplesKey = "examples"
const storeExamplesKind = "ExamplePredictions"
const storeExamplesKey = "examples"

type Lister struct {
	Fetcher         predictions.HtmlFetcher
	CacheStore      CacheStore
	PersistentStore PersistentStore
}

func (l *Lister) GetExamples(ctx context.Context) (data.ExamplePredictions, error) {

	var examples data.ExamplePredictions
	err := l.CacheStore.Get(ctx, cacheExamplesKey, &examples)
	if err == nil {
		return examples, nil
	}
	err = nil

	err = l.PersistentStore.GetOpaque(ctx, storeExamplesKind, storeExamplesKey, &examples)
	if err != nil {
		return nil, err
	}

	l.CacheStore.Set(ctx, cacheExamplesKey, &examples)

	return examples, nil
}

func (l *Lister) UpdateExamples(ctx context.Context) (data.ExamplePredictions, error) {

	s := predictions.NewSource(l.Fetcher, "https://predictionbook.com")
	summaries, _, err := s.RetrievePredictionListPage(ctx, 1)
	if err != nil {
		return nil, err
	}

	responses, err := s.AllPredictionResponses(ctx, summaries)
	if err != nil {
		return nil, err
	}

	var examples data.ExamplePredictions
	for i := range summaries {
		example := data.ExamplePrediction{
			PredictionSummary: summaries[i],
		}
		for _, r := range responses {
			if r.Prediction == example.Id && !math.IsNaN(r.Confidence) {
				example.Assignments = append(example.Assignments, r.Confidence)
			}
		}
		examples = append(examples, example)
	}

	err = l.PersistentStore.SetOpaque(ctx, storeExamplesKind, storeExamplesKey, examples)
	if err != nil {
		return nil, err
	}

	l.CacheStore.Delete(ctx, cacheExamplesKey)

	return examples, nil
}
