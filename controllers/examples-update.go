package controllers

import (
	"context"
	"github.com/jbeshir/moonbird-predictor-frontend/ctxlogrus"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net/http"
)

type ExamplesUpdate struct {
	ExampleLister ExampleLister
}

type WebExamplesUpdateResponder interface {
	OnContextError(w http.ResponseWriter, err error)
	OnError(ctx context.Context, w http.ResponseWriter, err error)
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
			resp.OnError(ctx, w, err)
		} else {
			resp.OnSuccess(w)
		}
	}
}

func (c *ExamplesUpdate) handle(ctx context.Context) error {
	ctx = ctxlogrus.WithFields(ctx, logrus.Fields{
		"controller": "ExamplesUpdate",
	})

	_, err := c.ExampleLister.UpdateExamples(ctx)
	return errors.Wrap(err, "")
}
