package main

import (
	"context"
	"github.com/jbeshir/predictionbook-extractor/htmlfetcher"
	"github.com/jbeshir/predictionbook-extractor/predictions"
	"golang.org/x/time/rate"
	"google.golang.org/appengine/memcache"
	"math"
)

var pbFetcher = htmlfetcher.NewFetcher(rate.NewLimiter(1, 2), 2)
const memcachePbLatestKey = "pb_latest"

type PredictionBookLatest struct {
	Summaries []*predictions.PredictionSummary
	Responses []*predictions.PredictionResponse
}

func getLatestPredictionBook(ctx context.Context) (*PredictionBookLatest, error) {

	latest := new(PredictionBookLatest)
	_, err := memcache.JSON.Get(ctx, memcachePbLatestKey, &latest)
	if err == nil {
		predictionBookJsonPostprocess(latest)
		return latest, nil
	}
	err = nil

	s := predictions.NewSource(pbFetcher, "https://predictionbook.com")
	summaries, _, err := s.RetrievePredictionListPage(ctx, 1)
	if err != nil {
		return nil, err
	}

	responses, err := s.AllPredictionResponses(ctx, summaries)
	if err != nil {
		return nil, err
	}

	latest.Summaries = summaries
	latest.Responses = responses

	cacheItem := &memcache.Item{
		Key:    memcachePbLatestKey,
		Object: &latest,
	}

	predictionBookJsonPreprocess(latest)
	memcache.JSON.Set(ctx, cacheItem)
	predictionBookJsonPostprocess(latest)

	return latest, nil
}

func predictionBookJsonPostprocess(latest *PredictionBookLatest) {
	for _, r := range latest.Responses {
		if r.Confidence == -1 {
			r.Confidence = math.NaN()
		}
	}
}

func predictionBookJsonPreprocess(latest *PredictionBookLatest) {
	for _, r := range latest.Responses {
		if math.IsNaN(r.Confidence) {
			r.Confidence = -1
		}
	}
}