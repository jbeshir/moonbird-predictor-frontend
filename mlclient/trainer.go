package mlclient

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"github.com/jbeshir/predictionbook-extractor/predictions"
	"google.golang.org/api/ml/v1"
	"math/rand"
	"net/http"
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
	ModelPath        string
	DataPath         string
	TrainPackage     string
	SleepFunc        func(time.Duration)
	HttpClientMaker  HttpClientMaker
}

func (tr *Trainer) Retrain(ctx context.Context, now time.Time) error {
	newModel := now.Unix()
	newModelStr := strconv.FormatInt(newModel, 10)

	client, err := tr.HttpClientMaker.MakeClient(ctx)
	if err != nil {
		return err
	}

	// Get the current version of the model; this provides us with the path to the data it was based on,
	// and tells us what time we need to incorporate predictions from after.
	status := new(trainerStatus)
	if err := tr.PersistentStore.GetOpaque(ctx, "TrainerStatus", "status", status); err != nil {
		return err
	}

	potentiallyResolved, unresolved, unresolvedRecords, err := tr.retrieveNewAndOutstandingPredictions(ctx, status.LatestModel, now)

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

	train, cv, test := divideSummaries(rand.New(rand.NewSource(time.Now().Unix())), resolvedSummaries)
	err = tr.writeCsv(ctx, newModelStr+"/summarydata-train.csv", tr.generateSummaryRecords(train))
	if err != nil {
		return err
	}
	err = tr.writeCsv(ctx, newModelStr+"/summarydata-cv.csv", tr.generateSummaryRecords(cv))
	if err != nil {
		return err
	}
	err = tr.writeCsv(ctx, newModelStr+"/summarydata-test.csv", tr.generateSummaryRecords(test))
	if err != nil {
		return err
	}

	mlService, err := ml.New(client)
	if err != nil {
		return err
	}

	createCall := mlService.Projects.Jobs.Create("", tr.newTrainJobSpec(status.LatestModel, newModel))
	_, err = createCall.Do()
	if err != nil {
		return err
	}

	err = tr.waitForTrainJob("predictor_"+strconv.FormatInt(newModel, 10), client)
	if err != nil {
		return err
	}

	versionCall := mlService.Projects.Models.Versions.Create("projects/Moonbird/models/Predictor", tr.newTrainVersionSpec(newModel))
	_, err = versionCall.Do()
	if err != nil {
		return err
	}

	err = tr.waitForVersionReady("v"+strconv.FormatInt(newModel, 10), client)
	if err != nil {
		return err
	}

	versionDefaultCall := mlService.Projects.Models.Versions.SetDefault("projects/Moonbird/models/Predictor/versions/v"+strconv.FormatInt(newModel, 10),
		&ml.GoogleCloudMlV1__SetDefaultVersionRequest{})
	_, err = versionDefaultCall.Do()
	if err != nil {
		return err
	}

	err = tr.updateLatestModel(ctx, status.LatestModel, newModel)
	if err != nil {
		return err
	}

	return nil
}

func (tr *Trainer) retrieveNewAndOutstandingPredictions(ctx context.Context, prevModel int64, now time.Time) (potentiallyResolved []*predictions.PredictionSummary, unresolved []*predictions.PredictionSummary, unresolvedRecords [][]string, err error) {
	oldPredictionFile, err := tr.FileStore.Load(ctx, strconv.FormatInt(prevModel, 10)+"/summarydata-unresolved.csv")
	if err != nil {
		return nil, nil, nil, err
	}

	newPredictions, err := tr.PredictionSource.AllPredictionsSince(ctx, time.Unix(prevModel, 0))
	if err != nil {
		return nil, nil, nil, err
	}

	csvReader := csv.NewReader(bytes.NewReader(oldPredictionFile))
	oldPredictionRecords, err := csvReader.ReadAll()
	if err != nil {
		return nil, nil, nil, err
	}

	potentiallyResolvedIds := make(map[int64]struct{})
	for _, p := range newPredictions {
		if p.Outcome != predictions.Unknown {
			_, exists := potentiallyResolvedIds[p.Id]
			if !exists {
				potentiallyResolved = append(potentiallyResolved, p)
				potentiallyResolvedIds[p.Id] = struct{}{}
			}
		} else {
			unresolved = append(unresolved, p)
		}
	}

	for _, prediction := range oldPredictionRecords {
		deadlineUnix, err := strconv.ParseInt(prediction[2], 10, 64)
		if err != nil {
			return nil, nil, nil, err
		}

		id, err := strconv.ParseInt(prediction[0], 10, 64)
		if err != nil {
			return nil, nil, nil, err
		}

		if now.After(time.Unix(deadlineUnix, 0)) {
			_, exists := potentiallyResolvedIds[id]
			if !exists {
				potentiallyResolved = append(potentiallyResolved, &predictions.PredictionSummary{
					Id: id,
				})
				potentiallyResolvedIds[id] = struct{}{}
			}
		} else {
			unresolvedRecords = append(unresolvedRecords, prediction)
		}
	}

	return potentiallyResolved, unresolved, unresolvedRecords, nil
}

