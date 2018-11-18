package main

import (
	"google.golang.org/appengine"
	"html/template"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var indexTemplate = template.Must(template.New("index").Parse(
	`<html>
<head>
	<link href="https://fonts.googleapis.com/css?family=Roboto|Roboto+Slab" rel="stylesheet">
	<link rel="stylesheet" type="text/css" href="/static/moonbird.css" />
</head>
<body class="predict-page">
<h1>Moonbird Predictor</h1>
<form id="prediction-form" action="/">
	<div>Input a comma-separated series of human-assigned probabilties to get Moonbird Predictor's best guess at the likelihood of the event happening. Slightly outperforms naive averaging in validation against PredictionBook data!</div>
	<input type="text" placeholder="Probabilities go here..." name="assignments" value="{{.AssignmentsStr}}" class="prediction-text-input"></input>
{{if .Err}}<div class="prediction-fault-msg">Fault predicting using given sequence!<div id="prediction-fault">{{.Err}}</div></div>{{end}}
{{if .Prediction}}<div class="prediction-result-msg"><div class="prediction-result-title">Predicted Likelihood</div><div class="prediction-result">{{.Prediction}}</div></div>{{end}}
</form>
{{if .ExampleList}}<div class="example-list">{{.ExampleList}}<div>{{end}}
</body>
</html>`))

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", handle)
	appengine.Main()
}

func handle(w http.ResponseWriter, r *http.Request) {
	var prediction *float64
	var err error

	ctx := appengine.NewContext(r)
	ctx, err = appengine.Namespace(ctx, "moonbird-predictor-frontend")

	var assignments []float64
	assignmentStrs := strings.Split(r.FormValue("assignments"), ",")
	for _, assignmentStr := range assignmentStrs {
		if assignmentStr == "" {
			continue
		}

		var assignment float64
		assignment, err = strconv.ParseFloat(strings.TrimSpace(assignmentStr), 64)
		if err != nil {
			break
		}

		assignments = append(assignments, assignment)
	}

	if err == nil && len(assignments) > 0 {
		var p float64
		p, err = makePrediction(ctx, assignments)
		if err == nil {
			prediction = &p
		}
	}

	var exampleList strings.Builder
	latest, listErr := getLatestPredictionBook(ctx)
	if listErr == nil {
		for _, summary := range latest.Summaries {
			var assignments []float64
			for _, r := range latest.Responses {
				if r.Prediction != summary.Id {
					continue
				}
				if math.IsNaN(r.Confidence) {
					continue
				}

				assignments = append(assignments, r.Confidence)
			}

			exampleList.WriteString(strconv.FormatInt(summary.Id, 10))
			examplePrediction, err := makePrediction(ctx, assignments)
			exampleList.WriteString(":")
			if err == nil {
				exampleList.WriteString(strconv.FormatFloat(examplePrediction, 'g', 4, 64))
			} else {
				exampleList.WriteString(err.Error())
			}
			exampleList.WriteString(" ")
		}
	} else {
		err = listErr
	}

	indexTemplate.Execute(w, &struct {
		AssignmentsStr string
		Prediction     *float64
		ExampleList    string
		Err            error
	}{
		AssignmentsStr: r.FormValue("assignments"),
		Prediction:     prediction,
		ExampleList:    exampleList.String(),
		Err:            err,
	})
}
