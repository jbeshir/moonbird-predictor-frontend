package data

import "github.com/jbeshir/predictionbook-extractor/predictions"

type ExamplePredictions struct {
	Summaries []predictions.PredictionSummary
	Responses []predictions.PredictionResponse
}