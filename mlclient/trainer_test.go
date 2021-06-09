package mlclient

import (
	"context"
	"encoding/json"
	"github.com/jbeshir/moonbird-auth-frontend/data"
	"github.com/jbeshir/moonbird-auth-frontend/testhelpers"
	testhelpers2 "github.com/jbeshir/moonbird-predictor-frontend/testhelpers"
	"github.com/jbeshir/predictionbook-extractor/predictions"
	"google.golang.org/api/ml/v1"
	"io/ioutil"
	"math/rand"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestTrainer_Retrain(t *testing.T) {
	t.Parallel()

	now := time.Unix(500, 0)
	step := 0

	ps := testhelpers.NewPersistentStore(t)
	ps.GetFunc = func(ctx context.Context, kind, key string, v interface{}) ([]data.Property, error) {
		wantKind := "TrainerStatus"
		if kind != wantKind {
			t.Errorf("Expected retrieval to be of kind %s, was %s", wantKind, kind)
		}

		wantKey := "status"
		if key != wantKey {
			t.Errorf("Expected retrieval to be of key %s, was %s", wantKey, key)
		}

		if status, ok := v.(*trainerStatus); !ok {
			t.Errorf("Expected output struct to be of type *trainerStatus, was not")
		} else {
			status.LatestModel = 123
		}

		if step != 0 && step != 15 {
			t.Errorf("Expected to be called at step 0 and 15, was called at step %d", step)
		}
		step++

		return nil, nil
	}
	ps.SetFunc = func(ctx context.Context, kind, key string, properties []data.Property, v interface{}) error {
		wantKind := "TrainerStatus"
		if kind != wantKind {
			t.Errorf("Expected retrieval to be of kind %s, was %s", wantKind, kind)
		}

		wantKey := "status"
		if key != wantKey {
			t.Errorf("Expected retrieval to be of key %s, was %s", wantKey, key)
		}

		if status, ok := v.(*trainerStatus); !ok {
			t.Errorf("Expected output struct to be of type *trainerStatus, was not")
		} else {
			wantModel := 500
			if status.LatestModel != 500 {
				t.Errorf("Expected new latest model to be %d, was %d", wantModel, status.LatestModel)
			}
		}

		wantStep := 16
		if step != wantStep {
			t.Errorf("Expected to be called at step %d, was called at step %d", wantStep, step)
		}
		step++

		return nil
	}
	ps.TransactFunc = func(ctx context.Context, f func(ctx context.Context) error) error {
		wantStep := 14
		if step != wantStep {
			t.Errorf("Expected to be called at step %d, was called at step %d", wantStep, step)
		}
		step++

		_ = f(ctx)

		wantStep = 17
		if step != wantStep {
			t.Errorf("Expected to be at step %d after transaction, was at step %d", wantStep, step)
		}
		step++

		return nil
	}

	fs := newTestFileStore(t)
	fs.LoadFunc = func(ctx context.Context, path string) (bytes []byte, e error) {
		wantPath := "123/summarydata-unresolved.csv"
		if wantPath != path {
			t.Errorf("Expected retrieval to be of path %s, was %s", wantPath, path)
		}

		wantStep := 1
		if step != wantStep {
			t.Errorf("Expected to be called at step %d, was called at step %d", wantStep, step)
		}
		step++

		return []byte("2,2,300,0.49,6,0,Person1,Deadline Due 1\n5,2,400,0.49,6,0,Person2,Deadline Due 2\n10,2,1000,0.96,2,0,Person3,Deadline Not Due"), nil
	}
	fs.SaveFunc = func(ctx context.Context, path string, content []byte) error {

		if step == 4 {
			wantPath := "500/responsedata.csv"
			if wantPath != path {
				t.Errorf("Expected saving to be at path %s, was %s", wantPath, path)
			}

			wantFile := "2,8,0.1,Responder1,bluh\n2,9,0.2,Responder1,\n14,11,0.4,Responder2,\n"
			if wantFile != string(content) {
				t.Errorf("Saved file content did not match expected, wanted:\n%s\n\ngot:\n\n%s", wantFile, string(content))
			}

			step++
		} else if step == 5 {
			wantPath := "500/summarydata-unresolved.csv"
			if wantPath != path {
				t.Errorf("Expected saving to be at path %s, was %s", wantPath, path)
			}

			wantFile := "5,2,600,0.3,1,0,creator2,fuzz\n7,200,3000,0,0,0,,\n10,2,1000,0.96,2,0,Person3,Deadline Not Due\n"
			if wantFile != string(content) {
				t.Errorf("Saved file content did not match expected, wanted:\n%s\n\ngot:\n\n%s", wantFile, string(content))
			}

			step++
		} else if step == 6 {
			wantPath := "500/summarydata-train.csv"
			if wantPath != path {
				t.Errorf("Expected saving to be at path %s, was %s", wantPath, path)
			}

			wantFile := "2,2,200,0.15,2,1,creator1,foo\n14,2,400,0.4,1,2,creator3,blah\n"
			if wantFile != string(content) {
				t.Errorf("Saved file content did not match expected, wanted:\n%s\n\ngot:\n\n%s", wantFile, string(content))
			}

			step++
		} else if step == 7 {
			wantPath := "500/summarydata-cv.csv"
			if wantPath != path {
				t.Errorf("Expected saving to be at path %s, was %s", wantPath, path)
			}

			wantFile := ""
			if wantFile != string(content) {
				t.Errorf("Saved file content did not match expected, wanted:\n%s\n\ngot:\n\n%s", wantFile, string(content))
			}

			step++
		} else if step == 8 {
			wantPath := "500/summarydata-test.csv"
			if wantPath != path {
				t.Errorf("Expected saving to be at path %s, was %s", wantPath, path)
			}

			wantFile := ""
			if wantFile != string(content) {
				t.Errorf("Saved file content did not match expected, wanted:\n%s\n\ngot:\n\n%s", wantFile, string(content))
			}

			step++
		}

		return nil
	}

	s := testhelpers2.NewPredictionSource(t)
	s.AllPredictionsSinceFunc = func(context context.Context, since time.Time) (summaries []*predictions.PredictionSummary, e error) {
		wantTime := time.Unix(123, 0)
		if since != wantTime {
			t.Errorf("Expected since to be called with a start time of %s, was %s", wantTime, since)
		}

		wantStep := 2
		if step != wantStep {
			t.Errorf("Expected to be called at step %d, was called at step %d", wantStep, step)
		}
		step++

		return []*predictions.PredictionSummary{
			{
				Id:       7,
				Outcome:  predictions.Unknown,
				Created:  time.Unix(200, 0),
				Deadline: time.Unix(3000, 0),
			},
			{
				Id:      14,
				Outcome: predictions.Right,
			},
		}, nil
	}

	s.AllPredictionResponsesFunc = func(ctx context.Context, summaries []*predictions.PredictionSummary) ([]*predictions.PredictionSummary, []*predictions.PredictionResponse, error) {
		wantSummaryIds := []int64{2, 5, 14}
		if len(summaries) != len(wantSummaryIds) {
			t.Errorf("Expected to be asked to retrieve %d predictions' responses, was given %d predictions", len(wantSummaryIds), len(summaries))
		} else {
			for _, s := range summaries {
				found := false
				for _, wantId := range wantSummaryIds {
					if wantId == s.Id {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Unexpected prediction response set requested; id=%d", s.Id)
				}
			}
		}

		wantStep := 3
		if step != wantStep {
			t.Errorf("Expected to be called at step %d, was called at step %d", wantStep, step)
		}
		step++

		return []*predictions.PredictionSummary{
				{
					Id:             2,
					Title:          "foo",
					Creator:        "creator1",
					Created:        time.Unix(2, 0),
					Deadline:       time.Unix(200, 0),
					MeanConfidence: 0.15,
					WagerCount:     2,
					Outcome:        predictions.Right,
				},
				{
					Id:             5,
					Title:          "fuzz",
					Creator:        "creator2",
					Created:        time.Unix(2, 0),
					Deadline:       time.Unix(600, 0),
					MeanConfidence: 0.3,
					WagerCount:     1,
					Outcome:        predictions.Unknown,
				},
				{
					Id:             14,
					Title:          "blah",
					Creator:        "creator3",
					Created:        time.Unix(2, 0),
					Deadline:       time.Unix(400, 0),
					MeanConfidence: 0.4,
					WagerCount:     1,
					Outcome:        predictions.Wrong,
				},
			}, []*predictions.PredictionResponse{
				{
					Prediction: 2,
					Time:       time.Unix(8, 0),
					Confidence: 0.1,
					User:       "Responder1",
					Comment:    "bluh",
				},
				{
					Prediction: 2,
					Time:       time.Unix(9, 0),
					Confidence: 0.2,
					User:       "Responder1",
				},
				{
					Prediction: 5,
					Time:       time.Unix(10, 0),
					Confidence: 0.3,
					User:       "Responder1",
				},
				{
					Prediction: 14,
					Time:       time.Unix(11, 0),
					Confidence: 0.4,
					User:       "Responder2",
				},
			}, nil
	}

	client := new(http.Client)
	client.Transport = &testRoundTripper{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			if step == 9 {
				wantPath := "/v1/projects/moonbird-beshir/jobs"
				if req.URL.Path != wantPath {
					t.Errorf("Wrong URL path; expected %s, got %s", wantPath, req.URL.Path)
				}

				wantHost := "ml.googleapis.com"
				if req.URL.Host != wantHost {
					t.Errorf("Wrong URL host; expected %s, got %s", wantHost, req.URL.Host)
				}

				job := new(ml.GoogleCloudMlV1__Job)
				jsonErr := json.NewDecoder(req.Body).Decode(job)
				if jsonErr != nil {
					t.Errorf("Unable to decode submitted job: %s", jsonErr)
				}

				wantJobId := "predictor_500"
				if job.JobId != wantJobId {
					t.Errorf("Expected a job ID of %s, got %s", wantJobId, job.JobId)
				}

				resp := new(http.Response)
				resp.StatusCode = 200
				resp.ContentLength = -1
				resp.Body = ioutil.NopCloser(strings.NewReader(`{}`))

				step++
				return resp, nil
			} else if step == 10 {
				wantPath := "/v1/projects/moonbird-beshir/jobs/predictor_500"
				if req.URL.Path != wantPath {
					t.Errorf("Wrong URL path; expected %s, got %s", wantPath, req.URL.Path)
				}

				wantHost := "ml.googleapis.com"
				if req.URL.Host != wantHost {
					t.Errorf("Wrong URL host; expected %s, got %s", wantHost, req.URL.Host)
				}

				wantMethod := "GET"
				if req.Method != wantMethod {
					t.Errorf("Expected method %s, got %s", wantMethod, req.Method)
				}

				resp := new(http.Response)
				resp.StatusCode = 200
				resp.ContentLength = -1
				resp.Body = ioutil.NopCloser(strings.NewReader(`{"State":"SUCCEEDED"}`))

				step++
				return resp, nil
			} else if step == 11 {
				wantPath := "/v1/projects/moonbird-beshir/models/Predictor/versions"
				if req.URL.Path != wantPath {
					t.Errorf("Wrong URL path; expected %s, got %s", wantPath, req.URL.Path)
				}

				wantHost := "ml.googleapis.com"
				if req.URL.Host != wantHost {
					t.Errorf("Wrong URL host; expected %s, got %s", wantHost, req.URL.Host)
				}

				version := new(ml.GoogleCloudMlV1__Version)
				jsonErr := json.NewDecoder(req.Body).Decode(version)
				if jsonErr != nil {
					t.Errorf("Unable to decode submitted version: %s", jsonErr)
				}

				wantName := "v500"
				if version.Name != wantName {
					t.Errorf("Expected a version name of %s, got %s", wantName, version.Name)
				}

				resp := new(http.Response)
				resp.StatusCode = 200
				resp.ContentLength = -1
				resp.Body = ioutil.NopCloser(strings.NewReader(`{}`))

				step++
				return resp, nil
			} else if step == 12 {
				wantPath := "/v1/projects/moonbird-beshir/models/Predictor/versions/v500"
				if req.URL.Path != wantPath {
					t.Errorf("Wrong URL path; expected %s, got %s", wantPath, req.URL.Path)
				}

				wantHost := "ml.googleapis.com"
				if req.URL.Host != wantHost {
					t.Errorf("Wrong URL host; expected %s, got %s", wantHost, req.URL.Host)
				}

				wantMethod := "GET"
				if req.Method != wantMethod {
					t.Errorf("Expected method %s, got %s", wantMethod, req.Method)
				}

				resp := new(http.Response)
				resp.StatusCode = 200
				resp.ContentLength = -1
				resp.Body = ioutil.NopCloser(strings.NewReader(`{"State":"READY"}`))

				step++
				return resp, nil
			} else if step == 13 {
				wantPath := "/v1/projects/moonbird-beshir/models/Predictor/versions/v500:setDefault"
				if req.URL.Path != wantPath {
					t.Errorf("Wrong URL path; expected %s, got %s", wantPath, req.URL.Path)
				}

				wantHost := "ml.googleapis.com"
				if req.URL.Host != wantHost {
					t.Errorf("Wrong URL host; expected %s, got %s", wantHost, req.URL.Host)
				}

				resp := new(http.Response)
				resp.StatusCode = 200
				resp.ContentLength = -1
				resp.Body = ioutil.NopCloser(strings.NewReader(`{}`))

				step++
				return resp, nil
			} else {
				t.Errorf("Expected to be called at step 9, 10, 11, 12, and 13, was called at step %d", step)
				step++
				return nil, nil
			}
		},
	}

	cm := newTestHttpClientMaker(t)
	cm.MakeClientFunc = func(ctx context.Context) (*http.Client, error) {
		return client, nil
	}
	ctx := context.Background()
	tr := &Trainer{
		PersistentStore:  ps,
		FileStore:        fs,
		PredictionSource: s,
		HttpClientMaker:  cm,
	}

	err := tr.Retrain(ctx, now)

	if err != nil {
		t.Errorf("Expected err to be nil, was %s", err.Error())
	}

	wantStep := 18
	if step != wantStep {
		t.Errorf("Expected to end on step %d, ended at step %d", wantStep, step)
	}
}

func TestTrainer_RetrieveNewAndOutstanding(t *testing.T) {
	t.Parallel()

	now := time.Unix(500, 0)
	step := 0

	fs := newTestFileStore(t)
	fs.LoadFunc = func(ctx context.Context, path string) (bytes []byte, e error) {
		wantPath := "123/summarydata-unresolved.csv"
		if wantPath != path {
			t.Errorf("Expected retrieval to be of path %s, was %s", wantPath, path)
		}

		wantStep := 0
		if step != wantStep {
			t.Errorf("Expected to be called at step %d, was called at step %d", wantStep, step)
		}
		step++

		return []byte("2,2,300,0.49,6,0,Person1,Deadline Due 1\n5,2,400,0.49,6,0,Person2,Deadline Due 2\n10,2,1000,0.96,2,0,Person3,Deadline Not Due"), nil
	}

	s := testhelpers2.NewPredictionSource(t)
	s.AllPredictionsSinceFunc = func(context context.Context, since time.Time) (summaries []*predictions.PredictionSummary, e error) {
		wantTime := time.Unix(123, 0)
		if since != wantTime {
			t.Errorf("Expected since to be called with a start time of %s, was %s", wantTime, since)
		}

		wantStep := 1
		if step != wantStep {
			t.Errorf("Expected to be called at step %d, was called at step %d", wantStep, step)
		}
		step++

		return []*predictions.PredictionSummary{
			{
				Id:       7,
				Outcome:  predictions.Unknown,
				Created:  time.Unix(200, 0),
				Deadline: time.Unix(3000, 0),
			},
			{
				Id:      14,
				Outcome: predictions.Right,
			},
		}, nil
	}

	ctx := context.Background()
	tr := &Trainer{
		FileStore:        fs,
		PredictionSource: s,
	}
	potentiallyResolved, unresolved, unresolvedRecords, err := tr.retrieveNewAndOutstandingPredictions(ctx, 123, now)

	if err != nil {
		t.Errorf("Expected err to be nil, was %s", err.Error())
	}

	if len(potentiallyResolved) != 3 {
		t.Errorf("Expected potentially resolved list to be of length %d, was %d", 3, len(potentiallyResolved))
	}
	if potentiallyResolved[1].Id != 2 {
		t.Errorf("Expected potentially resolved second item to be %d, was %d", 2, potentiallyResolved[1].Id)
	}

	if len(unresolved) != 1 {
		t.Errorf("Expected unresolved list to be of length %d, was %d", 1, len(unresolved))
	}
	if unresolved[0].Id != 7 {
		t.Errorf("Expected unresolved first item to be %d, was %d", 5, potentiallyResolved[1].Id)
	}

	if len(unresolvedRecords) != 1 {
		t.Errorf("Expected unresolved record list to be of length %d, was %d", 1, len(unresolvedRecords))
	}
	if strings.Join(unresolvedRecords[0], ",") != "10,2,1000,0.96,2,0,Person3,Deadline Not Due" {
		t.Errorf("Expected unresolved record to be %s, was %s", "10,2,1000,0.96,2,0,Person3,Deadline Not Due", unresolvedRecords[0])
	}

	wantStep := 2
	if step != wantStep {
		t.Errorf("Expected to end on step %d, ended at step %d", wantStep, step)
	}
}

func TestTrainer_RetrieveNewAndOutstanding_Deduplicate(t *testing.T) {
	t.Parallel()

	now := time.Unix(500, 0)
	step := 0

	fs := newTestFileStore(t)
	fs.LoadFunc = func(ctx context.Context, path string) (bytes []byte, e error) {
		wantPath := "123/summarydata-unresolved.csv"
		if wantPath != path {
			t.Errorf("Expected retrieval to be of path %s, was %s", wantPath, path)
		}

		wantStep := 0
		if step != wantStep {
			t.Errorf("Expected to be called at step %d, was called at step %d", wantStep, step)
		}
		step++

		return []byte("5,2,300,0.49,6,0,Person1,Deadline Due 1\n5,2,400,0.49,6,0,Person2,Deadline Due 2\n10,2,1000,0.96,2,0,Person3,Deadline Not Due"), nil
	}

	s := testhelpers2.NewPredictionSource(t)
	s.AllPredictionsSinceFunc = func(context context.Context, since time.Time) (summaries []*predictions.PredictionSummary, e error) {
		wantTime := time.Unix(123, 0)
		if since != wantTime {
			t.Errorf("Expected since to be called with a start time of %s, was %s", wantTime, since)
		}

		wantStep := 1
		if step != wantStep {
			t.Errorf("Expected to be called at step %d, was called at step %d", wantStep, step)
		}
		step++

		return []*predictions.PredictionSummary{
			{
				Id:       7,
				Outcome:  predictions.Unknown,
				Created:  time.Unix(200, 0),
				Deadline: time.Unix(3000, 0),
			},
			{
				Id:      5,
				Outcome: predictions.Right,
			},
		}, nil
	}

	ctx := context.Background()
	tr := &Trainer{
		FileStore:        fs,
		PredictionSource: s,
	}
	potentiallyResolved, _, _, err := tr.retrieveNewAndOutstandingPredictions(ctx, 123, now)

	if err != nil {
		t.Errorf("Expected err to be nil, was %s", err.Error())
	}

	if len(potentiallyResolved) != 1 {
		t.Errorf("Expected potentially resolved list to be of length %d, was %d", 1, len(potentiallyResolved))
	}

	wantStep := 2
	if step != wantStep {
		t.Errorf("Expected to end on step %d, ended at step %d", wantStep, step)
	}
}

func TestTrainer_DivideSummaries(t *testing.T) {
	t.Parallel()

	r := rand.NewSource(42)
	var summaries []*predictions.PredictionSummary
	for i := int64(0); i < 100; i++ {
		summaries = append(summaries, &predictions.PredictionSummary{
			Id: i,
		})
	}

	train, cv, test := divideSummaries(r, summaries)
	wantTrainLen := 60
	if len(train) != wantTrainLen {
		t.Errorf("Expected length of train set to be %d, was %d", wantTrainLen, len(train))
	}

	wantTrainFirstId := int64(0)
	if train[0].Id != wantTrainFirstId {
		t.Errorf("Expected first item train set to be %d, was %d", wantTrainFirstId, train[0].Id)
	}

	wantCvLen := 20
	if len(cv) != wantCvLen {
		t.Errorf("Expected length of train set to be %d, was %d", wantCvLen, len(cv))
	}

	wantCvFirstId := int64(3)
	if cv[0].Id != wantCvFirstId {
		t.Errorf("Expected first item cv set to be %d, was %d", wantCvFirstId, cv[0].Id)
	}

	wantTestLen := 20
	if len(test) != wantTestLen {
		t.Errorf("Expected length of train set to be %d, was %d", wantTestLen, len(test))
	}
}

func TestTrainer_UpdateLatestModel(t *testing.T) {
	t.Parallel()

	step := 0
	ps := testhelpers.NewPersistentStore(t)
	ps.GetFunc = func(ctx context.Context, kind, key string, v interface{}) ([]data.Property, error) {
		wantKind := "TrainerStatus"
		if kind != wantKind {
			t.Errorf("Expected retrieval to be of kind %s, was %s", wantKind, kind)
		}

		wantKey := "status"
		if key != wantKey {
			t.Errorf("Expected retrieval to be of key %s, was %s", wantKey, key)
		}

		if status, ok := v.(*trainerStatus); !ok {
			t.Errorf("Expected output struct to be of type *trainerStatus, was not")
		} else {
			status.LatestModel = 123
		}

		if step != 1 {
			t.Errorf("Expected to be called at step 1, was called at step %d", step)
		}
		step++

		return nil, nil
	}
	ps.SetFunc = func(ctx context.Context, kind, key string, properties []data.Property, v interface{}) error {
		wantKind := "TrainerStatus"
		if kind != wantKind {
			t.Errorf("Expected retrieval to be of kind %s, was %s", wantKind, kind)
		}

		wantKey := "status"
		if key != wantKey {
			t.Errorf("Expected retrieval to be of key %s, was %s", wantKey, key)
		}

		if status, ok := v.(*trainerStatus); !ok {
			t.Errorf("Expected output struct to be of type *trainerStatus, was not")
		} else {
			wantModel := 500
			if status.LatestModel != 500 {
				t.Errorf("Expected new latest model to be %d, was %d", wantModel, status.LatestModel)
			}
		}

		wantStep := 2
		if step != wantStep {
			t.Errorf("Expected to be called at step %d, was called at step %d", wantStep, step)
		}
		step++

		return nil
	}
	ps.TransactFunc = func(ctx context.Context, f func(ctx context.Context) error) error {
		wantStep := 0
		if step != wantStep {
			t.Errorf("Expected to be called at step %d, was called at step %d", wantStep, step)
		}
		step++

		_ = f(ctx)

		wantStep = 3
		if step != wantStep {
			t.Errorf("Expected to be at step %d after transaction, was at step %d", wantStep, step)
		}
		step++

		return nil
	}

	ctx := context.Background()
	tr := &Trainer{
		PersistentStore: ps,
	}
	err := tr.updateLatestModel(ctx, 123, 500)
	if err != nil {
		t.Errorf("Expected nil err from update, got non-nil err: %s", err)
	}

	wantStep := 4
	if step != wantStep {
		t.Errorf("Expected to end on step %d, ended at step %d", wantStep, step)
	}
}

func TestTrainer_UpdateLatestModel_Conflict(t *testing.T) {
	t.Parallel()

	step := 0
	ps := testhelpers.NewPersistentStore(t)
	ps.GetFunc = func(ctx context.Context, kind, key string, v interface{}) ([]data.Property, error) {
		wantKind := "TrainerStatus"
		if kind != wantKind {
			t.Errorf("Expected retrieval to be of kind %s, was %s", wantKind, kind)
		}

		wantKey := "status"
		if key != wantKey {
			t.Errorf("Expected retrieval to be of key %s, was %s", wantKey, key)
		}

		if status, ok := v.(*trainerStatus); !ok {
			t.Errorf("Expected output struct to be of type *trainerStatus, was not")
		} else {
			status.LatestModel = 123
		}

		if step != 1 {
			t.Errorf("Expected to be called at step 1, was called at step %d", step)
		}
		step++

		return nil, nil
	}
	ps.TransactFunc = func(ctx context.Context, f func(ctx context.Context) error) error {
		wantStep := 0
		if step != wantStep {
			t.Errorf("Expected to be called at step %d, was called at step %d", wantStep, step)
		}
		step++

		err := f(ctx)

		wantStep = 2
		if step != wantStep {
			t.Errorf("Expected to be at step %d after transaction, was at step %d", wantStep, step)
		}
		step++

		return err
	}

	ctx := context.Background()
	tr := &Trainer{
		PersistentStore: ps,
	}
	err := tr.updateLatestModel(ctx, 124, 500)
	if err == nil {
		t.Errorf("Expected non-nil err from update, got nil err")
	}

	wantStep := 3
	if step != wantStep {
		t.Errorf("Expected to end on step %d, ended at step %d", wantStep, step)
	}
}

func TestTrainer_JobSpec(t *testing.T) {
	t.Parallel()

	tr := &Trainer{
		ModelPath:    "moonbird-models/predictor",
		DataPath:     "moonbird-data/predictor",
		TrainPackage: "gs://foo/baz",
	}
	jobSpec := tr.newTrainJobSpec(123, 500)

	wantJobId := "predictor_500"
	if jobSpec.JobId != wantJobId {
		t.Errorf("Expected job ID %s, got %s", wantJobId, jobSpec.JobId)
	}

	wantJobDir := "gs://moonbird-models/predictor/500/"
	if jobSpec.TrainingInput.JobDir != wantJobDir {
		t.Errorf("Expected job dir %s, got %s", wantJobDir, jobSpec.TrainingInput.JobDir)
	}

	wantPythonModule := "trainer.train"
	if jobSpec.TrainingInput.PythonModule != wantPythonModule {
		t.Errorf("Expected python module %s, got %s", wantPythonModule, jobSpec.TrainingInput.PythonModule)
	}

	wantPythonVersion := "3.7"
	if jobSpec.TrainingInput.PythonVersion != wantPythonVersion {
		t.Errorf("Expected python version %s, got %s", wantPythonVersion, jobSpec.TrainingInput.PythonVersion)
	}

	wanRuntimeVersion := "2.4"
	if jobSpec.TrainingInput.RuntimeVersion != wanRuntimeVersion {
		t.Errorf("Expected runtime version %s, got %s", wanRuntimeVersion, jobSpec.TrainingInput.RuntimeVersion)
	}

	wantPackageUri := "gs://foo/baz"
	if jobSpec.TrainingInput.PackageUris[0] != wantPackageUri {
		t.Errorf("Expected package URI %s, got %s", wantPackageUri, jobSpec.TrainingInput.PackageUris[0])
	}

	wantRegion := "us-east1"
	if jobSpec.TrainingInput.Region != wantRegion {
		t.Errorf("Expected region %s, got %s", wantRegion, jobSpec.TrainingInput.Region)
	}

	wantArgs := []string{
		"--train-file",
		"gs://moonbird-data/predictor/500/",
		"--num-epochs",
		"1",
		"--prev-model-dir",
		"gs://moonbird-models/predictor/123/model/",
	}
	if !reflect.DeepEqual(jobSpec.TrainingInput.Args, wantArgs) {
		t.Errorf("Args attached to job did not match expected args")
	}
}

func TestTrainer_VersionSpec(t *testing.T) {
	t.Parallel()

	tr := &Trainer{
		ModelPath: "moonbird-models/predictor",
	}
	versionSpec := tr.newTrainVersionSpec(500)

	wantVersionName := "v500"
	if versionSpec.Name != wantVersionName {
		t.Errorf("Expected job ID %s, got %s", wantVersionName, versionSpec.Name)
	}

	wantDeploymentUri := "gs://moonbird-models/predictor/500/saved_model/"
	if versionSpec.DeploymentUri != wantDeploymentUri {
		t.Errorf("Expected version URI %s, got %s", wantDeploymentUri, versionSpec.DeploymentUri)
	}

	wanRuntimeVersion := "1.12"
	if versionSpec.RuntimeVersion != wanRuntimeVersion {
		t.Errorf("Expected runtime version %s, got %s", wanRuntimeVersion, versionSpec.RuntimeVersion)
	}
}

func TestTrainer_WaitForJob_Success(t *testing.T) {
	t.Parallel()

	var totalSleep time.Duration

	callCount := 0
	client := &http.Client{
		Transport: &testRoundTripper{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
				wantPath := "/v1/projects/moonbird-beshir/jobs/predictor_500"
				if req.URL.Path != wantPath {
					t.Errorf("Wrong URL path; expected %s, got %s", wantPath, req.URL.Path)
				}

				wantHost := "ml.googleapis.com"
				if req.URL.Host != wantHost {
					t.Errorf("Wrong URL host; expected %s, got %s", wantHost, req.URL.Host)
				}

				wantMethod := "GET"
				if req.Method != wantMethod {
					t.Errorf("Expected method %s, got %s", wantMethod, req.Method)
				}

				resp := new(http.Response)
				resp.StatusCode = 200
				resp.ContentLength = -1
				if totalSleep < 4*time.Second {
					resp.Body = ioutil.NopCloser(strings.NewReader(`{"State":"RUNNING"}`))
				} else {
					resp.Body = ioutil.NopCloser(strings.NewReader(`{"State":"SUCCEEDED"}`))
				}
				callCount++

				return resp, nil
			},
		},
	}

	tr := &Trainer{
		SleepFunc: func(d time.Duration) {
			wantSleep := 500 * time.Millisecond
			if d != wantSleep {
				t.Errorf("Wrong sleep duration; expected %d, got %d", wantSleep, d)
			}
			totalSleep += d
		},
	}

	err := tr.waitForTrainJob("predictor_500", client)
	if err != nil {
		t.Errorf("Expected nil err from update, got non-nil err: %s", err)
	}

	wantCallCount := 9
	if callCount != wantCallCount {
		t.Errorf("Expected call count before returning %d, got %d", wantCallCount, callCount)
	}
}

