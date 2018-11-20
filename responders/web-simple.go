package responders

import (
	"fmt"
	"net/http"
)

type WebSimpleResponder struct{}

func (_ *WebSimpleResponder) OnContextError(w http.ResponseWriter, err error) {
	http.Error(w, "Internal Server Error", 500)
}

func (_ *WebSimpleResponder) OnError(w http.ResponseWriter, err error) {
	http.Error(w, "Internal Server Error", 500)
}

func (_ *WebSimpleResponder) OnSuccess(w http.ResponseWriter) {
	fmt.Fprintln(w, "Done")
}
