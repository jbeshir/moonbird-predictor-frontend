package pbook

import (
	"context"
	"github.com/jbeshir/moonbird-predictor-frontend/ctxlogrus"
	"github.com/jbeshir/moonbird-predictor-frontend/data"
	"github.com/jbeshir/predictionbook-extractor/predictions"
	"github.com/pkg/errors"
	"math"
)

const cacheExamplesKey = "examples"
const storeExamplesKind = "ExamplePredictions"
const storeExamplesKey = "examples"

type Lister struct {
	PredictionSource PredictionSource
	CacheStore       CacheStore
	PersistentStore  PersistentStore
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
		return nil, errors.Wrap(err, "")
	}

	_ = l.CacheStore.Set(ctx, cacheExamplesKey, &examples)

	return examples, nil
}

func (l *Lister) UpdateExamples(ctx context.Context) (data.ExamplePredictions, error) {
	logger := ctxlogrus.Get(ctx)

	logger.Info("Retrieving first page of predictions from prediction source...")
	summaries, _, err := l.PredictionSource.RetrievePredictionListPage(ctx, 1)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	logger.Infof("Got %d predictions", len(summaries))

	logger.Info("Splitting unresolved predictions from resolved...")
	var unresolvedSummaries []*predictions.PredictionSummary
	for _, s := range summaries {
		if s.Outcome == predictions.Unknown {
			unresolvedSummaries = append(unresolvedSummaries, s)
		}
	}
	logger.Infof("Got %d unresolved predictions", len(unresolvedSummaries))

	logger.Info("Retrieving unresolved prediction responses...")
	_, responses, err := l.PredictionSource.AllPredictionResponses(ctx, unresolvedSummaries)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	logger.Info("Building and saving new example predictions list.")
	var examples data.ExamplePredictions
	for i := range unresolvedSummaries {
		example := data.ExamplePrediction{
			PredictionSummary: unresolvedSummaries[i],
		}
		for _, r := range responses {
			if r.Prediction == example.Id && !math.IsNaN(r.Confidence) {
				example.Assignments = append(example.Assignments, r.Confidence)
			}
		}
		examples = append(examples, example)
	}

	err = l.PersistentStore.SetOpaque(ctx, storeExamplesKind, storeExamplesKey, &examples)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	_ = l.CacheStore.Delete(ctx, cacheExamplesKey)

	return examples, nil
}
