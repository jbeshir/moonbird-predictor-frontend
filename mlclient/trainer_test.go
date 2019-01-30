package mlclient

import (
	"context"
	"github.com/jbeshir/moonbird-predictor-frontend/testhelpers"
	"github.com/jbeshir/predictionbook-extractor/predictions"
	"testing"
	"time"
)

func TestTrainer_Retrain(t *testing.T) {

	now := time.Unix(500, 0)
	step := 0

	ps := testhelpers.NewPersistentStore(t)
	ps.GetOpaqueFunc = func(ctx context.Context, kind, key string, v interface{}) error {
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

		if step != 0 && step != 10 {
			t.Errorf("Expected to be called at step 0 and 10, was called at step %d", step)
		}
		step++

		return nil
	}
	ps.SetOpaqueFunc = func(ctx context.Context, kind, key string, v interface{}) error {
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

		wantStep := 11
		if step != wantStep {
			t.Errorf("Expected to be called at step %d, was called at step %d", wantStep, step)
		}
		step++

		return nil
	}
	ps.TransactFunc = func(ctx context.Context, f func(ctx context.Context) error) error {
		wantStep := 9
		if step != wantStep {
			t.Errorf("Expected to be called at step %d, was called at step %d", wantStep, step)
		}
		step++

		f(ctx)

		wantStep = 12
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

	s := testhelpers.NewPredictionSource(t)
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

	ctx := context.Background()
	tr := &Trainer{
		PersistentStore:  ps,
		FileStore:        fs,
		PredictionSource: s,
	}
	err := tr.Retrain(ctx, now)

	if err != nil {
		t.Errorf("Expected err to be nil, was %s", err.Error())
	}

	wantStep := 13
	if step != wantStep {
		t.Errorf("Expected to end on step %d, ended at step %d", wantStep, step)
	}
}
