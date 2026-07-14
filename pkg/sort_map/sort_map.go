package sort_map

//go:generate mockgen -self_package=github.com/ClessLi/component-base/pkg/sort_map -destination=mock_sort_map.go -package=sort_map github.com/ClessLi/component-base/pkg/sort_map SortMap,MapIndexes
import (
	"cmp"
	"iter"
	"sync"
)

type SortMap[K cmp.Ordered, V any] interface {
	Insert(key K, value V) error
	GetByKey(key K) (v V, ok bool)
	RemoveByKey(key K) error
	Keys() iter.Seq[K]
	Range() iter.Seq2[K, V]
	Indexes() iter.Seq[int]
	RangeWithIndex() iter.Seq2[int, V]
}

type sortMap[K cmp.Ordered, V any] struct {
	mu        *sync.RWMutex
	dataMap   map[K]V
	indexList MapIndexes[K]
}

func (s *sortMap[K, V]) Insert(key K, value V) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.dataMap[key]; !ok {
		s.indexList.Insert(key)
	}
	s.dataMap[key] = value
	return nil
}

func (s *sortMap[K, V]) GetByKey(key K) (v V, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok = s.dataMap[key]
	return
}

func (s *sortMap[K, V]) RemoveByKey(key K) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.dataMap, key)
	s.indexList.Remove(key)
	return nil
}

func (s *sortMap[K, V]) Keys() iter.Seq[K] {
	return func(yield func(K) bool) {
		s.mu.RLock()
		defer s.mu.RUnlock()
		for _, k := range s.indexList.Range() {
			if !yield(k) {
				return
			}
		}
	}
}

func (s *sortMap[K, V]) Range() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		s.mu.RLock()
		defer s.mu.RUnlock()
		for _, k := range s.indexList.Range() {
			if !yield(k, s.dataMap[k]) {
				return
			}
		}
	}
}

func (s *sortMap[K, V]) Indexes() iter.Seq[int] {
	return func(yield func(int) bool) {
		s.mu.RLock()
		defer s.mu.RUnlock()
		for i := range s.indexList.Keys() {
			if !yield(i) {
				return
			}
		}
	}
}

func (s *sortMap[K, V]) RangeWithIndex() iter.Seq2[int, V] {
	return func(yield func(int, V) bool) {
		s.mu.RLock()
		defer s.mu.RUnlock()
		for i, k := range s.indexList.Range() {
			if !yield(i, s.dataMap[k]) {
				return
			}
		}
	}
}

func Map[K cmp.Ordered, V any]() SortMap[K, V] {
	return &sortMap[K, V]{
		mu:        new(sync.RWMutex),
		dataMap:   make(map[K]V),
		indexList: NewMapIndexes[K](),
	}
}
