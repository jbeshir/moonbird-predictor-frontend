package controllers

import (
	"context"
	"net/http"
)

type ExamplesUpdate struct {
	ExampleLister ExampleLister
}

type WebExamplesUpdateResponder interface {
	OnContextError(w http.ResponseWriter, err error)
	OnError(w http.ResponseWriter, err error)
	OnSuccess(w http.ResponseWriter)
}

func (c *ExamplesUpdate) HandleFunc(cm ContextMaker, resp WebExamplesUpdateResponder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, err := cm.MakeContext(r)
		if err != nil {
			resp.OnContextError(w, err)
			return
		}

		err = c.handle(ctx)
		if err != nil {
			resp.OnError(w, err)
		} else {
			resp.OnSuccess(w)
		}
	}
}

func (c *ExamplesUpdate) handle(ctx context.Context) error {
	_, err := c.ExampleLister.UpdateExamples(ctx)
	if err != nil {
		return err
	}

	return nil
}
