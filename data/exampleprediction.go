package data

import "github.com/jbeshir/predictionbook-extractor/predictions"

type ExamplePredictions []ExamplePrediction

type ExamplePrediction struct {
	*predictions.PredictionSummary
	Assignments []float64
}

type ExamplePredictionResult struct {
	ExamplePrediction
	Result    float64
	ResultErr error
}
