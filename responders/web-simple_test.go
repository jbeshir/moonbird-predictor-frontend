package responders

import (
	"errors"
	"io/ioutil"
	"net/http/httptest"
	"testing"
)

func TestWebSimpleResponder_OnContextError(t *testing.T) {

	r := &WebSimpleResponder{
		ExposeErrors: false,
	}

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

func TestWebSimpleResponder_OnError(t *testing.T) {

	r := &WebSimpleResponder{
		ExposeErrors: false,
	}

	recorder := httptest.NewRecorder()
	r.OnError(recorder, errors.New("bluh"))

	result := recorder.Result()
	if result.StatusCode != 500 {
		t.Errorf("Expected a status code of 500, got %d", result.StatusCode)
	}

	content, _ := ioutil.ReadAll(result.Body)
	if string(content) != "Internal Server Error\n" {
		t.Errorf("Expected a body of 'Internal Server Error\n', got '%s'", content)
	}
}

func TestWebSimpleResponder_OnSuccess(t *testing.T) {

	r := &WebSimpleResponder{
		ExposeErrors: false,
	}

	recorder := httptest.NewRecorder()
	r.OnSuccess(recorder)

	result := recorder.Result()
	if result.StatusCode != 200 {
		t.Errorf("Expected a status code of 200, got %d", result.StatusCode)
	}

	content, _ := ioutil.ReadAll(result.Body)
	if string(content) != "Done\n" {
		t.Errorf("Expected a body of 'Done\n', got '%s'", content)
	}
}