package mlclient

import (
	"context"
	"github.com/jbeshir/moonbird-auth-frontend/testhelpers"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestNewMLRequest(t *testing.T) {
	t.Parallel()

	mlRequest, err := newMLRequest([]float64{0.4, 0.1})
	if err != nil {
		t.Errorf("Unexpected error from newMlRequest: %s", err)
		return
	}

	if mlRequest.HttpBody.ContentType != "application/json" {
		t.Errorf("Unexpected request content type; expected %s, was %s",
			"application/json", mlRequest.HttpBody.ContentType)
	}

	if mlRequest.HttpBody.Data != `{"instances":[{"input":[[0.4],[0.1]]}]}` {
		t.Errorf("Incorrect request body; expected `%s`, was `%s`",
			`{"instances":[{"input":[[0.4],[0.1]]}]}`, mlRequest.HttpBody.Data)
	}
}

func TestNewMLRequest_OutOfRange(t *testing.T) {
	t.Parallel()

	mlRequest, err := newMLRequest([]float64{1.4, 0.1})
	if mlRequest != nil {
		t.Errorf("Expected nil request, got request")
		return
	}
	if err == nil {
		t.Errorf("Expected error, got nil error")
		return
	}
}

func TestGeneratePredictionCacheKey(t *testing.T) {
	t.Parallel()

	key := generatePredictionCacheKey([]float64{0.4, 0.1})
	expectedKey := "P9mZmZmZmZo/uZmZmZmZmg=="
	if key != expectedKey {
		t.Errorf("Incorrect generated prediction cache key; expected %v, was %v", expectedKey, key)
	}
}

func TestPredictionMaker_Predict_OutOfRange(t *testing.T) {
	t.Parallel()

	cs := testhelpers.NewCacheStore(t)
	cm := newTestHttpClientMaker(t)
	pm := &PredictionMaker{
		CacheStorage:    cs,
		HttpClientMaker: cm,
	}

	c := context.Background()
	result, err := pm.Predict(c, []float64{0.4, -0.1})
	if err == nil {
		t.Errorf("Expected error, got nil error")
	}
	if result != 0 {
		t.Errorf("Unexpected prediction result; expected %g, was %g", 0.0, result)
	}
}

func TestPredictionMaker_Predict_FromCache(t *testing.T) {
	t.Parallel()

	cs := testhelpers.NewCacheStore(t)
	cm := newTestHttpClientMaker(t)
	pm := &PredictionMaker{
		CacheStorage:    cs,
		HttpClientMaker: cm,
	}

	expectedCacheKey := generatePredictionCacheKey([]float64{0.4, 0.1})
	cs.GetFunc = func(ctx context.Context, key string, v interface{}) error {
		if key != expectedCacheKey {
			t.Errorf("Reading from wrong cache key; expected %x, was %x", expectedCacheKey, key)
		}

		result, valid := v.(*float64)
		if !valid {
			t.Errorf("Cache reading wrong type; expected *float64")
		} else {
			*result = 0.3
		}

		return nil
	}

	c := context.Background()
	result, err := pm.Predict(c, []float64{0.4, 0.1})
	if err != nil {
		t.Errorf("Unexpected error from Predict: %s", err)
	}
	if result != 0.3 {
		t.Errorf("Incorrect prediction result; expected %g, was %g", 0.3, result)
	}
}

func TestPredictionMaker_Predict_FromMLEngine(t *testing.T) {
	t.Parallel()

	cs, pm := predictionMaker_Predict_FromMLEngineSetup(t, func(r *http.Request) (*http.Response, error) {
		body, _ := ioutil.ReadAll(r.Body)
		if r.Method != "POST" {
			t.Errorf("Incorrect request method, expected %s, was %s", "POST", r.Method)
		}
		if r.URL.String() != "https://ml.googleapis.com/v1/projects/moonbird-beshir/models/Predictor:predict?alt=json&prettyPrint=false" {
			t.Errorf("Incorrect request URL, expected %s, was %s", "https://ml.googleapis.com/v1/projects/moonbird-beshir/models/Predictor:predict?alt=json&prettyPrint=false", r.URL.String())
		}
		if !reflect.DeepEqual(body, []byte(`{"instances":[{"input":[[0.4],[0.1]]}]}`)) {
			t.Errorf("Incorrect request body, expected `%s`, was `%s`", `{"instances":[{"input":[[0.4],[0.1]]}]}`, body)
		}

		resp := new(http.Response)
		resp.StatusCode = 200
		resp.ContentLength = -1
		resp.Body = ioutil.NopCloser(strings.NewReader(`{"predictions":[{"income":[0.3]}]}`))
		return resp, nil
	})

	expectedCacheKey := generatePredictionCacheKey([]float64{0.4, 0.1})
	cs.SetFunc = func(ctx context.Context, key string, v interface{}) error {
		if key != expectedCacheKey {
			t.Errorf("Reading from wrong cache key; expected %x, was %x", expectedCacheKey, key)
		}

		result, valid := v.(*float64)
		if !valid {
			t.Errorf("Cache writing wrong type; expected *float64")
		} else {
			if *result != 0.3 {
				t.Errorf("Cache writing wrong result; expected %g, was %g", 0.3, *result)
			}
		}

		return nil
	}

	c := context.Background()
	result, err := pm.Predict(c, []float64{0.4, 0.1})
	if err != nil {
		t.Errorf("Unexpected error from Predict: %s", err)
	}
	if result != 0.3 {
		t.Errorf("Incorrect prediction result; expected %g, was %g", 0.3, result)
	}
}

func TestPredictionMaker_Predict_FromMLEngineReqErr(t *testing.T) {
	t.Parallel()

	_, pm := predictionMaker_Predict_FromMLEngineSetup(t, func(r *http.Request) (*http.Response, error) {
		return nil, errors.New("nope")
	})

	c := context.Background()
	result, err := pm.Predict(c, []float64{0.4, 0.1})
	if err == nil {
		t.Errorf("Expected error from Predict, got nil")
	}
	if result != 0 {
		t.Errorf("Unexpected prediction result; expected %g, was %g", 0.0, result)
	}
}

func TestPredictionMaker_Predict_FromMLEngineStatusCodeErr(t *testing.T) {
	t.Parallel()

	_, pm := predictionMaker_Predict_FromMLEngineSetup(t, func(r *http.Request) (*http.Response, error) {
		resp := new(http.Response)
		resp.StatusCode = 401
		resp.ContentLength = -1
		resp.Body = ioutil.NopCloser(strings.NewReader(`{"predictions":[{"income":[0.3]}]}`))
		return resp, nil
	})

	c := context.Background()
	result, err := pm.Predict(c, []float64{0.4, 0.1})
	if err == nil {
		t.Errorf("Expected error from Predict, got nil")
	}
	if result != 0 {
		t.Errorf("Unexpected prediction result; expected %g, was %g", 0.0, result)
	}
}

func TestPredictionMaker_Predict_FromMLEngineStatusCodeNotJsonErr(t *testing.T) {
	t.Parallel()

	_, pm := predictionMaker_Predict_FromMLEngineSetup(t, func(r *http.Request) (*http.Response, error) {
		resp := new(http.Response)
		resp.StatusCode = 200
		resp.ContentLength = -1
		resp.Body = ioutil.NopCloser(strings.NewReader(`nope`))
		return resp, nil
	})

	c := context.Background()
	result, err := pm.Predict(c, []float64{0.4, 0.1})
	if err == nil {
		t.Errorf("Expected error from Predict, got nil")
	}
	if result != 0 {
		t.Errorf("Unexpected prediction result; expected %g, was %g", 0.0, result)
	}
}

func TestPredictionMaker_Predict_FromMLEngineStatusCodeWrongPredictionCountErr(t *testing.T) {
	t.Parallel()

	_, pm := predictionMaker_Predict_FromMLEngineSetup(t, func(r *http.Request) (*http.Response, error) {
		resp := new(http.Response)
		resp.StatusCode = 200
		resp.ContentLength = -1
		resp.Body = ioutil.NopCloser(strings.NewReader(`{"predictions":[{"income":[0.3]},{"income":[0.7]}]}`))
		return resp, nil
	})

	c := context.Background()
	result, err := pm.Predict(c, []float64{0.4, 0.1})
	if err == nil {
		t.Errorf("Expected error from Predict, got nil")
	}
	if result != 0 {
		t.Errorf("Unexpected prediction result; expected %g, was %g", 0.0, result)
	}
}

func TestPredictionMaker_Predict_FromMLEngineStatusCodeWrongIncomeCountErr(t *testing.T) {
	t.Parallel()

	_, pm := predictionMaker_Predict_FromMLEngineSetup(t, func(r *http.Request) (*http.Response, error) {
		resp := new(http.Response)
		resp.StatusCode = 200
		resp.ContentLength = -1
		resp.Body = ioutil.NopCloser(strings.NewReader(`{"predictions":[{"income":[0.3, 0.1]}]}`))
		return resp, nil
	})

	c := context.Background()
	result, err := pm.Predict(c, []float64{0.4, 0.1})
	if err == nil {
		t.Errorf("Expected error from Predict, got nil")
	}
	if result != 0 {
		t.Errorf("Unexpected prediction result; expected %g, was %g", 0.0, result)
	}
}

func predictionMaker_Predict_FromMLEngineSetup(t *testing.T, roundTripFunc func(*http.Request) (*http.Response, error)) (*testhelpers.CacheStore, *PredictionMaker) {
	cs := testhelpers.NewCacheStore(t)
	cm := newTestHttpClientMaker(t)
	pm := &PredictionMaker{
		CacheStorage:    cs,
		HttpClientMaker: cm,
	}

	cm.MakeClientFunc = func(ctx context.Context) (*http.Client, error) {
		client := new(http.Client)
		client.Transport = &testRoundTripper{
			RoundTripFunc: roundTripFunc,
		}
		return client, nil
	}

	cs.GetFunc = func(ctx context.Context, key string, v interface{}) error {
		return errors.New("nope")
	}

	return cs, pm
}
