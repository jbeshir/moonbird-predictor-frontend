package controllers

import (
	"context"
	"github.com/jbeshir/moonbird-predictor-frontend/data"
	"net/http"
	"strconv"
	"strings"
)

type Index struct {
	PredictionMaker PredictionMaker
	ExampleLister   ExampleLister
}

type IndexInput struct {
	AssignmentsStr string
}

type IndexResult struct {
	AssignmentsStr string
	Prediction     *float64
	PredictionErr  error
	ExampleList    []data.ExamplePredictionResult
	ExampleListErr error
}

type WebIndexResponder interface {
	OnContextError(w http.ResponseWriter, err error)
	OnResult(w http.ResponseWriter, r *IndexResult)
}

func (c *Index) HandleFunc(cm ContextMaker, resp WebIndexResponder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, err := cm.MakeContext(r)
		if err != nil {
			resp.OnContextError(w, err)
			return
		}

		input := &IndexInput{AssignmentsStr: r.FormValue("assignments")}
		result := c.handle(ctx, input)
		resp.OnResult(w, result)
	}
}

func (c *Index) handle(ctx context.Context, input *IndexInput) *IndexResult {
	var prediction *float64
	var err error

	var assignments []float64
	assignmentsStr := input.AssignmentsStr
	assignmentStrs := strings.Split(assignmentsStr, ",")
	for _, assignmentStr := range assignmentStrs {
		if assignmentStr == "" {
			continue
		}

		var assignment float64
		assignment, err = strconv.ParseFloat(strings.TrimSpace(assignmentStr), 64)
		if err != nil {
			break
		}

		assignments = append(assignments, assignment)
	}

	if err == nil && len(assignments) > 0 {
		var p float64
		p, err = c.PredictionMaker.Predict(ctx, assignments)
		if err == nil {
			prediction = &p
		}
	}

	var exampleResults []data.ExamplePredictionResult
	examples, listErr := c.ExampleLister.GetExamples(ctx)
	if listErr == nil {
		for _, example := range examples {
			var exampleResult data.ExamplePredictionResult
			exampleResult.ExamplePrediction = example
			exampleResult.Result, exampleResult.ResultErr = c.PredictionMaker.Predict(ctx, example.Assignments)
			exampleResults = append(exampleResults, exampleResult)
		}
	}

	result := &IndexResult{
		AssignmentsStr: assignmentsStr,
		Prediction:     prediction,
		PredictionErr:  err,
		ExampleList:    exampleResults,
		ExampleListErr: listErr,
	}
	return result
}
