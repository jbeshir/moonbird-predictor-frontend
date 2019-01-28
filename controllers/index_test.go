package controllers

import (
	"context"
	"errors"
	"github.com/jbeshir/moonbird-predictor-frontend/data"
	predictions2 "github.com/jbeshir/predictionbook-extractor/predictions"
	"net/http"
	"net/url"
	"reflect"
	"testing"
)

func TestIndex_HandleFunc_NoExamples_NoAssignments(t *testing.T) {
	t.Parallel()

	var createdContext context.Context

	l := newTestExamplesLister(t)
	pm := newTestPredictionMaker(t)

	calledOnResult := false
	r := newTestWebIndexResponder(t)
	r.OnResultFunc = func(w http.ResponseWriter, result *IndexResult) {
		calledOnResult = true
		if result.AssignmentsStr != "" {
			t.Errorf("Result AssignmentStr should be '%s', was '%s'", "", result.AssignmentsStr)
		}
		if len(result.ExampleList) != 0 {
			t.Errorf("Result ExampleList should be empty, was not")
		}
		if result.ExampleListErr != nil {
			t.Errorf("Result ExampleListErr should be nil, was %s", result.ExampleListErr)
		}
		if result.Prediction != nil {
			t.Errorf("Result Prediction should be nil, was %f", *result.Prediction)
		}
		if result.PredictionErr != nil {
			t.Errorf("Result PredictionErr should be nil, was %s", result.PredictionErr)
		}
	}
	cm := newTestContextMaker(t)
	cm.MakeContextFunc = func(r *http.Request) (i context.Context, e error) {
		createdContext = context.Background()
		return createdContext, nil
	}

	var calledGetExamples bool
	l.GetExamplesFunc = func(ctx context.Context) (predictions data.ExamplePredictions, e error) {
		if ctx == nil {
			t.Error("Got nil context, expected non-nil context")
		}
		if ctx != createdContext {
			t.Error("Got context that didn't match one we created")
		}
		calledGetExamples = true
		return nil, nil
	}

	c := &Index{
		ExampleLister:   l,
		PredictionMaker: pm,
	}
	handler := c.HandleFunc(cm, r)
	handler(nil, &http.Request{})

	if !calledGetExamples {
		t.Error("Expected get examples to be called, was not called")
	}
	if !calledOnResult {
		t.Error("Expected responder's OnResult method to be called, was not called")
	}
}

func TestIndex_HandleFunc_NoExamples_Assignments(t *testing.T) {
	t.Parallel()

	var createdContext context.Context

	l := newTestExamplesLister(t)
	pm := newTestPredictionMaker(t)

	calledOnResult := false
	r := newTestWebIndexResponder(t)
	r.OnResultFunc = func(w http.ResponseWriter, result *IndexResult) {
		calledOnResult = true
		if result.AssignmentsStr != "0.1,0.2" {
			t.Errorf("Result AssignmentStr should be '%s', was '%s'", "0.1,0.2", result.AssignmentsStr)
		}
		if len(result.ExampleList) != 0 {
			t.Errorf("Result ExampleList should be empty, was not")
		}
		if result.ExampleListErr != nil {
			t.Errorf("Result ExampleListErr should be nil, was %s", result.ExampleListErr)
		}
		if *result.Prediction != 0.17 {
			t.Errorf("Result Prediction should be 0.17, was %f", *result.Prediction)
		}
		if result.PredictionErr != nil {
			t.Errorf("Result PredictionErr should be nil, was %s", result.PredictionErr)
		}
	}
	cm := newTestContextMaker(t)
	cm.MakeContextFunc = func(r *http.Request) (i context.Context, e error) {
		createdContext = context.Background()
		return createdContext, nil
	}

	l.GetExamplesFunc = func(ctx context.Context) (predictions data.ExamplePredictions, e error) {
		return nil, nil
	}

	var calledPredict bool
	pm.PredictFunc = func(ctx context.Context, predictions []float64) (p float64, err error) {
		calledPredict = true
		if ctx == nil {
			t.Error("Got nil context, expected non-nil context")
		}
		if ctx != createdContext {
			t.Error("Got context that didn't match one we created")
		}
		if !reflect.DeepEqual(predictions, []float64{0.1, 0.2}) {
			t.Error("Unexpected prediction values")
		}
		return 0.17, nil
	}

	c := &Index{
		ExampleLister:   l,
		PredictionMaker: pm,
	}
	handler := c.HandleFunc(cm, r)
	formValues := make(url.Values)
	formValues.Add("assignments", "0.1,0.2")
	handler(nil, &http.Request{Form: formValues})

	if !calledPredict {
		t.Error("Expected get examples to be called, was not called")
	}
	if !calledOnResult {
		t.Error("Expected responder's OnResult method to be called, was not called")
	}
}

