package sort_map

import (
	"encoding/json"
	"iter"

	"github.com/marmotedu/errors"
	"gopkg.in/yaml.v3"
)

type MapList[V any] interface {
	Get(index int) (value V, present bool)
	Set(index int, value V) (oldValue V, present bool)
	InsertToEmpty(value V) int
	Append(value V) int
	Range() iter.Seq2[int, V]
	Indexes() iter.Seq[int]
	json.Unmarshaler
	json.Marshaler
	yaml.Unmarshaler
	yaml.Marshaler
}

type mapList[V any] struct {
	*sortMap[int, V]
}

func (m *mapList[V]) Get(index int) (value V, present bool) {
	if index < 0 {
		return
	}
	return m.sortMap.Get(index)
}

func (m *mapList[V]) Set(index int, value V) (oldValue V, present bool) {
	if index < 0 {
		return
	}
	return m.sortMap.Set(index, value)
}

func (m *mapList[V]) InsertToEmpty(value V) int {
	index := bSearchFirstFreeIndex(m.indexList.(*mapIndexes[int]))
	m.sortMap.Set(index, value)
	return index
}

func (m *mapList[V]) Append(value V) int {
	index := lastFreeIndex(m.indexList.(*mapIndexes[int]))
	m.sortMap.Set(index, value)
	return index
}

func (m *mapList[V]) Range() iter.Seq2[int, V] {
	return m.sortMap.Range()
}

func (m *mapList[V]) UnmarshalJSON(bytes []byte) error {
	var data map[int]V
	if err := json.Unmarshal(bytes, &data); err != nil {
		return err
	}

	for k := range data {
		if k < 0 {
			return errors.Errorf("negative index %d is not allowed", k)
		}
	}

	return m.sortMap.UnmarshalJSON(bytes)
}

func (m *mapList[V]) UnmarshalYAML(value *yaml.Node) error {
	if value == nil {
		return errors.Errorf("nil yaml node")
	}

	var data map[int]V
	if err := value.Decode(&data); err != nil {
		return err
	}

	for k := range data {
		if k < 0 {
			return errors.Errorf("negative index %d is not allowed", k)
		}
	}

	return m.sortMap.UnmarshalYAML(value)
}

func List[V any]() MapList[V] {
	return &mapList[V]{
		sortMap: Map[int, V]().(*sortMap[int, V]),
	}
}
