package pbook

import (
	"context"
	"testing"
)

type testCacheStore struct {
	GetFunc func(ctx context.Context, key string, v interface{}) error
	SetFunc func(ctx context.Context, key string, v interface{}) error
	DeleteFunc func(ctx context.Context, key string) error
}

func newTestCacheStore(t *testing.T) *testCacheStore {
	return &testCacheStore{
		GetFunc: func(ctx context.Context, key string, v interface{}) error {
			t.Error("cs.Get should not be called")
			return nil
		},
		SetFunc: func(ctx context.Context, key string, v interface{}) error {
			t.Error("cs.Set should not be called")
			return nil
		},
		DeleteFunc: func(ctx context.Context, key string) error {
			t.Error("cs.Delete should not be called")
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

type testPersistentStore struct {
	GetOpaqueFunc func(ctx context.Context, kind, key string, v interface{}) error
	SetOpaqueFunc func(ctx context.Context, kind, key string, v interface{}) error
}

func newTestPersistentStore(t *testing.T) *testPersistentStore {
	return &testPersistentStore{
		GetOpaqueFunc: func(ctx context.Context, kind, key string, v interface{}) error {
			t.Error("ps.GetOpaque should not be called")
			return nil
		},
		SetOpaqueFunc: func(ctx context.Context, kind, key string, v interface{}) error {
			t.Error("ps.SetOpaque should not be called")
			return nil
		},
	}
}

func (ps *testPersistentStore) GetOpaque(ctx context.Context, kind, key string, v interface{}) error {
	return ps.GetOpaqueFunc(ctx, kind, key, v)
}

func (ps *testPersistentStore) SetOpaque(ctx context.Context, kind, key string, v interface{}) error {
	return ps.SetOpaqueFunc(ctx, kind, key, v)
}