func TestIndex_HandleFunc_NoExamples_JunkAssignments(t *testing.T) {
	t.Parallel()

	var createdContext context.Context

	l := newTestExamplesLister(t)
	pm := newTestPredictionMaker(t)

	calledOnResult := false
	r := newTestWebIndexResponder(t)
	r.OnResultFunc = func(w http.ResponseWriter, result *IndexResult) {
		calledOnResult = true
		if result.AssignmentsStr != "bluh" {
			t.Errorf("Result AssignmentStr should be '%s', was '%s'", "bluh", result.AssignmentsStr)
		}
		if len(result.ExampleList) != 0 {
			t.Errorf("Result ExampleList should be empty, was not")
		}
		if result.ExampleListErr != nil {
			t.Errorf("Result ExampleListErr should be nil, was %s", result.ExampleListErr)
		}
		if result.Prediction != nil {
			t.Errorf("Result Prediction should be nil, was %f", *result.Prediction)
		}
		if result.PredictionErr == nil {
			t.Errorf("Result PredictionErr should be non-nil, was nil")
		}
	}
	cm := newTestContextMaker(t)
	cm.MakeContextFunc = func(r *http.Request) (i context.Context, e error) {
		createdContext = context.Background()
		return createdContext, nil
	}

	var calledGetExamples bool
	l.GetExamplesFunc = func(ctx context.Context) (predictions data.ExamplePredictions, e error) {
		if ctx == nil {
			t.Error("Got nil context, expected non-nil context")
		}
		if ctx != createdContext {
			t.Error("Got context that didn't match one we created")
		}
		calledGetExamples = true
		return nil, nil
	}

	c := &Index{
		ExampleLister:   l,
		PredictionMaker: pm,
	}
	handler := c.HandleFunc(cm, r)
	formValues := make(url.Values)
	formValues.Add("assignments", "bluh")
	handler(nil, &http.Request{Form: formValues})

	if !calledGetExamples {
		t.Error("Expected get examples to be called, was not called")
	}
	if !calledOnResult {
		t.Error("Expected responder's OnResult method to be called, was not called")
	}
}

func TestIndex_HandleFunc_NoExamples_Assignments_PredictErr(t *testing.T) {
	t.Parallel()

	var createdContext context.Context

	l := newTestExamplesLister(t)
	pm := newTestPredictionMaker(t)

	calledOnResult := false
	r := newTestWebIndexResponder(t)
	r.OnResultFunc = func(w http.ResponseWriter, result *IndexResult) {
		calledOnResult = true
		if result.AssignmentsStr != "0.1,0.2" {
			t.Errorf("Result AssignmentStr should be '%s', was '%s'", "0.1,0.2", result.AssignmentsStr)
		}
		if len(result.ExampleList) != 0 {
			t.Errorf("Result ExampleList should be empty, was not")
		}
		if result.ExampleListErr != nil {
			t.Errorf("Result ExampleListErr should be nil, was %s", result.ExampleListErr)
		}
		if result.Prediction != nil {
			t.Errorf("Result Prediction should be nil, was %f", *result.Prediction)
		}
		if result.PredictionErr == nil {
			t.Errorf("Result PredictionErr should be non-nil, was nil")
		}
	}
	cm := newTestContextMaker(t)
	cm.MakeContextFunc = func(r *http.Request) (i context.Context, e error) {
		createdContext = context.Background()
		return createdContext, nil
	}

	l.GetExamplesFunc = func(ctx context.Context) (predictions data.ExamplePredictions, e error) {
		return nil, nil
	}

	var calledPredict bool
	pm.PredictFunc = func(ctx context.Context, predictions []float64) (p float64, err error) {
		calledPredict = true
		if ctx == nil {
			t.Error("Got nil context, expected non-nil context")
		}
		if ctx != createdContext {
			t.Error("Got context that didn't match one we created")
		}
		if !reflect.DeepEqual(predictions, []float64{0.1, 0.2}) {
			t.Error("Unexpected prediction values")
		}
		return 0, errors.New("bluh")
	}

	c := &Index{
		ExampleLister:   l,
		PredictionMaker: pm,
	}
	handler := c.HandleFunc(cm, r)
	formValues := make(url.Values)
	formValues.Add("assignments", "0.1,0.2")
	handler(nil, &http.Request{Form: formValues})

	if !calledPredict {
		t.Error("Expected get examples to be called, was not called")
	}
	if !calledOnResult {
		t.Error("Expected responder's OnResult method to be called, was not called")
	}
}

