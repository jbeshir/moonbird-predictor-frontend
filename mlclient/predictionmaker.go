package mlclient

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"github.com/pkg/errors"
	"google.golang.org/api/ml/v1"
	"net/http"
	"strings"
)

type PredictionMaker struct {
	CacheStorage    CacheStorage
	HttpClientMaker HttpClientMaker
}

func (pm *PredictionMaker) Predict(ctx context.Context, predictions []float64) (p float64, err error) {

	cacheKey := generatePredictionCacheKey(predictions)
	err = pm.CacheStorage.Get(ctx, cacheKey, &p)
	if err == nil {
		return
	}

	req, err := newMLRequest(predictions)
	if err != nil {
		return 0, errors.Wrap(err, "makePrediction couldn't create request")
	}

	client, err := pm.HttpClientMaker.MakeClient(ctx)
	if err != nil {
		return 0, errors.Wrap(err, "makePrediction couldn't create client")
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

	// We ignore failures in writing to cache.
	pm.CacheStorage.Set(ctx, cacheKey, &p)

	return
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
