// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package apiserver

import (
	"strconv"
	"sync"

	apiv1 "github.com/ClessLi/component-base/examples/apiserver/api/v1"
	"github.com/ClessLi/component-base/pkg/errors"
)

// UserStore defines the storage operations for the user resource.
type UserStore interface {
	Get(id string) (*apiv1.User, error)
	List() ([]*apiv1.User, error)
	Create(name string) (*apiv1.User, error)
}

// StoreFactory provides access to all store instances. It is the entry point of
// the store layer, mirroring gjallarhorn's store/v1.Factory.
type StoreFactory interface {
	Users() UserStore
}

// NewStoreFactory creates an in-memory StoreFactory for demonstration.
func NewStoreFactory() StoreFactory {
	return &storeFactory{users: newMemoryUserStore()}
}

type storeFactory struct {
	users UserStore
}

func (f *storeFactory) Users() UserStore {
	return f.users
}

// memoryUserStore is a thread-safe in-memory UserStore.
type memoryUserStore struct {
	mu    sync.RWMutex
	seq   int
	users map[string]*apiv1.User
	order []string
}

func newMemoryUserStore() *memoryUserStore {
	return &memoryUserStore{users: make(map[string]*apiv1.User)}
}

func (s *memoryUserStore) Get(id string) (*apiv1.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.users[id]
	if !ok {
		return nil, errors.Errorf("user %s not found", id)
	}
	return u, nil
}

func (s *memoryUserStore) List() ([]*apiv1.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*apiv1.User, 0, len(s.order))
	for _, id := range s.order {
		out = append(out, s.users[id])
	}
	return out, nil
}

func (s *memoryUserStore) Create(name string) (*apiv1.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.seq++
	u := &apiv1.User{ID: strconv.Itoa(s.seq), Name: name}
	s.users[u.ID] = u
	s.order = append(s.order, u.ID)
	return u, nil
}
