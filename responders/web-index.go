package responders

import (
	"github.com/jbeshir/moonbird-predictor-frontend/controllers"
	"html/template"
	"net/http"
)

var indexTemplate = template.Must(template.New("index").Funcs(template.FuncMap{
	"DerefFloat64": func(f *float64) float64 { return *f },
}).Parse(
	`<html>
<head>
	<link href="https://fonts.googleapis.com/css?family=Roboto|Roboto+Slab" rel="stylesheet">
	<link rel="stylesheet" type="text/css" href="/static/moonbird.css" />
</head>
<body class="predict-page">
<h1>Moonbird Predictor</h1>
<form id="prediction-form" action="/">
	<div>Input a comma-separated series of human-assigned probabilties (between 0 and 1) to get Moonbird Predictor's best guess at the likelihood of the event happening. Slightly outperforms naive averaging in validation against PredictionBook data!</div>
	<input type="text" placeholder="Probabilities go here..." name="assignments" value="{{.AssignmentsStr}}" class="prediction-text-input"></input>
{{if .Prediction}}<div class="prediction-result-msg"><div class="prediction-result-title">Predicted Likelihood</div><div class="prediction-result">{{printf "%.3f" (DerefFloat64 .Prediction)}}</div></div>{{end}}
{{if .PredictionErr}}<div class="prediction-fault-msg">Fault predicting using given sequence!<div id="prediction-fault">{{.PredictionErr}}</div></div>{{end}}
</form>
{{if .ExampleList}}<div class="example-list">
	<div class="example-headers">
		<div class="example-header">Proposition</div>
		<div class="example-header">Prediction</div>
	</div>
	{{range .ExampleList}}
		<div class="example">
			<a href="https://predictionbook.com/predictions/{{.Id}}" class="example-link">{{.Title}}</a>
			{{if .Result}}<span class="example-result">{{printf "%.3f" .Result}}</span>{{end}}
			{{if .ResultErr}}<span class="example-result-error">{{.ResultErr}}</span>{{end}}
		</div>
	{{end}}
</div>{{end}}
{{if .ExampleListErr}}<div class="example-list-fault-msg">{{.ExampleListErr}}</div>{{end}}
</body>
</html>`))

type WebIndexResponder struct{}

func (_ *WebIndexResponder) OnContextError(w http.ResponseWriter, err error) {
	http.Error(w, "Internal Server Error", 500)
}

func (_ *WebIndexResponder) OnResult(w http.ResponseWriter, r *controllers.IndexResult) {
	indexTemplate.Execute(w, r)
}