func TestTrainer_WaitForJob_Failed(t *testing.T) {
	t.Parallel()

	var totalSleep time.Duration

	callCount := 0
	client := &http.Client{
		Transport: &testRoundTripper{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
				wantPath := "/v1/projects/moonbird-beshir/jobs/predictor_500"
				if req.URL.Path != wantPath {
					t.Errorf("Wrong URL path; expected %s, got %s", wantPath, req.URL.Path)
				}

				wantHost := "ml.googleapis.com"
				if req.URL.Host != wantHost {
					t.Errorf("Wrong URL host; expected %s, got %s", wantHost, req.URL.Host)
				}

				wantMethod := "GET"
				if req.Method != wantMethod {
					t.Errorf("Expected method %s, got %s", wantMethod, req.Method)
				}

				resp := new(http.Response)
				resp.StatusCode = 200
				resp.ContentLength = -1
				if totalSleep < 4*time.Second {
					resp.Body = ioutil.NopCloser(strings.NewReader(`{"State":"RUNNING"}`))
				} else {
					resp.Body = ioutil.NopCloser(strings.NewReader(`{"State":"FAILED"}`))
				}
				callCount++

				return resp, nil
			},
		},
	}

	tr := &Trainer{
		SleepFunc: func(d time.Duration) {
			wantSleep := 500 * time.Millisecond
			if d != wantSleep {
				t.Errorf("Wrong sleep duration; expected %d, got %d", wantSleep, d)
			}
			totalSleep += d
		},
	}

	err := tr.waitForTrainJob("predictor_500", client)
	if err == nil {
		t.Errorf("Expected non-nil err from update, got nil")
	}

	wantCallCount := 9
	if callCount != wantCallCount {
		t.Errorf("Expected call count before returning %d, got %d", wantCallCount, callCount)
	}
}