func TestIndex_HandleFunc_ExamplesErr_NoAssignments(t *testing.T) {
	t.Parallel()

	var createdContext context.Context

	l := newTestExamplesLister(t)
	pm := newTestPredictionMaker(t)

	calledOnResult := false
	r := newTestWebIndexResponder(t)
	r.OnResultFunc = func(w http.ResponseWriter, result *IndexResult) {
		calledOnResult = true
		if result.AssignmentsStr != "" {
			t.Errorf("Result AssignmentStr should be '%s', was '%s'", "", result.AssignmentsStr)
		}
		if len(result.ExampleList) != 0 {
			t.Errorf("Result ExampleList should be empty, was not")
		}
		if result.ExampleListErr == nil {
			t.Errorf("Result ExampleListErr should be non-nil, was nil")
		}
		if result.Prediction != nil {
			t.Errorf("Result Prediction should be nil, was %f", *result.Prediction)
		}
		if result.PredictionErr != nil {
			t.Errorf("Result PredictionErr should be nil, was %s", result.PredictionErr)
		}
	}
	cm := newTestContextMaker(t)
	cm.MakeContextFunc = func(r *http.Request) (i context.Context, e error) {
		createdContext = context.Background()
		return createdContext, nil
	}

	var calledGetExamples bool
	l.GetExamplesFunc = func(ctx context.Context) (predictions data.ExamplePredictions, e error) {
		if ctx == nil {
			t.Error("Got nil context, expected non-nil context")
		}
		if ctx != createdContext {
			t.Error("Got context that didn't match one we created")
		}
		calledGetExamples = true
		return nil, errors.New("bluh")
	}

	c := &Index{
		ExampleLister:   l,
		PredictionMaker: pm,
	}
	handler := c.HandleFunc(cm, r)
	handler(nil, &http.Request{})

	if !calledGetExamples {
		t.Error("Expected get examples to be called, was not called")
	}
	if !calledOnResult {
		t.Error("Expected responder's OnResult method to be called, was not called")
	}
}

