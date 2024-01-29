package prevent_cycles

import (
	"fmt"
	"github.com/dominikbraun/graph"
	"github.com/marmotedu/errors"
	"strings"
)

type Preventer interface {
	CheckLoop(any, any) error
}

type preventer[K comparable, T any] struct {
	graph graph.Graph[K, T]
}

func (s *preventer[K, T]) CheckLoop(src, dst any) error {
	err := s.graph.AddVertex(src.(T))
	if err != nil && !errors.Is(err, graph.ErrVertexAlreadyExists) {
		return err
	}

	err = s.graph.AddVertex(dst.(T))
	if err != nil && !errors.Is(err, graph.ErrVertexAlreadyExists) {
		return err
	}

	err = s.graph.AddEdge(src.(K), dst.(K))
	if err != nil && !errors.Is(err, graph.ErrEdgeAlreadyExists) {
		topo, topoErr := graph.TopologicalSort(s.graph)
		if topoErr != nil {
			return topoErr
		}
		topoStr := make([]string, 0)
		for _, k := range topo {
			topoStr = append(topoStr, fmt.Sprintf("%v", k))
		}

		return errors.Errorf("%v [Topological sort: %s; Add edge failed: %s]", err, strings.Join(topoStr, " -> "), fmt.Sprintf("%v -x-> %v", src, dst))
	}
	return nil
}

func NewStringPreventer() Preventer {
	return New[string, string](graph.StringHash)
}

func New[K comparable, T any](hash graph.Hash[K, T]) Preventer {
	g := graph.New(hash, graph.PreventCycles(), graph.Directed())
	return &preventer[K, T]{graph: g}
}
