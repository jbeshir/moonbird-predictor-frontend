package main

import (
	"github.com/jbeshir/moonbird-predictor-frontend/aengine"
	"github.com/jbeshir/moonbird-predictor-frontend/controllers"
	"github.com/jbeshir/moonbird-predictor-frontend/mlclient"
	"github.com/jbeshir/moonbird-predictor-frontend/responders"
	"github.com/jbeshir/predictionbook-extractor/htmlfetcher"
	"golang.org/x/time/rate"
	"google.golang.org/api/ml/v1"
	"google.golang.org/appengine"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	contextMaker := &aengine.ContextMaker{
		Namespace: "moonbird-predictor-frontend",
	}

	exampleLister := &PredictionBookLister{
		Fetcher: htmlfetcher.NewFetcher(rate.NewLimiter(1, 2), 2),
	}

	predictionMaker := &mlclient.PredictionMaker{
		CacheStorage: &aengine.CacheStorage{
			Prefix: "~",
			Codec:  aengine.BinaryMemcacheCodec,
		},
		HttpClientMaker: &aengine.AuthenticatedClientMaker{
			Scope: ml.CloudPlatformScope,
		},
	}

	indexController := &controllers.Index{
		ExampleLister:   exampleLister,
		PredictionMaker: predictionMaker,
	}
	indexResponder := &responders.WebIndexResponder{}
	http.Handle("/", indexController.HandleFunc(contextMaker, indexResponder))

	pbUpdateController := &controllers.ExamplesUpdate{
		ExampleLister: exampleLister,
	}
	pbUpdateResponder := &responders.WebSimpleResponder{}
	http.Handle("/cron/pb-update", pbUpdateController.HandleFunc(contextMaker, pbUpdateResponder))

	appengine.Main()
}
