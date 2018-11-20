package responders

import (
	"github.com/jbeshir/moonbird-predictor-frontend/controllers"
	"html/template"
	"net/http"
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
{{if .Prediction}}<div class="prediction-result-msg"><div class="prediction-result-title">Predicted Likelihood</div><div class="prediction-result">{{.Prediction}}</div></div>{{end}}
{{if .PredictionErr}}<div class="prediction-fault-msg">Fault predicting using given sequence!<div id="prediction-fault">{{.PredictionErr}}</div></div>{{end}}
</form>
{{if .ExampleList}}<div class="example-list">{{.ExampleList}}<div>{{end}}
{{if .ExampleListErr}}<div class="example-list-fault-msg">{{.ExampleListErr}}<div>{{end}}
</body>
</html>`))

type WebIndexResponder struct{}

func (_ *WebIndexResponder) OnContextError(w http.ResponseWriter, err error) {
	http.Error(w, "Internal Server Error", 500)
}

func (_ *WebIndexResponder) OnResult(w http.ResponseWriter, r *controllers.IndexResult) {
	indexTemplate.Execute(w, r)
}
