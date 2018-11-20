package controllers

import (
	"context"
	"math"
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
	ExampleList    string
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

	var exampleList strings.Builder
	latest, listErr := c.ExampleLister.GetExamples(ctx)
	if listErr == nil {
		for i := range latest.Summaries {
			summary := &latest.Summaries[i]
			var assignments []float64
			for _, r := range latest.Responses {
				if r.Prediction != summary.Id {
					continue
				}
				if math.IsNaN(r.Confidence) {
					continue
				}

				assignments = append(assignments, r.Confidence)
			}

			exampleList.WriteString(strconv.FormatInt(summary.Id, 10))
			examplePrediction, err := c.PredictionMaker.Predict(ctx, assignments)
			exampleList.WriteString(":")
			if err == nil {
				exampleList.WriteString(strconv.FormatFloat(examplePrediction, 'g', 4, 64))
			} else {
				exampleList.WriteString(err.Error())
			}
			exampleList.WriteString(" ")
		}
	}

	result := &IndexResult{
		AssignmentsStr: assignmentsStr,
		Prediction:     prediction,
		PredictionErr:  err,
		ExampleList:    exampleList.String(),
		ExampleListErr: listErr,
	}
	return result
}
