package mlclient

import (
	"bytes"
	"context"
	"encoding/csv"
	"github.com/jbeshir/moonbird-auth-frontend/ctxlogrus"
	"github.com/jbeshir/predictionbook-extractor/predictions"
	"github.com/pkg/errors"
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
	l := ctxlogrus.Get(ctx)

	newModel := now.Unix()
	newModelStr := strconv.FormatInt(newModel, 10)

	client, err := tr.HttpClientMaker.MakeClient(ctx)
	if err != nil {
		return errors.Wrap(err, "")
	}

	// Get the current version of the model; this provides us with the path to the data it was based on,
	// and tells us what time we need to incorporate predictions from after.
	status := new(trainerStatus)
	if _, err := tr.PersistentStore.Get(ctx, "TrainerStatus", "status", status); err != nil {
		return errors.Wrap(err, "")
	}

	potentiallyResolved, unresolved, unresolvedRecords, err := tr.retrieveNewAndOutstandingPredictions(ctx, status.LatestModel, now)
	l.Infof("Have %d potentially resolved, %d unresolved, and %d existing not due predictions",
		len(potentiallyResolved), len(unresolved), len(unresolvedRecords))

	// Retrieve and save out the responses to the newly resolved predictions.
	l.Infof("Retrieving prediction responses and status for %d potentially resolved predictions",
		len(potentiallyResolved))
	newSummaries, responses, err := tr.PredictionSource.AllPredictionResponses(ctx, potentiallyResolved)
	if err != nil {
		return errors.Wrap(err, "")
	}

	l.Info("Sorting potentially resolved into newly resolved and still unresolved predictions...")
	var resolvedSummaries []*predictions.PredictionSummary
	for _, newSummary := range newSummaries {
		if newSummary.Outcome != predictions.Unknown {
			resolvedSummaries = append(resolvedSummaries, newSummary)
		} else {
			unresolved = append(unresolved, newSummary)
		}
	}
	l.Infof("Now have %d newly resolved and %d still unresolved predictions", len(resolvedSummaries),
		len(unresolved))

	l.Info("Writing resolved prediction responses to CSV...")
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
		return errors.Wrap(err, "")
	}

	err = tr.FileStore.Save(ctx, strconv.FormatInt(now.Unix(), 10)+"/responsedata.csv", buf.Bytes())
	if err != nil {
		return errors.Wrap(err, "")
	}

	unresolvedRecords = append(unresolvedRecords, tr.generateSummaryRecords(unresolved)...)
	sort.Slice(unresolvedRecords, func(i, j int) bool {
		iId, _ := strconv.ParseInt(unresolvedRecords[i][0], 10, 64)
		jId, _ := strconv.ParseInt(unresolvedRecords[j][0], 10, 64)
		return iId < jId
	})

	l.Infof("Writing %d total known outstanding prediction summaries to CSV...", len(unresolvedRecords))
	err = tr.writeCsv(ctx, newModelStr+"/summarydata-unresolved.csv", unresolvedRecords)
	if err != nil {
		return errors.Wrap(err, "")
	}

	train, cv, test := divideSummaries(rand.New(rand.NewSource(time.Now().Unix())), resolvedSummaries)
	l.Infof("Split newly resolved predictions into %d train, %d cv, %d test", len(train), len(cv), len(test))

	err = tr.writeCsv(ctx, newModelStr+"/summarydata-train.csv", tr.generateSummaryRecords(train))
	if err != nil {
		return errors.Wrap(err, "")
	}
	err = tr.writeCsv(ctx, newModelStr+"/summarydata-cv.csv", tr.generateSummaryRecords(cv))
	if err != nil {
		return errors.Wrap(err, "")
	}
	err = tr.writeCsv(ctx, newModelStr+"/summarydata-test.csv", tr.generateSummaryRecords(test))
	if err != nil {
		return errors.Wrap(err, "")
	}

	mlService, err := ml.New(client)
	if err != nil {
		return errors.Wrap(err, "")
	}

	l.Info("Launching training job...")
	createCall := mlService.Projects.Jobs.Create("projects/moonbird-beshir", tr.newTrainJobSpec(status.LatestModel, newModel))
	_, err = createCall.Do()
	if err != nil {
		return errors.Wrap(err, "")
	}

	l.Info("Waiting for training job...")
	err = tr.waitForTrainJob("predictor_"+strconv.FormatInt(newModel, 10), client)
	if err != nil {
		return errors.Wrap(err, "")
	}

	l.Info("Creating new version...")
	versionCall := mlService.Projects.Models.Versions.Create("projects/moonbird-beshir/models/Predictor", tr.newTrainVersionSpec(newModel))
	_, err = versionCall.Do()
	if err != nil {
		return errors.Wrap(err, "")
	}

	l.Info("Waiting for new version to be ready...")
	err = tr.waitForVersionReady("v"+strconv.FormatInt(newModel, 10), client)
	if err != nil {
		return errors.Wrap(err, "")
	}

	l.Info("Setting new version as default...")
	versionDefaultCall := mlService.Projects.Models.Versions.SetDefault("projects/moonbird-beshir/models/Predictor/versions/v"+strconv.FormatInt(newModel, 10),
		&ml.GoogleCloudMlV1__SetDefaultVersionRequest{})
	_, err = versionDefaultCall.Do()
	if err != nil {
		return errors.Wrap(err, "")
	}

	l.Infof("Updating latest model version to %d", newModel)
	err = tr.updateLatestModel(ctx, status.LatestModel, newModel)
	if err != nil {
		return errors.Wrap(err, "")
	}

	return nil
}

