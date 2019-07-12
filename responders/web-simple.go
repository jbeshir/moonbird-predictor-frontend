package responders

import (
	"context"
	"fmt"
	"github.com/jbeshir/moonbird-auth-frontend/ctxlogrus"
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

func (r *WebSimpleResponder) OnError(ctx context.Context, w http.ResponseWriter, err error) {
	l := ctxlogrus.Get(ctx)
	l.Error(err)

	if r.ExposeErrors {
		http.Error(w, fmt.Sprintf("Internal Server Error: %s", err), 500)
	} else {
		http.Error(w, "Internal Server Error", 500)
	}
}

func (r *WebSimpleResponder) OnSuccess(w http.ResponseWriter) {
	fmt.Fprintln(w, "Done")
}
