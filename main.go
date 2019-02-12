package main

import (
	"github.com/jbeshir/moonbird-predictor-frontend/aengine"
	"github.com/jbeshir/moonbird-predictor-frontend/controllers"
	"github.com/jbeshir/moonbird-predictor-frontend/mlclient"
	"github.com/jbeshir/moonbird-predictor-frontend/pbook"
	"github.com/jbeshir/moonbird-predictor-frontend/responders"
	"github.com/jbeshir/predictionbook-extractor/htmlfetcher"
	"github.com/jbeshir/predictionbook-extractor/predictions"
	"golang.org/x/time/rate"
	"google.golang.org/api/ml/v1"
	"google.golang.org/api/storage/v1"
	"google.golang.org/appengine"
	"google.golang.org/appengine/memcache"
	"net/http"
	"os"
	"time"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	contextMaker := &aengine.ContextMaker{
		Namespace: "moonbird-predictor-frontend",
	}

	exampleSource := predictions.NewSource(
		htmlfetcher.NewFetcher(rate.NewLimiter(1, 2), 2),
		"https://predictionbook.com")
	exampleLister := &pbook.Lister{
		PredictionSource: exampleSource,
		CacheStore: &aengine.CacheStore{
			Prefix: "pbook-",
			Codec:  memcache.Gob,
		},
		PersistentStore: &aengine.PersistentStore{
			Prefix: "pbook-",
		},
	}

	predictionMaker := &mlclient.PredictionMaker{
		CacheStorage: &aengine.CacheStore{
			Prefix: "~",
			Codec:  aengine.BinaryMemcacheCodec,
		},
		HttpClientMaker: &aengine.AuthenticatedClientMaker{
			Scope: []string{
				ml.CloudPlatformScope,
			},
		},
	}

	indexController := &controllers.Index{
		ExampleLister:   exampleLister,
		PredictionMaker: predictionMaker,
	}
	indexResponder := &responders.WebIndexResponder{}
	http.Handle("/", indexController.HandleFunc(contextMaker, indexResponder))

	cronResponder := &responders.WebSimpleResponder{
		ExposeErrors: true,
	}

	pbUpdateController := &controllers.ExamplesUpdate{
		ExampleLister: exampleLister,
	}
	http.Handle("/cron/pb-update", pbUpdateController.HandleFunc(contextMaker, cronResponder))

	modelTrainer := &mlclient.Trainer{
		PersistentStore: &aengine.PersistentStore{
			Prefix: "model-",
		},
		FileStore: &aengine.GcsFileStore{
			Bucket: "moonbird-data",
			Prefix: "predictor/",
		},
		ModelPath:    "moonbird-models/predictor",
		DataPath:     "moonbird-data/predictor",
		SleepFunc:    time.Sleep,
		TrainPackage: "gs://moonbird-models/predictor/trainer.tar.gz",
		HttpClientMaker: &aengine.AuthenticatedClientMaker{
			Scope: []string{
				ml.CloudPlatformScope,
				storage.CloudPlatformScope,
			},
		},
	}
	mlRetrainController := &controllers.ModelRetrain{
		Trainer: modelTrainer,
	}
	http.Handle("/cron/ml-retrain", mlRetrainController.HandleFunc(contextMaker, cronResponder))

	appengine.Main()
}