func (tr *Trainer) retrieveNewAndOutstandingPredictions(ctx context.Context, prevModel int64, now time.Time) (potentiallyResolved []*predictions.PredictionSummary, unresolved []*predictions.PredictionSummary, unresolvedRecords [][]string, err error) {
	l := ctxlogrus.Get(ctx)

	oldPredictionFile, err := tr.FileStore.Load(ctx, strconv.FormatInt(prevModel, 10)+"/summarydata-unresolved.csv")
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "")
	}

	l.Debugf("Retrieving new predictions from source since %d", prevModel)
	newPredictions, err := tr.PredictionSource.AllPredictionsSince(ctx, time.Unix(prevModel, 0))
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "")
	}

	l.Debug("Parsing old outstanding predictions...")
	csvReader := csv.NewReader(bytes.NewReader(oldPredictionFile))
	oldPredictionRecords, err := csvReader.ReadAll()
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "")
	}

	l.Debug("Sorting potentially resolved and unresolved predictions apart...")
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
			return nil, nil, nil, errors.Wrap(err, "")
		}

		id, err := strconv.ParseInt(prediction[0], 10, 64)
		if err != nil {
			return nil, nil, nil, errors.Wrap(err, "")
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
		if _, err := tr.PersistentStore.Get(ctx, "TrainerStatus", "status", status); err != nil {
			return errors.Wrap(err, "")
		}
		if status.LatestModel != oldModel {
			return errors.New("concurrent latest model update")
		}

		status.LatestModel = newModel
		if err := tr.PersistentStore.Set(ctx, "TrainerStatus", "status", nil, status); err != nil {
			return errors.Wrap(err, "")
		}

		return nil
	})
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
			return errors.Wrap(err, "")
		}
	}
	csvWriter.Flush()
	err := csvWriter.Error()
	if err != nil {
		return errors.Wrap(err, "")
	}

	err = tr.FileStore.Save(ctx, path, buf.Bytes())
	if err != nil {
		return errors.Wrap(err, "")
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
			Region: "us-east1",
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
		return errors.Wrap(err, "")
	}

	for {
		jobCall := mlService.Projects.Jobs.Get("projects/moonbird-beshir/jobs/" + jobID)
		job, err := jobCall.Do()
		if err != nil {
			return errors.Wrap(err, "")
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
		return errors.Wrap(err, "")
	}

	for {
		versionCall := mlService.Projects.Models.Versions.Get("projects/moonbird-beshir/models/Predictor/versions/" + version)
		version, err := versionCall.Do()
		if err != nil {
			return errors.Wrap(err, "")
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
