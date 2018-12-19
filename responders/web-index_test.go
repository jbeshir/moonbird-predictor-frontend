package responders

import (
	"errors"
	"github.com/PuerkitoBio/goquery"
	"github.com/jbeshir/moonbird-predictor-frontend/controllers"
	"golang.org/x/net/html"
	"io/ioutil"
	"net/http/httptest"
	"testing"
)

func TestWebIndexResponder_OnContextError(t *testing.T) {

	r := &WebIndexResponder{}

	recorder := httptest.NewRecorder()
	r.OnContextError(recorder, errors.New("bluh"))

	result := recorder.Result()
	if result.StatusCode != 500 {
		t.Errorf("Expected a status code of 500, got %d", result.StatusCode)
	}

	content, _ := ioutil.ReadAll(result.Body)
	if string(content) != "Internal Server Error\n" {
		t.Errorf("Expected a body of 'Internal Server Error\n', got '%s'", content)
	}
}

func TestWebIndexResponder_OnResult(t *testing.T) {

	r := &WebIndexResponder{}

	indexResult := &controllers.IndexResult{}

	recorder := httptest.NewRecorder()
	r.OnResult(recorder, indexResult)

	result := recorder.Result()
	if result.StatusCode != 200 {
		t.Errorf("Expected a status code of 200, got %d", result.StatusCode)
	}

	pageHtml, _ := html.Parse(result.Body)
	page := goquery.NewDocumentFromNode(pageHtml)

	predictionInputs := len(page.Find(".prediction-text-input").Nodes)
	if predictionInputs != 1 {
		t.Errorf("Expected page to contain 1 prediction input, found %d", predictionInputs)
	}

	predictionInputValue, _ := page.Find(".prediction-text-input").Attr("value")
	if predictionInputValue != "" {
		t.Errorf("Expected page to contain empty prediction input, contained %s", predictionInputValue)
	}

	predictionResults := len(page.Find(".prediction-result").Nodes)
	if predictionResults != 0 {
		t.Errorf("Expected page to contain 0 prediction results, found %d", predictionResults)
	}

	predictionFaults := len(page.Find("#prediction-fault").Nodes)
	if predictionFaults != 0 {
		t.Errorf("Expected page to contain 0 prediction faults, found %d", predictionFaults)
	}

	exampleLists := len(page.Find(".example-list").Nodes)
	if exampleLists != 0 {
		t.Errorf("Expected page to contain 0 example lists, found %d", exampleLists)
	}

	exampleListFaults := len(page.Find(".example-list-fault-msg").Nodes)
	if exampleListFaults != 0 {
		t.Errorf("Expected page to contain 0 example list faults, found %d", exampleListFaults)
	}
}

func TestWebIndexResponder_OnResult_Prediction(t *testing.T) {

	r := &WebIndexResponder{}

	indexResult := &controllers.IndexResult{
		AssignmentsStr: "0.1, 0.2",
		Prediction: new(float64),
	}
	*indexResult.Prediction = 0.17

	recorder := httptest.NewRecorder()
	r.OnResult(recorder, indexResult)

	result := recorder.Result()
	if result.StatusCode != 200 {
		t.Errorf("Expected a status code of 200, got %d", result.StatusCode)
	}

	pageHtml, _ := html.Parse(result.Body)
	page := goquery.NewDocumentFromNode(pageHtml)

	predictionInputs := len(page.Find(".prediction-text-input").Nodes)
	if predictionInputs != 1 {
		t.Errorf("Expected page to contain 1 prediction input, found %d", predictionInputs)
	}

	predictionInputValue, _ := page.Find(".prediction-text-input").Attr("value")
	if predictionInputValue != "0.1, 0.2" {
		t.Errorf("Expected page to contain '0.1, 0.2' prediction input, contained %s", predictionInputValue)
	}

	predictionResults := len(page.Find(".prediction-result").Nodes)
	if predictionResults != 1 {
		t.Errorf("Expected page to contain 1 prediction result, found %d", predictionResults)
	}
	predictionResultValue, _ := page.Find(".prediction-result").Html()
	if predictionResultValue != "0.17" {
		t.Errorf("Expected page to contain '0.17' prediction result, contained %s", predictionResultValue)
	}
}

func TestWebIndexResponder_OnResult_PredictionErr(t *testing.T) {

	r := &WebIndexResponder{}

	indexResult := &controllers.IndexResult{
		AssignmentsStr: "foo",
		PredictionErr: errors.New("bluh"),
	}

	recorder := httptest.NewRecorder()
	r.OnResult(recorder, indexResult)

	result := recorder.Result()
	if result.StatusCode != 200 {
		t.Errorf("Expected a status code of 200, got %d", result.StatusCode)
	}

	pageHtml, _ := html.Parse(result.Body)
	page := goquery.NewDocumentFromNode(pageHtml)

	predictionInputs := len(page.Find(".prediction-text-input").Nodes)
	if predictionInputs != 1 {
		t.Errorf("Expected page to contain 1 prediction input, found %d", predictionInputs)
	}

	predictionInputValue, _ := page.Find(".prediction-text-input").Attr("value")
	if predictionInputValue != "foo" {
		t.Errorf("Expected page to contain 'foo' prediction input, contained %s", predictionInputValue)
	}

	predictionFaults := len(page.Find("#prediction-fault").Nodes)
	if predictionFaults != 1 {
		t.Errorf("Expected page to contain 1 prediction fault, found %d", predictionFaults)
	}

	predictionFaultValue, _ := page.Find("#prediction-fault").Html()
	if predictionFaultValue != "bluh" {
		t.Errorf("Expected page to contain 'bluh' prediction fault, contained %s", predictionFaultValue)
	}
}

func TestWebIndexResponder_OnResult_ExampleList(t *testing.T) {

	r := &WebIndexResponder{}

	indexResult := &controllers.IndexResult{
		ExampleList: "bluh",
	}

	recorder := httptest.NewRecorder()
	r.OnResult(recorder, indexResult)

	result := recorder.Result()
	if result.StatusCode != 200 {
		t.Errorf("Expected a status code of 200, got %d", result.StatusCode)
	}

	pageHtml, _ := html.Parse(result.Body)
	page := goquery.NewDocumentFromNode(pageHtml)

	exampleLists := len(page.Find(".example-list").Nodes)
	if exampleLists != 1 {
		t.Errorf("Expected page to contain 1 example list, found %d", exampleLists)
	}
}

func TestWebIndexResponder_OnResult_ExampleListErr(t *testing.T) {

	r := &WebIndexResponder{}

	indexResult := &controllers.IndexResult{
		ExampleListErr: errors.New("bluh"),
	}

	recorder := httptest.NewRecorder()
	r.OnResult(recorder, indexResult)

	result := recorder.Result()
	if result.StatusCode != 200 {
		t.Errorf("Expected a status code of 200, got %d", result.StatusCode)
	}

	pageHtml, _ := html.Parse(result.Body)
	page := goquery.NewDocumentFromNode(pageHtml)

	exampleListFaults := len(page.Find(".example-list-fault-msg").Nodes)
	if exampleListFaults != 1 {
		t.Errorf("Expected page to contain 1 example list fault fault, found %d", exampleListFaults)
	}

	exampleListFaultValue, _ := page.Find(".example-list-fault-msg").Html()
	if exampleListFaultValue != "bluh" {
		t.Errorf("Expected page to contain 'bluh' example list fault, contained %s", exampleListFaultValue)
	}
}