package sort_map

import (
	"cmp"
	"encoding/json"
	"fmt"
	"iter"
	"sync"

	"github.com/ClessLi/component-base/pkg/errors"
	"gopkg.in/yaml.v3"
)

// SortMap is the exported concrete ordered-map type. Business structs should use it (or a
// pointer to it) directly as a field type: yaml.v3 only dispatches UnmarshalYAML on
// addressable concrete values and never through interface-typed fields, so wrapping these
// containers behind an interface field cannot work with yaml.v3 (see
// docs/guide/zh-CN/libs/sort_map.md for details and migration notes).
//
// The zero value of SortMap is an empty, ready-to-use container: its internal state is
// lazily initialized on the first write or decode. When starting from a zero value, the
// container must be initialized (write or decode) before it is accessed concurrently.
type SortMap[K cmp.Ordered, V any] struct {
	mu        *sync.RWMutex
	dataMap   map[K]V
	indexList MapIndexes[K]
}

// sortMapInitMu guards lazy field initialization of zero-value containers so that the
// initialization path itself is race free for concurrent first writes.
var sortMapInitMu sync.Mutex

func newSortMap[K cmp.Ordered, V any]() SortMap[K, V] {
	return SortMap[K, V]{
		mu:        new(sync.RWMutex),
		dataMap:   make(map[K]V),
		indexList: NewMapIndexes[K](),
	}
}

// ensure lazily initializes the internal state. Once initialized, the mu/dataMap/indexList
// field references are never reassigned (contents are mutated in place under s.mu), so the
// unlocked nil checks in the read paths stay race free for properly synchronized use.
func (s *SortMap[K, V]) ensure() {
	sortMapInitMu.Lock()
	defer sortMapInitMu.Unlock()
	if s.mu == nil {
		s.mu = new(sync.RWMutex)
	}
	if s.dataMap == nil {
		s.dataMap = make(map[K]V)
	}
	if s.indexList == nil {
		s.indexList = NewMapIndexes[K]()
	}
}

func (s *SortMap[K, V]) Set(key K, value V) (oldValue V, present bool) {
	s.ensure()
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.setLocked(key, value)
}

// setLocked writes key/value without locking. The caller must hold s.mu's write lock.
func (s *SortMap[K, V]) setLocked(key K, value V) (oldValue V, present bool) {
	oldValue, present = s.dataMap[key]
	s.dataMap[key] = value
	s.indexList.Insert(key)
	return
}

func (s *SortMap[K, V]) Get(key K) (value V, present bool) {
	if s.mu == nil {
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, present = s.dataMap[key]
	return
}

func (s *SortMap[K, V]) Contains(key K) (present bool) {
	if s.mu == nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, present = s.dataMap[key]
	return
}

func (s *SortMap[K, V]) RemoveByKey(key K) {
	if s.mu == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.dataMap, key)
	s.indexList.Remove(key)
}

func (s *SortMap[K, V]) Clear() {
	s.ensure()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resetLocked()
}

func (s *SortMap[K, V]) Len() int {
	if s.mu == nil || s.indexList == nil {
		return 0
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.indexList.Len()
}

func (s *SortMap[K, V]) Keys() iter.Seq[K] {
	return func(yield func(K) bool) {
		if s.mu == nil || s.indexList == nil {
			return
		}
		s.mu.RLock()
		defer s.mu.RUnlock()
		for _, k := range s.indexList.Range() {
			if !yield(k) {
				return
			}
		}
	}
}

func (s *SortMap[K, V]) Values() iter.Seq[V] {
	return func(yield func(V) bool) {
		if s.mu == nil || s.indexList == nil {
			return
		}
		s.mu.RLock()
		defer s.mu.RUnlock()
		for _, k := range s.indexList.Range() {
			if !yield(s.dataMap[k]) {
				return
			}
		}
	}
}

func (s *SortMap[K, V]) Range() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		if s.mu == nil || s.indexList == nil {
			return
		}
		s.mu.RLock()
		defer s.mu.RUnlock()
		for _, k := range s.indexList.Range() {
			if !yield(k, s.dataMap[k]) {
				return
			}
		}
	}
}

func (s *SortMap[K, V]) Indexes() iter.Seq[int] {
	return func(yield func(int) bool) {
		if s.mu == nil || s.indexList == nil {
			return
		}
		s.mu.RLock()
		defer s.mu.RUnlock()
		for i := range s.indexList.Keys() {
			if !yield(i) {
				return
			}
		}
	}
}

