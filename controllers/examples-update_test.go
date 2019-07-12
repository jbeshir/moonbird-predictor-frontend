package controllers

import (
	"context"
	"errors"
	"github.com/jbeshir/moonbird-auth-frontend/testhelpers"
	"github.com/jbeshir/moonbird-predictor-frontend/data"
	"net/http"
	"testing"
)

func TestExamplesUpdate_HandleFunc_Success(t *testing.T) {
	t.Parallel()

	var createdContext context.Context

	calledUpdateExamples := false
	l := newTestExamplesLister(t)
	l.UpdateExamplesFunc = func(ctx context.Context) (predictions data.ExamplePredictions, e error) {
		if ctx == nil {
			t.Error("Got nil context, expected non-nil context")
		}
		calledUpdateExamples = true
		return nil, nil
	}

	calledOnSuccess := false
	r := newTestWebExamplesUpdateResponder(t)
	r.OnSuccessFunc = func(w http.ResponseWriter) {
		calledOnSuccess = true
	}

	cm := testhelpers.NewContextMaker(t)
	cm.MakeContextFunc = func(r *http.Request) (i context.Context, e error) {
		createdContext = context.Background()
		return createdContext, nil
	}

	c := &ExamplesUpdate{
		ExampleLister: l,
	}
	handler := c.HandleFunc(cm, r)
	handler(nil, &http.Request{})

	if !calledUpdateExamples {
		t.Error("Expected update examples to be called, was not called")
	}
	if !calledOnSuccess {
		t.Error("Expected responder's OnSuccess method to be called, was not called")
	}
}

func TestExamplesUpdate_HandleFunc_ContextError(t *testing.T) {
	t.Parallel()

	l := newTestExamplesLister(t)

	calledOnContextError := false
	r := newTestWebExamplesUpdateResponder(t)
	r.OnContextErrorFunc = func(w http.ResponseWriter, err error) {
		calledOnContextError = true
		if err == nil {
			t.Error("Expected non-nil error in OnContextError, got nil error")
		}
	}
	cm := testhelpers.NewContextMaker(t)
	cm.MakeContextFunc = func(r *http.Request) (i context.Context, e error) {
		return nil, errors.New("bluh")
	}

	c := &ExamplesUpdate{
		ExampleLister: l,
	}
	handler := c.HandleFunc(cm, r)
	handler(nil, &http.Request{})

	if !calledOnContextError {
		t.Error("Expected responder's OnContextError method to be called, was not called")
	}
}

func TestExamplesUpdate_HandleFunc_UpdateError(t *testing.T) {
	t.Parallel()

	var createdContext context.Context

	calledUpdateExamples := false
	l := newTestExamplesLister(t)
	l.UpdateExamplesFunc = func(ctx context.Context) (predictions data.ExamplePredictions, e error) {
		if ctx == nil {
			t.Error("Got nil context, expected non-nil context")
		}
		calledUpdateExamples = true
		return nil, errors.New("bluh")
	}

	calledOnError := false
	r := newTestWebExamplesUpdateResponder(t)
	r.OnErrorFunc = func(ctx context.Context, w http.ResponseWriter, err error) {
		calledOnError = true
		if err == nil {
			t.Error("Expected non-nil error in OnError, got nil error")
		}
	}

	cm := testhelpers.NewContextMaker(t)
	cm.MakeContextFunc = func(r *http.Request) (i context.Context, e error) {
		createdContext = context.Background()
		return createdContext, nil
	}

	c := &ExamplesUpdate{
		ExampleLister: l,
	}
	handler := c.HandleFunc(cm, r)
	handler(nil, &http.Request{})

	if !calledUpdateExamples {
		t.Error("Expected update examples to be called, was not called")
	}
	if !calledOnError {
		t.Error("Expected responder's OnError method to be called, was not called")
	}
}

func newTestWebExamplesUpdateResponder(t *testing.T) *testWebExamplesUpdateResponder {
	return &testWebExamplesUpdateResponder{
		OnContextErrorFunc: func(w http.ResponseWriter, err error) {
			t.Error("OnContextErrorFunc should not be called")
		},
		OnErrorFunc: func(ctx context.Context, w http.ResponseWriter, err error) {
			t.Error("OnErrorFunc should not be called")
		},
		OnSuccessFunc: func(w http.ResponseWriter) {
			t.Error("OnSuccessFunc should not be called")
		},
	}
}

type testWebExamplesUpdateResponder struct {
	OnContextErrorFunc func(w http.ResponseWriter, err error)
	OnErrorFunc        func(ctx context.Context, w http.ResponseWriter, err error)
	OnSuccessFunc      func(w http.ResponseWriter)
}

func (r *testWebExamplesUpdateResponder) OnContextError(w http.ResponseWriter, err error) {
	r.OnContextErrorFunc(w, err)
}

func (r *testWebExamplesUpdateResponder) OnError(ctx context.Context, w http.ResponseWriter, err error) {
	r.OnErrorFunc(ctx, w, err)
}

func (r *testWebExamplesUpdateResponder) OnSuccess(w http.ResponseWriter) {
	r.OnSuccessFunc(w)
}
