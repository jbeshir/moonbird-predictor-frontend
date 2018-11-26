package mlclient

import (
	"context"
	"net/http"
	"testing"
)

type testCacheStore struct {
	GetFunc    func(ctx context.Context, key string, v interface{}) error
	SetFunc    func(ctx context.Context, key string, v interface{}) error
	DeleteFunc func(ctx context.Context, key string) error
}

func newTestCacheStore(t *testing.T) *testCacheStore {
	return &testCacheStore{
		GetFunc: func(ctx context.Context, key string, v interface{}) error {
			t.Error("Get should not be called")
			return nil
		},
		SetFunc: func(ctx context.Context, key string, v interface{}) error {
			t.Error("Set should not be called")
			return nil
		},
		DeleteFunc: func(ctx context.Context, key string) error {
			t.Error("Delete should not be called")
			return nil
		},
	}
}

func (cs *testCacheStore) Get(ctx context.Context, key string, v interface{}) error {
	return cs.GetFunc(ctx, key, v)
}

func (cs *testCacheStore) Set(ctx context.Context, key string, v interface{}) error {
	return cs.SetFunc(ctx, key, v)
}

func (cs *testCacheStore) Delete(ctx context.Context, key string) error {
	return cs.DeleteFunc(ctx, key)
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
