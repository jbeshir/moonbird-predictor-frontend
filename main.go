package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"google.golang.org/appengine"
	"google.golang.org/appengine/memcache"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/ml/v1"
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

	indexTemplate.Execute(w, &struct {
		AssignmentsStr string
		Prediction     *float64
		Err            error
	}{
		AssignmentsStr: r.FormValue("assignments"),
		Prediction:     prediction,
		Err:            err,
	})
}

type request struct {
	Instances []requestInput `json:"instances"`
}
type requestInput struct {
	Input [][1]float64 `json:"input"`
}

type result struct {
	Predictions []resultPrediction `json:"predictions"`
}

type resultPrediction struct {
	Income []float64 `json:"income"`
}

func makePrediction(ctx context.Context, predictions []float64) (p float64, err error) {

	cacheKey := generatePredictionCacheKey(predictions)
	_, err = binaryMemcacheCodec.Get(ctx, cacheKey, &p)
	if err == nil {
		return
	}

	client, err := google.DefaultClient(ctx, ml.CloudPlatformScope)
	if err != nil {
		return 0, errors.Wrap(err, "makePrediction couldn't create client")
	}

	req, err := newMLRequest(predictions)
	if err != nil {
		return 0, errors.Wrap(err, "makePrediction couldn't create request")
	}

	s, err := ml.New(client)
	if err != nil {
		return 0, errors.Wrap(err, "makePrediction couldn't create service")
	}

	mlPredictCall := s.Projects.Predict("projects/moonbird-beshir/models/Predictor", req)
	r, err := mlPredictCall.Context(ctx).Do()
	if err != nil {
		return 0, errors.Wrap(err, "makePrediction couldn't run request")
	}
	if r.HTTPStatusCode != http.StatusOK {
		return 0, errors.Errorf("makePrediction couldn't run request, status code: %d", r.HTTPStatusCode)
	}

	var result result
	err = json.NewDecoder(strings.NewReader(r.Data)).Decode(&result)
	if err != nil {
		return 0, errors.Wrap(err, "makePrediction couldn't decode response")
	}
	if len(result.Predictions) != 1 || len(result.Predictions[0].Income) != 1 {
		return 0, errors.New("makePrediction got malformed predict response: Did not get one and only one probability")
	}
	p = result.Predictions[0].Income[0]

	// We ignore failures in writing to memcache.
	cacheItem := &memcache.Item{
		Key:    cacheKey,
		Object: &p,
	}
	binaryMemcacheCodec.Set(ctx, cacheItem)

	return
}

func newMLRequest(predictions []float64) (*ml.GoogleCloudMlV1__PredictRequest, error) {

	var predictionsMatrix [][1]float64
	for _, p := range predictions {
		if p < 0 || p > 1 {
			return nil, errors.Errorf("Probability assignment out of range: %g", p)
		}
		predictionsMatrix = append(predictionsMatrix, [1]float64{p})
	}

	jsonreq := request{
		Instances: []requestInput{
			{
				Input: predictionsMatrix,
			},
		},
	}

	payload, err := json.Marshal(&jsonreq)
	if err != nil {
		return nil, errors.Wrap(err, "mkreq could not marshal JSON")
	}

	body := &ml.GoogleApi__HttpBody{
		Data: string(payload),
	}

	req := ml.GoogleCloudMlV1__PredictRequest{
		HttpBody: body,
	}

	req.HttpBody.ContentType = "application/json"

	return &req, nil
}

func generatePredictionCacheKey(predictions []float64) string {
	var buf strings.Builder
	for _, p := range predictions {
		binary.Write(&buf, binary.BigEndian, p)
	}
	return buf.String()
}

var binaryMemcacheCodec = memcache.Codec{
	Marshal:   binaryMarshal,
	Unmarshal: binaryUnmarshal,
}

// Can only marshal fixed-size data as defined by the encoding/binary package.
func binaryMarshal(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	err := binary.Write(&buf, binary.BigEndian, v)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Can only unmarshal fixed-size data as defined by the encoding/binary package.
func binaryUnmarshal(data []byte, v interface{}) error {
	return binary.Read(bytes.NewReader(data), binary.BigEndian, v)
}