func TestTrainer_WaitForJob_Cancelled(t *testing.T) {
	t.Parallel()

	var totalSleep time.Duration

	callCount := 0
	client := &http.Client{
		Transport: &testRoundTripper{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
				wantPath := "/v1/projects/moonbird-beshir/jobs/predictor_500"
				if req.URL.Path != wantPath {
					t.Errorf("Wrong URL path; expected %s, got %s", wantPath, req.URL.Path)
				}

				wantHost := "ml.googleapis.com"
				if req.URL.Host != wantHost {
					t.Errorf("Wrong URL host; expected %s, got %s", wantHost, req.URL.Host)
				}

				wantMethod := "GET"
				if req.Method != wantMethod {
					t.Errorf("Expected method %s, got %s", wantMethod, req.Method)
				}

				resp := new(http.Response)
				resp.StatusCode = 200
				resp.ContentLength = -1
				if totalSleep < 4*time.Second {
					resp.Body = ioutil.NopCloser(strings.NewReader(`{"State":"RUNNING"}`))
				} else {
					resp.Body = ioutil.NopCloser(strings.NewReader(`{"State":"CANCELLED"}`))
				}
				callCount++

				return resp, nil
			},
		},
	}

	tr := &Trainer{
		SleepFunc: func(d time.Duration) {
			wantSleep := 500 * time.Millisecond
			if d != wantSleep {
				t.Errorf("Wrong sleep duration; expected %d, got %d", wantSleep, d)
			}
			totalSleep += d
		},
	}

	err := tr.waitForTrainJob("predictor_500", client)
	if err == nil {
		t.Errorf("Expected non-nil err from update, got nil")
	}

	wantCallCount := 9
	if callCount != wantCallCount {
		t.Errorf("Expected call count before returning %d, got %d", wantCallCount, callCount)
	}
}

