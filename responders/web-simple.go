package responders

import (
	"fmt"
	"net/http"
)

type WebSimpleResponder struct {
	ExposeErrors bool
}

func (r *WebSimpleResponder) OnContextError(w http.ResponseWriter, err error) {
	if r.ExposeErrors {
		http.Error(w, fmt.Sprintf("Internal Server Error: %s", err), 500)
	} else {
		http.Error(w, "Internal Server Error", 500)
	}
}

func (r *WebSimpleResponder) OnError(w http.ResponseWriter, err error) {
	if r.ExposeErrors {
		http.Error(w, fmt.Sprintf("Internal Server Error: %s", err), 500)
	} else {
		http.Error(w, "Internal Server Error", 500)
	}
}

func (r *WebSimpleResponder) OnSuccess(w http.ResponseWriter) {
	fmt.Fprintln(w, "Done")
}