func (s *SortMap[K, V]) RangeWithIndex() iter.Seq2[int, V] {
	return func(yield func(int, V) bool) {
		if s.mu == nil || s.indexList == nil {
			return
		}
		s.mu.RLock()
		defer s.mu.RUnlock()
		for i, k := range s.indexList.Range() {
			if !yield(i, s.dataMap[k]) {
				return
			}
		}
	}
}

// resetLocked empties the container in place. The caller must hold s.mu's write lock.
// Field references stay stable (dataMap is cleared key by key, the concrete index slice is
// truncated) so that unlocked nil checks elsewhere never observe a torn initialization.
func (s *SortMap[K, V]) resetLocked() {
	for k := range s.dataMap {
		delete(s.dataMap, k)
	}
	if mi, ok := s.indexList.(*mapIndexes[K]); ok {
		*mi = (*mi)[:0]
		return
	}
	s.indexList = NewMapIndexes[K]()
}

// replaceLocked resets the container and fills it with data. A nil data yields an empty
// container. The caller must hold s.mu's write lock.
func (s *SortMap[K, V]) replaceLocked(data map[K]V) {
	s.resetLocked()
	for k, v := range data {
		s.dataMap[k] = v
		s.indexList.Insert(k)
	}
}

func (s SortMap[K, V]) MarshalJSON() ([]byte, error) {
	if s.mu == nil || s.dataMap == nil {
		return []byte("{}"), nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return json.Marshal(s.dataMap)
}

func (s *SortMap[K, V]) UnmarshalJSON(bytes []byte) error {
	var data map[K]V
	if err := json.Unmarshal(bytes, &data); err != nil {
		return err
	}

	s.ensure()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.replaceLocked(data)

	return nil
}

// MarshalYAML uses a value receiver on purpose: the yaml.v3 encoder does not fall back to
// Addr() when probing the Marshaler interface, so a pointer-receiver MarshalYAML would be
// silently skipped for non-addressable value-typed struct fields (rendered as an empty
// mapping instead). mu is a pointer, so copying the receiver does not copy the lock.
func (s SortMap[K, V]) MarshalYAML() (interface{}, error) {
	if s.mu == nil || s.dataMap == nil {
		return map[K]V{}, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dataMap, nil
}

func (s *SortMap[K, V]) UnmarshalYAML(value *yaml.Node) error {
	if value == nil {
		return errors.Errorf("sort_map: nil YAML node when unmarshaling into %T", s)
	}

	node := resolveYAMLNode(value)

	var data map[K]V
	switch {
	case node == nil || isNullYAMLNode(node):
		// null/empty nodes reset the container to an empty map
	case node.Kind == yaml.MappingNode:
		if err := node.Decode(&data); err != nil {
			return err
		}
	default:
		return errors.Errorf("sort_map: expecting a YAML mapping node, but got %s when unmarshaling into %T",
			yamlNodeKindName(node.Kind), s)
	}

	s.ensure()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.replaceLocked(data)

	return nil
}

// resolveYAMLNode unwraps document nodes and follows alias nodes to their target so that
// custom unmarshalers receive the effective content node regardless of entry point.
func resolveYAMLNode(n *yaml.Node) *yaml.Node {
	for i := 0; n != nil && i < 8; i++ {
		switch n.Kind {
		case yaml.DocumentNode:
			if len(n.Content) == 0 {
				return nil
			}
			n = n.Content[0]
		case yaml.AliasNode:
			if n.Alias == nil {
				return n
			}
			n = n.Alias
		default:
			return n
		}
	}
	return n
}

func isNullYAMLNode(n *yaml.Node) bool {
	return n == nil || n.Kind == 0 || (n.Kind == yaml.ScalarNode && n.Tag == "!!null")
}

func yamlNodeKindName(kind yaml.Kind) string {
	switch kind {
	case yaml.DocumentNode:
		return "document"
	case yaml.SequenceNode:
		return "sequence"
	case yaml.MappingNode:
		return "mapping"
	case yaml.ScalarNode:
		return "scalar"
	case yaml.AliasNode:
		return "alias"
	default:
		return fmt.Sprintf("unknown kind %d", kind)
	}
}

func Map[K cmp.Ordered, V any]() SortMap[K, V] {
	return newSortMap[K, V]()
}
