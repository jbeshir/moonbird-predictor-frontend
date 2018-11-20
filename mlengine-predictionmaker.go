package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"github.com/pkg/errors"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/ml/v1"
	"google.golang.org/appengine/memcache"
	"net/http"
	"strings"
)

type MLEnginePredictionMaker struct{}

func (_ *MLEnginePredictionMaker) Predict(ctx context.Context, predictions []float64) (p float64, err error) {

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
	buf.WriteString("~")
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
