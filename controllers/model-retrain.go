package controllers

import (
	"context"
	"github.com/jbeshir/moonbird-predictor-frontend/ctxlogrus"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
)

type ModelRetrain struct {
	Trainer         ModelTrainer
	PredictionCache PredictionCache
}

type WebModelRetrainResponder interface {
	OnContextError(w http.ResponseWriter, err error)
	OnError(ctx context.Context, w http.ResponseWriter, err error)
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
			resp.OnError(ctx, w, err)
		} else {
			resp.OnSuccess(w)
		}
	}
}

func (c *ModelRetrain) handle(ctx context.Context) error {
	ctx = ctxlogrus.WithFields(ctx, logrus.Fields{
		"controller": "ModelRetrain",
	})

	err := c.Trainer.Retrain(ctx, time.Now())
	if err != nil {
		return errors.Wrap(err, "")
	}
	return errors.Wrap(c.PredictionCache.Flush(ctx), "")
}