func TestTrainer_WaitForVersion_Success(t *testing.T) {
	t.Parallel()

	var totalSleep time.Duration

	callCount := 0
	client := &http.Client{
		Transport: &testRoundTripper{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
				wantPath := "/v1/projects/moonbird-beshir/models/Predictor/versions/v500"
				if req.URL.Path != wantPath {
					t.Errorf("Wrong URL path; expected %s, got %s", wantPath, req.URL.Path)
				}

				wantHost := "ml.googleapis.com"
				if req.URL.Host != wantHost {
					t.Errorf("Wrong URL host; expected %s, got %s", wantHost, req.URL.Host)
				}

				wantMethod := "GET"
				if req.Method != wantMethod {
					t.Errorf("Expected method %s, got %s", wantMethod, req.Method)
				}

				resp := new(http.Response)
				resp.StatusCode = 200
				resp.ContentLength = -1
				if totalSleep < 4*time.Second {
					resp.Body = ioutil.NopCloser(strings.NewReader(`{"State":"CREATING"}`))
				} else {
					resp.Body = ioutil.NopCloser(strings.NewReader(`{"State":"READY"}`))
				}
				callCount++

				return resp, nil
			},
		},
	}

	tr := &Trainer{
		SleepFunc: func(d time.Duration) {
			wantSleep := 500 * time.Millisecond
			if d != wantSleep {
				t.Errorf("Wrong sleep duration; expected %d, got %d", wantSleep, d)
			}
			totalSleep += d
		},
	}

	err := tr.waitForVersionReady("v500", client)
	if err != nil {
		t.Errorf("Expected nil err from update, got %s", err)
	}

	wantCallCount := 9
	if callCount != wantCallCount {
		t.Errorf("Expected call count before returning %d, got %d", wantCallCount, callCount)
	}
}

