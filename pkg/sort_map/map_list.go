package sort_map

import (
	"encoding/json"

	"github.com/ClessLi/component-base/pkg/errors"
	"gopkg.in/yaml.v3"
)

// MapList is the exported concrete sparse list type. Like SortMap it is exported so that
// business structs can use it (or a pointer to it) directly as a field type, which is
// the only way yaml.v3 can dispatch UnmarshalYAML for nested struct fields.
//
// Get/Contains/Len/Clear/Range/Indexes and the value-receiver MarshalJSON/MarshalYAML are
// inherited from the embedded SortMap; the value receivers are what allow the yaml.v3/json
// encoders to find the Marshaler on non-addressable value-typed struct fields.
//
// The zero value of MapList is an empty, ready-to-use list; when starting from a zero
// value, the list must be initialized (write or decode) before it is accessed concurrently.
type MapList[V any] struct {
	SortMap[int, V]
}

func (m *MapList[V]) Set(index int, value V) (oldValue V, present bool) {
	if index < 0 {
		return
	}
	return m.SortMap.Set(index, value)
}

func (m *MapList[V]) InsertToEmpty(value V) int {
	s := &m.SortMap
	s.ensure()
	s.mu.Lock()
	defer s.mu.Unlock()
	index := bSearchFirstFreeIndex(s.indexList.(*mapIndexes[int]))
	s.setLocked(index, value)
	return index
}

func (m *MapList[V]) Append(value V) int {
	s := &m.SortMap
	s.ensure()
	s.mu.Lock()
	defer s.mu.Unlock()
	index := lastFreeIndex(s.indexList.(*mapIndexes[int]))
	s.setLocked(index, value)
	return index
}

func (m *MapList[V]) Remove(index int) {
	if index < 0 {
		return
	}
	m.SortMap.RemoveByKey(index)
}

func (m *MapList[V]) UnmarshalJSON(bytes []byte) error {
	var data map[int]V
	if err := json.Unmarshal(bytes, &data); err != nil {
		return err
	}

	if err := checkNonNegativeIndexes(data); err != nil {
		return err
	}

	s := &m.SortMap
	s.ensure()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.replaceLocked(data)

	return nil
}

// UnmarshalYAML decodes a YAML mapping node (index -> value) or a YAML sequence node
// (expanded to indexes 0..n-1) into the list. Null/empty nodes reset it to an empty list.
// The node is decoded exactly once and validated before the container is replaced.
func (m *MapList[V]) UnmarshalYAML(value *yaml.Node) error {
	if value == nil {
		return errors.Errorf("sort_map: nil YAML node when unmarshaling into %T", m)
	}

	node := resolveYAMLNode(value)

	var data map[int]V
	switch {
	case isNullYAMLNode(node):
		// null/empty nodes reset the list to empty
	case node.Kind == yaml.MappingNode:
		if err := node.Decode(&data); err != nil {
			return err
		}
		if err := checkNonNegativeIndexes(data); err != nil {
			return err
		}
	case node.Kind == yaml.SequenceNode:
		var items []V
		if err := node.Decode(&items); err != nil {
			return err
		}
		data = make(map[int]V, len(items))
		for i, item := range items {
			data[i] = item
		}
	default:
		return errors.Errorf("sort_map: expecting a YAML mapping or sequence node, but got %s when unmarshaling into %T",
			yamlNodeKindName(node.Kind), m)
	}

	s := &m.SortMap
	s.ensure()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.replaceLocked(data)

	return nil
}

func checkNonNegativeIndexes[V any](data map[int]V) error {
	for k := range data {
		if k < 0 {
			return errors.Errorf("sort_map: negative index %d is not allowed in map list", k)
		}
	}
	return nil
}

func List[V any]() MapList[V] {
	return MapList[V]{
		SortMap: newSortMap[int, V](),
	}
}