func TestIndex_HandleFunc_Examples_NoAssignments(t *testing.T) {
	t.Parallel()

	var createdContext context.Context

	l := newTestExamplesLister(t)
	pm := newTestPredictionMaker(t)

	calledOnResult := false
	r := newTestWebIndexResponder(t)
	r.OnResultFunc = func(w http.ResponseWriter, result *IndexResult) {
		calledOnResult = true
		if result.AssignmentsStr != "" {
			t.Errorf("Result AssignmentStr should be '%s', was '%s'", "", result.AssignmentsStr)
		}
		if len(result.ExampleList) == 0 {
			t.Errorf("Result ExampleList should be non-empty, was empty")
		}
		if result.ExampleList[1].Title != "bluh2" {
			t.Errorf("Result ExampleList second prediction title should be 'bluh', was '%s'", result.ExampleList[1].Title)
		}
		if result.ExampleList[0].ResultErr == nil {
			t.Errorf("Result ExampleList first prediction should have a non-nil error, had nil error")
		}
		if result.ExampleList[1].ResultErr != nil {
			t.Errorf("Result ExampleList second prediction should have a nil error, had non-nil error")
		}
		if result.ExampleList[1].Result != 0.32 {
			t.Errorf("Result ExampleList second prediction result should be 0.32, was %g", result.ExampleList[1].Result)
		}
		if result.ExampleListErr != nil {
			t.Errorf("Result ExampleListErr should be nil, was %s", result.ExampleListErr)
		}
		if result.Prediction != nil {
			t.Errorf("Result Prediction should be nil, was %f", *result.Prediction)
		}
		if result.PredictionErr != nil {
			t.Errorf("Result PredictionErr should be nil, was %s", result.PredictionErr)
		}
	}
	cm := newTestContextMaker(t)
	cm.MakeContextFunc = func(r *http.Request) (i context.Context, e error) {
		createdContext = context.Background()
		return createdContext, nil
	}

	var calledGetExamples bool
	l.GetExamplesFunc = func(ctx context.Context) (predictions data.ExamplePredictions, e error) {
		if ctx == nil {
			t.Error("Got nil context, expected non-nil context")
		}
		if ctx != createdContext {
			t.Error("Got context that didn't match one we created")
		}
		calledGetExamples = true
		return []data.ExamplePrediction{
			{
				PredictionSummary: &predictions2.PredictionSummary{
					Title: "bluh",
				},
				Assignments: []float64{0.5, 0.7},
			},
			{
				PredictionSummary: &predictions2.PredictionSummary{
					Title: "bluh2",
				},
				Assignments: []float64{0.3, 0.4},
			},
		}, nil
	}

	var calledPredictCount = 0
	pm.PredictFunc = func(ctx context.Context, predictions []float64) (p float64, err error) {
		calledPredictCount++
		if ctx == nil {
			t.Error("Got nil context, expected non-nil context")
		}
		if ctx != createdContext {
			t.Error("Got context that didn't match one we created")
		}
		if calledPredictCount == 1 {
			if !reflect.DeepEqual(predictions, []float64{0.5, 0.7}) {
				t.Error("Unexpected prediction values")
			}
			return 0, errors.New("bluh")
		} else if calledPredictCount == 2 {
			if !reflect.DeepEqual(predictions, []float64{0.3, 0.4}) {
				t.Error("Unexpected prediction values")
			}
			return 0.32, nil
		} else {
			t.Errorf("Expected Predict to be called only twice, called additional time")
			return 0, nil
		}
	}

	c := &Index{
		ExampleLister:   l,
		PredictionMaker: pm,
	}
	handler := c.HandleFunc(cm, r)
	handler(nil, &http.Request{})

	if !calledGetExamples {
		t.Error("Expected get examples to be called, was not called")
	}
	if calledPredictCount != 2 {
		t.Errorf("Expected predict to be called twice, was called %d times", calledPredictCount)
	}
	if !calledOnResult {
		t.Error("Expected responder's OnResult method to be called, was not called")
	}
}

func TestIndex_HandleFunc_ContextError(t *testing.T) {
	l := newTestExamplesLister(t)
	pm := newTestPredictionMaker(t)

	calledOnContextError := false
	r := newTestWebIndexResponder(t)
	r.OnContextErrorFunc = func(w http.ResponseWriter, err error) {
		calledOnContextError = true
		if err == nil {
			t.Error("Expected non-nil error in OnContextError, got nil error")
		}
	}
	cm := newTestContextMaker(t)
	cm.MakeContextFunc = func(r *http.Request) (i context.Context, e error) {
		return nil, errors.New("bluh")
	}

	c := &Index{
		ExampleLister:   l,
		PredictionMaker: pm,
	}
	handler := c.HandleFunc(cm, r)
	handler(nil, &http.Request{})

	if !calledOnContextError {
		t.Error("Expected responder's OnContextError method to be called, was not called")
	}
}

type testWebIndexResponder struct {
	OnContextErrorFunc func(w http.ResponseWriter, err error)
	OnResultFunc       func(w http.ResponseWriter, r *IndexResult)
}

func newTestWebIndexResponder(t *testing.T) *testWebIndexResponder {
	return &testWebIndexResponder{
		OnContextErrorFunc: func(w http.ResponseWriter, err error) {
			t.Error("OnContextErrorFunc should not be called")
		},
		OnResultFunc: func(w http.ResponseWriter, result *IndexResult) {
			t.Error("OnResultFunc should not be called")
		},
	}
}

func (r *testWebIndexResponder) OnContextError(w http.ResponseWriter, err error) {
	r.OnContextErrorFunc(w, err)
}

func (r *testWebIndexResponder) OnResult(w http.ResponseWriter, result *IndexResult) {
	r.OnResultFunc(w, result)
}
