package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/ml/v1"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", handle)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

func handle(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var assignments [][1]float64
	assignmentsStr := strings.Split(r.FormValue("assignments"), ",")
	for _, assignmentStr := range assignmentsStr {
		assignment, err := strconv.ParseFloat(assignmentStr, 64)
		if err != nil {
			fmt.Fprintln(w, err)
			return
		}

		assignments = append(assignments, [1]float64{assignment})
	}

	data, err := makePrediction(ctx, assignments)
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}
	fmt.Fprintln(w, data)
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

func makePrediction(ctx context.Context, predictions [][1]float64) (float64, error) {
	req, err := newMLRequest(predictions)
	if err != nil {
		return 0, err
	}

	client, err := google.DefaultClient(ctx, ml.CloudPlatformScope)
	if err != nil {
		return 0, err
	}

	s, err := ml.New(client)
	if err != nil {
		return 0, err
	}

	mlPredictCall := s.Projects.Predict("projects/moonbird-beshir/models/Predictor", req)
	r, err := mlPredictCall.Context(ctx).Do()
	if err != nil {
		return 0, err
	}

	var result result
	err = json.NewDecoder(strings.NewReader(r.Data)).Decode(&result)
	if err != nil {
		return 0, err
	}
	if len(result.Predictions) != 1 || len(result.Predictions[0].Income) != 1 {
		return 0, errors.New("malformed predict response: Did not get one and only one probability")
	}

	return result.Predictions[0].Income[0], nil
}

func newMLRequest(predictions [][1]float64) (*ml.GoogleCloudMlV1__PredictRequest, error) {

	jsonreq := request{
		Instances: []requestInput{
			{
				Input: predictions,
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
