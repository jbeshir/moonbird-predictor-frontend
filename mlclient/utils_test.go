package mlclient

import (
	"context"
	"net/http"
	"testing"
)

type testFileStore struct {
	LoadFunc func(ctx context.Context, path string) ([]byte, error)
	SaveFunc func(ctx context.Context, path string, content []byte) error
}

func newTestFileStore(t *testing.T) *testFileStore {
	return &testFileStore{
		LoadFunc: func(ctx context.Context, path string) (bytes []byte, e error) {
			t.Error("Load should not be called")
			return nil, nil
		},
		SaveFunc: func(ctx context.Context, path string, content []byte) error {
			t.Error("Save should not be called")
			return nil
		},
	}
}

func (fs *testFileStore) Load(ctx context.Context, path string) ([]byte, error) {
	return fs.LoadFunc(ctx, path)
}

func (fs *testFileStore) Save(ctx context.Context, path string, content []byte) error {
	return fs.SaveFunc(ctx, path, content)
}

type testHttpClientMaker struct {
	MakeClientFunc func(ctx context.Context) (*http.Client, error)
}

func newTestHttpClientMaker(t *testing.T) *testHttpClientMaker {
	return &testHttpClientMaker{
		MakeClientFunc: func(ctx context.Context) (*http.Client, error) {
			t.Error("MakeClient should not be called")
			return nil, nil
		},
	}
}

func (cm *testHttpClientMaker) MakeClient(ctx context.Context) (*http.Client, error) {
	return cm.MakeClientFunc(ctx)
}

type testRoundTripper struct {
	RoundTripFunc func(*http.Request) (*http.Response, error)
}

func (rt *testRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return rt.RoundTripFunc(r)
}
