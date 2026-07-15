package sort_map

import (
	"cmp"
	"iter"
)

type MapIndexes[K cmp.Ordered] interface {
	Insert(key K)
	GetKey(idx int) K
	IndexOf(key K) int
	Contains(key K) (present bool)
	Remove(key K)
	Len() int
	Keys() iter.Seq[int]
	Range() iter.Seq2[int, K]
}

type mapIndexes[K cmp.Ordered] []K

func (mi *mapIndexes[K]) Insert(key K) {
	idx, _ := bSearchFirstGT[K](*mi, key)
	if idx == -1 {
		if n := len(*mi); n > 0 && (*mi)[n-1] == key {
			return
		}
	} else if idx > 0 && (*mi)[idx-1] == key {
		return
	}
	mi.insert(idx, key)
}

func (mi *mapIndexes[K]) GetKey(idx int) K {
	if idx < 0 || idx >= len(*mi) {
		var zero K
		return zero
	}
	return (*mi)[idx]
}

func (mi *mapIndexes[K]) IndexOf(key K) int {
	idx, _ := bSearch[K](*mi, key)
	return idx
}

func (mi *mapIndexes[K]) Contains(key K) (present bool) {
	idx := mi.IndexOf(key)
	return idx != -1
}

func (mi *mapIndexes[K]) Remove(key K) {
	idx := mi.IndexOf(key)
	if idx == -1 {
		return
	}
	*mi = append((*mi)[:idx], (*mi)[idx+1:]...)
}

func (mi *mapIndexes[K]) Len() int {
	return len(*mi)
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

func bSearch[K cmp.Ordered](ints []K, val K) (int, K) {
	return bSearchInternally[K](ints, 0, len(ints)-1, val)
}

func bSearchInternally[K cmp.Ordered](ints []K, low int, high int, val K) (int, K) {
	var zero K
	if low > high {
		return -1, zero
	}

	mid := low + ((high - low) >> 1)
	if ints[mid] == val {
		return mid, ints[mid]
	} else if ints[mid] > val {
		return bSearchInternally[K](ints, low, mid-1, val)
	} else {
		return bSearchInternally[K](ints, mid+1, high, val)
	}
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

func bSearchFirstFreeIndex(indexes *mapIndexes[int]) int {
	return bSearchFirstFreeIndexInternally(indexes, 0, len(*indexes)-1)
}

func bSearchFirstFreeIndexInternally(indexes *mapIndexes[int], low, high int) int {
	if low > high {
		return low
	}
	if indexes.GetKey(low) > low {
		return low
	}
	mid := low + ((high - low) >> 1)
	if indexes.GetKey(mid) > mid {
		return bSearchFirstFreeIndexInternally(indexes, low, mid)
	} else {
		return bSearchFirstFreeIndexInternally(indexes, mid+1, high)
	}
}

func lastFreeIndex(indexes *mapIndexes[int]) int {
	if indexes.Len() == 0 {
		return 0
	}
	return indexes.GetKey(indexes.Len()-1) + 1
}
