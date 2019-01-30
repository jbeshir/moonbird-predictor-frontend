package mlclient

import (
	"bytes"
	"context"
	"encoding/csv"
	"github.com/jbeshir/predictionbook-extractor/predictions"
	"strconv"
	"time"
)

type trainerStatus struct {
	LatestModel int64
}

type Trainer struct {
	PersistentStore  PersistentStore
	FileStore        FileStore
	PredictionSource PredictionSource
}

func (tr *Trainer) Retrain(ctx context.Context, now time.Time) error {

	// Get the current version of the model; this provides us with the path to the data it was based on,
	// and tells us what time we need to incorporate predictions from after.
	status := new(trainerStatus)
	if err := tr.PersistentStore.GetOpaque(ctx, "TrainerStatus", "status", status); err != nil {
		return err
	}

	oldPredictionFile, err := tr.FileStore.Load(ctx, strconv.FormatInt(status.LatestModel, 10)+"/summarydata-unresolved.csv")
	if err != nil {
		return err
	}

	newPredictions, err := tr.PredictionSource.AllPredictionsSince(ctx, time.Unix(status.LatestModel, 0))
	if err != nil {
		return err
	}

	csvReader := csv.NewReader(bytes.NewReader(oldPredictionFile))
	oldPredictionRecords, err := csvReader.ReadAll()
	if err != nil {
		return err
	}

	var potentiallyResolved []*predictions.PredictionSummary
	for _, prediction := range oldPredictionRecords {
		deadlineUnix, err := strconv.ParseInt(prediction[2], 10, 64)
		if err != nil {
			return err
		}

		id, err := strconv.ParseInt(prediction[0], 10, 64)
		if err != nil {
			return err
		}

		if now.After(time.Unix(deadlineUnix, 0)) {
			potentiallyResolved = append(potentiallyResolved, &predictions.PredictionSummary{
				Id: id,
			})
		}
	}
	for _, p := range newPredictions {
		if p.Outcome != predictions.Unknown {
			potentiallyResolved = append(potentiallyResolved, p)
		}
	}

	// Retrieve and save out the responses to the newly resolved predictions.
	newSummaries, responses, err := tr.PredictionSource.AllPredictionResponses(ctx, potentiallyResolved)
	if err != nil {
		return err
	}

	var resolvedSummaries []*predictions.PredictionSummary
	for _, newSummary := range newSummaries {
		if newSummary.Outcome != predictions.Unknown {
			resolvedSummaries = append(resolvedSummaries, newSummary)
		}
	}

	var buf bytes.Buffer
	csvWriter := csv.NewWriter(&buf)
	for _, r := range responses {
		var summary *predictions.PredictionSummary
		for _, candidate := range resolvedSummaries {
			if candidate.Id == r.Prediction {
				summary = candidate
				break
			}
		}
		if summary == nil {
			continue
		}

		_ = csvWriter.Write([]string{
			strconv.FormatInt(r.Prediction, 10),
			strconv.FormatInt(r.Time.Unix(), 10),
			strconv.FormatFloat(r.Confidence, 'f', -1, 64),
			r.User,
			r.Comment,
		})
	}
	csvWriter.Flush()
	err = csvWriter.Error()
	if err != nil {
		return err
	}

	err = tr.FileStore.Save(ctx, strconv.FormatInt(now.Unix(), 10)+"/responsedata.csv", buf.Bytes())
	if err != nil {
		return err
	}

	return nil
}
