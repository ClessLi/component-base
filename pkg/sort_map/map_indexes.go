package sort_map

import (
	"cmp"
	"iter"
)

type MapIndexes[K cmp.Ordered] interface {
	Insert(key K)
	Remove(key K)
	Keys() iter.Seq[int]
	Range() iter.Seq2[int, K]
}

type mapIndexes[K cmp.Ordered] []K

func (mi *mapIndexes[K]) Insert(key K) {
	idx, _ := bSearchFirstGT[K](*mi, key)
	mi.insert(idx, key)
}

func (mi *mapIndexes[K]) Remove(key K) {
	for i, k := range *mi {
		if k == key {
			*mi = append((*mi)[:i], (*mi)[i+1:]...)
			return
		}
	}
}

func (mi *mapIndexes[K]) Keys() iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := range *mi {
			if !yield(i) {
				return
			}
		}
	}
}

func (mi *mapIndexes[K]) Range() iter.Seq2[int, K] {
	return func(yield func(int, K) bool) {
		for i, key := range *mi {
			if !yield(i, key) {
				return
			}
		}
	}
}

func NewMapIndexes[K cmp.Ordered]() MapIndexes[K] {
	return &mapIndexes[K]{}
}

func bSearchFirstGT[K cmp.Ordered](ints []K, val K) (int, K) {
	return bSearchFirstGTInternally[K](ints, 0, len(ints)-1, val)
}

func bSearchFirstGTInternally[K cmp.Ordered](ints []K, low int, high int, val K) (int, K) {
	var zero K
	if low > high {
		return -1, zero
	}

	if ints[low] > val {
		return low, ints[low]
	}
	mid := low + ((high - low) >> 1)
	if ints[mid] > val {
		return bSearchFirstGTInternally[K](ints, low, mid, val)
	} else {
		return bSearchFirstGTInternally[K](ints, mid+1, high, val)
	}
}

func (mi *mapIndexes[K]) insert(index int, key K) {
	n := len(*mi)
	*mi = append(*mi, key)
	if index == -1 {
		return
	}
	for i := n; i > index; i-- {
		(*mi)[i] = (*mi)[i-1]
	}
	(*mi)[index] = key
}
