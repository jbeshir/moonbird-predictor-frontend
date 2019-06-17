package controllers

import (
	"context"
	"errors"
	"github.com/jbeshir/moonbird-predictor-frontend/testhelpers"
	"net/http"
	"testing"
	"time"
)

func TestModelRetrain_HandleFunc_Success(t *testing.T) {
	t.Parallel()

	var createdContext context.Context

	calledRetrain := false
	tr := newTestModelTrainer(t)
	tr.RetrainFunc = func(ctx context.Context, now time.Time) error {
		if ctx == nil {
			t.Error("Got nil context, expected non-nil context")
		}
		calledRetrain = true
		return nil
	}

	calledClear := false
	cache := newTestPredictionCache(t)
	cache.FlushFunc = func(ctx context.Context) error {
		if ctx == nil {
			t.Error("Got nil context, expected non-nil context")
		}
		if !calledRetrain {
			t.Error("Cache flush called without retrain being called first")
		}
		calledClear = true
		return nil
	}

	calledOnSuccess := false
	r := newTestWebModelRetrainResponder(t)
	r.OnSuccessFunc = func(w http.ResponseWriter) {
		calledOnSuccess = true
	}

	cm := testhelpers.NewContextMaker(t)
	cm.MakeContextFunc = func(r *http.Request) (i context.Context, e error) {
		createdContext = context.Background()
		return createdContext, nil
	}

	c := &ModelRetrain{
		Trainer:         tr,
		PredictionCache: cache,
	}
	handler := c.HandleFunc(cm, r)
	handler(nil, &http.Request{})

	if !calledRetrain {
		t.Error("Expected retrain to be called, was not called")
	}
	if !calledClear {
		t.Error("Expected cache flush to be called, was not called")
	}
	if !calledOnSuccess {
		t.Error("Expected responder's OnSuccess method to be called, was not called")
	}
}

func TestModelRetrain_HandleFunc_Error(t *testing.T) {
	t.Parallel()

	var createdContext context.Context

	calledRetrain := false
	tr := newTestModelTrainer(t)
	tr.RetrainFunc = func(ctx context.Context, now time.Time) error {
		if ctx == nil {
			t.Error("Got nil context, expected non-nil context")
		}
		calledRetrain = true
		return errors.New("bluh")
	}

	calledOnError := false
	r := newTestWebModelRetrainResponder(t)
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

	c := &ModelRetrain{
		Trainer: tr,
	}
	handler := c.HandleFunc(cm, r)
	handler(nil, &http.Request{})

	if !calledRetrain {
		t.Error("Expected retrain to be called, was not called")
	}
	if !calledOnError {
		t.Error("Expected responder's OnError method to be called, was not called")
	}
}

func TestModelRetrain_HandleFunc_ContextError(t *testing.T) {
	t.Parallel()

	calledOnContextError := false
	r := newTestWebModelRetrainResponder(t)
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

	c := &ModelRetrain{}
	handler := c.HandleFunc(cm, r)
	handler(nil, &http.Request{})

	if !calledOnContextError {
		t.Error("Expected responder's OnContextError method to be called, was not called")
	}
}

func newTestWebModelRetrainResponder(t *testing.T) *testWebModelRetrainResponder {
	return &testWebModelRetrainResponder{
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

type testWebModelRetrainResponder struct {
	OnContextErrorFunc func(w http.ResponseWriter, err error)
	OnErrorFunc        func(ctx context.Context, w http.ResponseWriter, err error)
	OnSuccessFunc      func(w http.ResponseWriter)
}

func (r *testWebModelRetrainResponder) OnContextError(w http.ResponseWriter, err error) {
	r.OnContextErrorFunc(w, err)
}

func (r *testWebModelRetrainResponder) OnError(ctx context.Context, w http.ResponseWriter, err error) {
	r.OnErrorFunc(ctx, w, err)
}

func (r *testWebModelRetrainResponder) OnSuccess(w http.ResponseWriter) {
	r.OnSuccessFunc(w)
}

func newTestPredictionCache(t *testing.T) *testPredictionCache {
	return &testPredictionCache{
		FlushFunc: func(ctx context.Context) error {
			t.Error("Flush should not be called")
			return nil
		},
	}
}

type testPredictionCache struct {
	FlushFunc func(ctx context.Context) error
}

func (c *testPredictionCache) Flush(ctx context.Context) error {
	return c.FlushFunc(ctx)
}
