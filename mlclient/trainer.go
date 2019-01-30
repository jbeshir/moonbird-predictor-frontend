package mlclient

import (
	"bytes"
	"context"
	"encoding/csv"
	"github.com/jbeshir/predictionbook-extractor/predictions"
	"sort"
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
	newModel := now.Unix()
	newModelStr := strconv.FormatInt(newModel, 10)

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

	var unresolved []*predictions.PredictionSummary
	var unresolvedRecords [][]string
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
		} else {
			unresolvedRecords = append(unresolvedRecords, prediction)
		}
	}
	for _, p := range newPredictions {
		if p.Outcome != predictions.Unknown {
			potentiallyResolved = append(potentiallyResolved, p)
		} else {
			unresolved = append(unresolved, p)
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
		} else {
			unresolved = append(unresolved, newSummary)
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

	unresolvedRecords = append(unresolvedRecords, tr.generateSummaryRecords(unresolved)...)
	sort.Slice(unresolvedRecords, func(i, j int) bool {
		iId, _ := strconv.ParseInt(unresolvedRecords[i][0], 10, 64)
		jId, _ := strconv.ParseInt(unresolvedRecords[j][0], 10, 64)
		return iId < jId
	})

	err = tr.writeCsv(ctx, newModelStr+"/summarydata-unresolved.csv", unresolvedRecords)
	if err != nil {
		return err
	}

	trainRecords := tr.generateSummaryRecords(resolvedSummaries)
	err = tr.writeCsv(ctx, newModelStr+"/summarydata-train.csv", trainRecords)
	if err != nil {
		return err
	}
	err = tr.writeCsv(ctx, newModelStr+"/summarydata-cv.csv", nil)
	if err != nil {
		return err
	}
	err = tr.writeCsv(ctx, newModelStr+"/summarydata-test.csv", nil)
	if err != nil {
		return err
	}

	err = tr.PersistentStore.Transact(ctx, func(ctx context.Context) error {
		status := new(trainerStatus)
		if err := tr.PersistentStore.GetOpaque(ctx, "TrainerStatus", "status", status); err != nil {
			return err
		}

		status.LatestModel = newModel
		if err := tr.PersistentStore.SetOpaque(ctx, "TrainerStatus", "status", status); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (tr *Trainer) generateSummaryRecords(summaries []*predictions.PredictionSummary) (records [][]string) {
	for _, p := range summaries {
		records = append(records, []string{
			strconv.FormatInt(p.Id, 10),
			strconv.FormatInt(p.Created.Unix(), 10),
			strconv.FormatInt(p.Deadline.Unix(), 10),
			strconv.FormatFloat(p.MeanConfidence, 'f', -1, 64),
			strconv.FormatInt(p.WagerCount, 10),
			strconv.FormatInt(int64(p.Outcome), 10),
			p.Creator,
			p.Title,
		})
	}

	return
}

func (tr *Trainer) writeCsv(ctx context.Context, path string, records [][]string) error {
	var buf bytes.Buffer
	csvWriter := csv.NewWriter(&buf)
	for _, r := range records {

		err := csvWriter.Write(r)
		if err != nil {
			return err
		}
	}
	csvWriter.Flush()
	err := csvWriter.Error()
	if err != nil {
		return err
	}

	err = tr.FileStore.Save(ctx, path, buf.Bytes())
	if err != nil {
		return err
	}

	return nil
}
