package sort_map

//go:generate mockgen -self_package=github.com/ClessLi/component-base/pkg/sort_map -destination=mock_sort_map.go -package=sort_map github.com/ClessLi/component-base/pkg/sort_map SortMap,MapList,MapIndexes
import (
	"cmp"
	"encoding/json"
	"iter"
	"sync"

	"github.com/marmotedu/errors"
	"gopkg.in/yaml.v3"
)

type SortMap[K cmp.Ordered, V any] interface {
	Set(key K, value V) (oldValue V, present bool)
	Get(key K) (value V, present bool)
	RemoveByKey(key K)
	Keys() iter.Seq[K]
	Range() iter.Seq2[K, V]
	Indexes() iter.Seq[int]
	RangeWithIndex() iter.Seq2[int, V]
	json.Unmarshaler
	json.Marshaler
	yaml.Unmarshaler
	yaml.Marshaler
}

type sortMap[K cmp.Ordered, V any] struct {
	mu        *sync.RWMutex
	dataMap   map[K]V
	indexList MapIndexes[K]
}

func (s *sortMap[K, V]) Set(key K, value V) (oldValue V, present bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldValue, present = s.dataMap[key]
	s.dataMap[key] = value
	s.indexList.Insert(key)
	return
}

func (s *sortMap[K, V]) Get(key K) (value V, present bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, present = s.dataMap[key]
	return
}

func (s *sortMap[K, V]) RemoveByKey(key K) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.dataMap, key)
	s.indexList.Remove(key)
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

func (s *sortMap[K, V]) UnmarshalJSON(bytes []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var data map[K]V
	if err := json.Unmarshal(bytes, &data); err != nil {
		return err
	}

	s.dataMap = data
	s.indexList = NewMapIndexes[K]()
	for k := range data {
		s.indexList.Insert(k)
	}

	return nil
}

func (s *sortMap[K, V]) MarshalJSON() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return json.Marshal(s.dataMap)
}

func (s *sortMap[K, V]) UnmarshalYAML(value *yaml.Node) error {
	if value == nil {
		return errors.New("nil yaml node")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var data map[K]V
	if err := value.Decode(&data); err != nil {
		return err
	}

	s.dataMap = data
	s.indexList = NewMapIndexes[K]()
	for k := range data {
		s.indexList.Insert(k)
	}

	return nil
}

func (s *sortMap[K, V]) MarshalYAML() (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.dataMap, nil
}

func Map[K cmp.Ordered, V any]() SortMap[K, V] {
	return &sortMap[K, V]{
		mu:        new(sync.RWMutex),
		dataMap:   make(map[K]V),
		indexList: NewMapIndexes[K](),
	}
}
