package controllers

import (
	"context"
	"net/http"
	"time"
)

type ModelRetrain struct {
	Trainer ModelTrainer
}

type WebModelRetrainResponder interface {
	OnContextError(w http.ResponseWriter, err error)
	OnError(w http.ResponseWriter, err error)
	OnSuccess(w http.ResponseWriter)
}

func (c *ModelRetrain) HandleFunc(cm ContextMaker, resp WebExamplesUpdateResponder) http.HandlerFunc {
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

func (c *ModelRetrain) handle(ctx context.Context) error {
	err := c.Trainer.Retrain(ctx, time.Now())
	if err != nil {
		return err
	}

	return nil
}
