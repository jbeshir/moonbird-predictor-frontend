package main

import (
	"github.com/jbeshir/moonbird-predictor-frontend/controllers"
	"github.com/jbeshir/moonbird-predictor-frontend/responders"
	"github.com/jbeshir/predictionbook-extractor/htmlfetcher"
	"golang.org/x/time/rate"
	"google.golang.org/appengine"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	contextMaker := &AppEngineContextMaker{}
	exampleLister := &PredictionBookLister{
		Fetcher: htmlfetcher.NewFetcher(rate.NewLimiter(1, 2), 2),
	}

	indexController := &controllers.Index{
		ExampleLister:   exampleLister,
		PredictionMaker: &MLEnginePredictionMaker{},
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
