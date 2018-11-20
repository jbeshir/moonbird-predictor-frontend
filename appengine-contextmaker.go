package main

import (
	"context"
	"google.golang.org/appengine"
	"net/http"
)

type AppEngineContextMaker struct{}

func (cm *AppEngineContextMaker) MakeContext(r *http.Request) (context.Context, error) {
	ctx := appengine.NewContext(r)
	return appengine.Namespace(ctx, "moonbird-predictor-frontend")
}