func TestTrainer_WaitForVersion_Failed(t *testing.T) {
	t.Parallel()

	var totalSleep time.Duration

	callCount := 0
	client := &http.Client{
		Transport: &testRoundTripper{
			RoundTripFunc: func(req *http.Request) (*http.Response, error) {
				wantPath := "/v1/projects/moonbird-beshir/models/Predictor/versions/v500"
				if req.URL.Path != wantPath {
					t.Errorf("Wrong URL path; expected %s, got %s", wantPath, req.URL.Path)
				}

				wantHost := "ml.googleapis.com"
				if req.URL.Host != wantHost {
					t.Errorf("Wrong URL host; expected %s, got %s", wantHost, req.URL.Host)
				}

				wantMethod := "GET"
				if req.Method != wantMethod {
					t.Errorf("Expected method %s, got %s", wantMethod, req.Method)
				}

				resp := new(http.Response)
				resp.StatusCode = 200
				resp.ContentLength = -1
				if totalSleep < 4*time.Second {
					resp.Body = ioutil.NopCloser(strings.NewReader(`{"State":"CREATING"}`))
				} else {
					resp.Body = ioutil.NopCloser(strings.NewReader(`{"State":"FAILED"}`))
				}
				callCount++

				return resp, nil
			},
		},
	}

	tr := &Trainer{
		SleepFunc: func(d time.Duration) {
			wantSleep := 500 * time.Millisecond
			if d != wantSleep {
				t.Errorf("Wrong sleep duration; expected %d, got %d", wantSleep, d)
			}
			totalSleep += d
		},
	}

	err := tr.waitForVersionReady("v500", client)
	if err == nil {
		t.Errorf("Expected non-nil err from update, got nil")
	}

	wantCallCount := 9
	if callCount != wantCallCount {
		t.Errorf("Expected call count before returning %d, got %d", wantCallCount, callCount)
	}
}
