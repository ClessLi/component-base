// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package grpcserver

import (
	"strings"
	"sync"

	"github.com/ClessLi/component-base/pkg/errors"
)

// sampleKey/sampleData 是预置的示例数据，供 Download 流式 RPC 演示分块下发。
// 数据长度刻意超过默认 ChunkSize（1024 字节），以便产生多个流式分块。
const sampleKey = "greeting"

var sampleData = []byte(strings.Repeat("component-base generic gRPC server streaming demo. ", 64))

// DemoStore defines the storage operations for the demo resource.
type DemoStore interface {
	Get(key string) ([]byte, error)
	Put(key string, data []byte) error
}

// StoreFactory provides access to all store instances. It is the entry point of
// the store layer, mirroring bifrost's store/v1.StoreFactory.
type StoreFactory interface {
	Demo() DemoStore
}

// NewStoreFactory creates an in-memory StoreFactory for demonstration.
func NewStoreFactory() StoreFactory {
	s := newMemoryStore()
	_ = s.Put(sampleKey, sampleData)
	return &storeFactory{demo: s}
}

type storeFactory struct {
	demo DemoStore
}

func (f *storeFactory) Demo() DemoStore {
	return f.demo
}

// memoryStore is a thread-safe in-memory DemoStore.
type memoryStore struct {
	mu   sync.RWMutex
	data map[string][]byte
}

func newMemoryStore() *memoryStore {
	return &memoryStore{data: make(map[string][]byte)}
}

func (s *memoryStore) Get(key string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	d, ok := s.data[key]
	if !ok {
		return nil, errors.Errorf("key %q not found", key)
	}
	return d, nil
}

func (s *memoryStore) Put(key string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = data
	return nil
}