func (tr *Trainer) updateLatestModel(ctx context.Context, oldModel, newModel int64) error {
	return tr.PersistentStore.Transact(ctx, func(ctx context.Context) error {
		status := new(trainerStatus)
		if err := tr.PersistentStore.GetOpaque(ctx, "TrainerStatus", "status", status); err != nil {
			return err
		}
		if status.LatestModel != oldModel {
			return errors.New("concurrent latest model update")
		}

		status.LatestModel = newModel
		if err := tr.PersistentStore.SetOpaque(ctx, "TrainerStatus", "status", status); err != nil {
			return err
		}

		return nil
	})
}

func divideSummaries(rs rand.Source, summaries []*predictions.PredictionSummary) (train, cv, test []*predictions.PredictionSummary) {

	cvSize := int(float64(len(summaries)) * 0.2)
	testSize := cvSize
	trainSize := len(summaries) - cvSize - testSize

	summaries = append([]*predictions.PredictionSummary{}, summaries...)
	r := rand.New(rs)
	r.Shuffle(len(summaries), func(i, j int) { summaries[i], summaries[j] = summaries[j], summaries[i] })

	train = summaries[:trainSize]
	sort.Slice(train, func(i, j int) bool {
		return train[i].Id < train[j].Id
	})

	cv = summaries[trainSize : trainSize+cvSize]
	sort.Slice(cv, func(i, j int) bool {
		return cv[i].Id < cv[j].Id
	})

	test = summaries[trainSize+cvSize:]
	sort.Slice(test, func(i, j int) bool {
		return test[i].Id < test[j].Id
	})

	return train, cv, test
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

func (tr *Trainer) newTrainJobSpec(oldModel, newModel int64) *ml.GoogleCloudMlV1__Job {
	return &ml.GoogleCloudMlV1__Job{
		JobId: "predictor_" + strconv.FormatInt(newModel, 10),
		TrainingInput: &ml.GoogleCloudMlV1__TrainingInput{
			JobDir:         "gs://" + tr.ModelPath + "/" + strconv.FormatInt(newModel, 10) + "/",
			PythonModule:   "trainer.train",
			PythonVersion:  "3.5",
			RuntimeVersion: "1.12",
			Args: []string{
				"--train-file",
				"gs://" + tr.DataPath + "/" + strconv.FormatInt(newModel, 10) + "/",
				"--num-epochs",
				"1",
				"--prev-model-dir",
				"gs://" + tr.ModelPath + "/" + strconv.FormatInt(oldModel, 10) + "/model/",
			},
			PackageUris: []string{
				tr.TrainPackage,
			},
		},
	}
}

func (tr *Trainer) newTrainVersionSpec(model int64) *ml.GoogleCloudMlV1__Version {
	return &ml.GoogleCloudMlV1__Version{
		Name:           "v" + strconv.FormatInt(model, 10),
		DeploymentUri:  "gs://" + tr.ModelPath + "/" + strconv.FormatInt(model, 10) + "/saved_model/",
		RuntimeVersion: "1.12",
	}
}

func (tr *Trainer) waitForTrainJob(jobID string, client *http.Client) error {

	mlService, err := ml.New(client)
	if err != nil {
		return err
	}

	for {
		jobCall := mlService.Projects.Jobs.Get(jobID)
		job, err := jobCall.Do()
		if err != nil {
			return err
		}
		if job.State == "FAILED" {
			return errors.New("job failed")
		}
		if job.State == "CANCELLED" {
			return errors.New("job cancelled")
		}
		if job.State == "SUCCEEDED" {
			return nil
		}

		tr.SleepFunc(500 * time.Millisecond)
	}
}

func (tr *Trainer) waitForVersionReady(version string, client *http.Client) error {

	mlService, err := ml.New(client)
	if err != nil {
		return err
	}

	for {
		versionCall := mlService.Projects.Models.Versions.Get("projects/Moonbird/models/Predictor/versions/" + version)
		version, err := versionCall.Do()
		if err != nil {
			return err
		}
		if version.State == "FAILED" {
			return errors.New("job failed")
		}
		if version.State == "READY" {
			return nil
		}

		tr.SleepFunc(500 * time.Millisecond)
	}
}
