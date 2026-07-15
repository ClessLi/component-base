package sort_map

import (
	"cmp"
	"reflect"
	"testing"
)

func Test_bSearchFirstGT(t *testing.T) {
	type args[K cmp.Ordered] struct {
		ints []K
		val  K
	}
	type testCase[K cmp.Ordered] struct {
		name  string
		args  args[K]
		want  int
		want1 K
	}
	tests := []testCase[int]{
		{
			name:  "empty slice",
			args:  args[int]{ints: []int{}, val: 5},
			want:  -1,
			want1: 0,
		},
		{
			name:  "single element greater than val",
			args:  args[int]{ints: []int{10}, val: 5},
			want:  0,
			want1: 10,
		},
		{
			name:  "single element less than val",
			args:  args[int]{ints: []int{3}, val: 5},
			want:  -1,
			want1: 0,
		},
		{
			name:  "single element equal to val",
			args:  args[int]{ints: []int{5}, val: 5},
			want:  -1,
			want1: 0,
		},
		{
			name:  "all elements greater than val",
			args:  args[int]{ints: []int{6, 7, 8, 9, 10}, val: 5},
			want:  0,
			want1: 6,
		},
		{
			name:  "all elements less than val",
			args:  args[int]{ints: []int{1, 2, 3, 4, 5}, val: 10},
			want:  -1,
			want1: 0,
		},
		{
			name:  "val in middle of array",
			args:  args[int]{ints: []int{1, 3, 5, 7, 9}, val: 4},
			want:  2,
			want1: 5,
		},
		{
			name:  "val equals an element",
			args:  args[int]{ints: []int{1, 3, 5, 7, 9}, val: 5},
			want:  3,
			want1: 7,
		},
		{
			name:  "val less than first element",
			args:  args[int]{ints: []int{10, 20, 30}, val: 5},
			want:  0,
			want1: 10,
		},
		{
			name:  "val greater than last element",
			args:  args[int]{ints: []int{10, 20, 30}, val: 35},
			want:  -1,
			want1: 0,
		},
		{
			name:  "val between first and second element",
			args:  args[int]{ints: []int{10, 20, 30}, val: 15},
			want:  1,
			want1: 20,
		},
		{
			name:  "val between last two elements",
			args:  args[int]{ints: []int{10, 20, 30}, val: 25},
			want:  2,
			want1: 30,
		},
		{
			name:  "duplicate elements all greater",
			args:  args[int]{ints: []int{5, 5, 5, 5}, val: 3},
			want:  0,
			want1: 5,
		},
		{
			name:  "duplicate elements all less",
			args:  args[int]{ints: []int{3, 3, 3, 3}, val: 5},
			want:  -1,
			want1: 0,
		},
		{
			name:  "duplicate elements with val equal",
			args:  args[int]{ints: []int{3, 5, 5, 5, 7}, val: 5},
			want:  4,
			want1: 7,
		},
		{
			name:  "two elements both greater",
			args:  args[int]{ints: []int{10, 20}, val: 5},
			want:  0,
			want1: 10,
		},
		{
			name:  "two elements first less second greater",
			args:  args[int]{ints: []int{10, 20}, val: 15},
			want:  1,
			want1: 20,
		},
		{
			name:  "two elements both less",
			args:  args[int]{ints: []int{10, 20}, val: 25},
			want:  -1,
			want1: 0,
		},
		{
			name:  "negative values all greater",
			args:  args[int]{ints: []int{-5, -3, -1}, val: -10},
			want:  0,
			want1: -5,
		},
		{
			name:  "negative values all less",
			args:  args[int]{ints: []int{-5, -3, -1}, val: 0},
			want:  -1,
			want1: 0,
		},
		{
			name:  "negative and positive mixed",
			args:  args[int]{ints: []int{-3, -1, 2, 5}, val: 0},
			want:  2,
			want1: 2,
		},
		{
			name:  "large array val in first half",
			args:  args[int]{ints: []int{2, 4, 6, 8, 10, 12, 14, 16, 18, 20}, val: 5},
			want:  2,
			want1: 6,
		},
		{
			name:  "large array val in second half",
			args:  args[int]{ints: []int{2, 4, 6, 8, 10, 12, 14, 16, 18, 20}, val: 15},
			want:  7,
			want1: 16,
		},
		{
			name:  "consecutive integers val at boundary",
			args:  args[int]{ints: []int{1, 2, 3, 4, 5}, val: 1},
			want:  1,
			want1: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := bSearchFirstGT(tt.args.ints, tt.args.val)
			if got != tt.want {
				t.Errorf("bSearchFirstGT() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("bSearchFirstGT() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_mapIndexes_Insert(t *testing.T) {
	type args[K cmp.Ordered] struct {
		key K
	}
	type testCase[K cmp.Ordered] struct {
		name     string
		mi       mapIndexes[K]
		args     args[K]
		wantKeys []K
	}
	tests := []testCase[int]{
		{
			name:     "insert into empty slice",
			mi:       mapIndexes[int]{},
			args:     args[int]{key: 5},
			wantKeys: []int{5},
		},
		{
			name:     "insert at beginning",
			mi:       mapIndexes[int]{10, 20, 30},
			args:     args[int]{key: 5},
			wantKeys: []int{5, 10, 20, 30},
		},
		{
			name:     "insert at end",
			mi:       mapIndexes[int]{10, 20, 30},
			args:     args[int]{key: 40},
			wantKeys: []int{10, 20, 30, 40},
		},
		{
			name:     "insert in middle",
			mi:       mapIndexes[int]{10, 30, 50},
			args:     args[int]{key: 25},
			wantKeys: []int{10, 25, 30, 50},
		},
		{
			name:     "insert duplicate value is ignored",
			mi:       mapIndexes[int]{10, 20, 30},
			args:     args[int]{key: 20},
			wantKeys: []int{10, 20, 30},
		},
		{
			name:     "insert between first two elements",
			mi:       mapIndexes[int]{10, 20},
			args:     args[int]{key: 15},
			wantKeys: []int{10, 15, 20},
		},
		{
			name:     "insert between last two elements",
			mi:       mapIndexes[int]{10, 20, 30},
			args:     args[int]{key: 25},
			wantKeys: []int{10, 20, 25, 30},
		},
		{
			name:     "insert negative value at beginning",
			mi:       mapIndexes[int]{-5, 0, 5},
			args:     args[int]{key: -10},
			wantKeys: []int{-10, -5, 0, 5},
		},
		{
			name:     "insert negative value in middle",
			mi:       mapIndexes[int]{-10, 0, 10},
			args:     args[int]{key: -5},
			wantKeys: []int{-10, -5, 0, 10},
		},
		{
			name:     "insert single element greater",
			mi:       mapIndexes[int]{10},
			args:     args[int]{key: 5},
			wantKeys: []int{5, 10},
		},
		{
			name:     "insert single element less",
			mi:       mapIndexes[int]{5},
			args:     args[int]{key: 10},
			wantKeys: []int{5, 10},
		},
		{
			name:     "insert into large sorted array",
			mi:       mapIndexes[int]{2, 4, 6, 8, 10, 12, 14, 16, 18, 20},
			args:     args[int]{key: 11},
			wantKeys: []int{2, 4, 6, 8, 10, 11, 12, 14, 16, 18, 20},
		},
		{
			name:     "insert duplicate at first position is ignored",
			mi:       mapIndexes[int]{10, 20, 30},
			args:     args[int]{key: 10},
			wantKeys: []int{10, 20, 30},
		},
		{
			name:     "insert duplicate at last position is ignored",
			mi:       mapIndexes[int]{10, 20, 30},
			args:     args[int]{key: 30},
			wantKeys: []int{10, 20, 30},
		},
		{
			name:     "insert multiple duplicates is ignored",
			mi:       mapIndexes[int]{10, 20},
			args:     args[int]{key: 20},
			wantKeys: []int{10, 20},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mi.Insert(tt.args.key)
			if !reflect.DeepEqual([]int(tt.mi), tt.wantKeys) {
				t.Errorf("Set() keys = %v, want %v", []int(tt.mi), tt.wantKeys)
			}
		})
	}
}

func Test_mapIndexes_Keys(t *testing.T) {
	type testCase struct {
		name string
		mi   mapIndexes[int]
		want []int
	}
	tests := []testCase{
		{
			name: "empty slice",
			mi:   mapIndexes[int]{},
			want: []int{},
		},
		{
			name: "single element",
			mi:   mapIndexes[int]{10},
			want: []int{0},
		},
		{
			name: "multiple elements",
			mi:   mapIndexes[int]{5, 10, 15, 20},
			want: []int{0, 1, 2, 3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []int
			for idx := range tt.mi.Keys() {
				got = append(got, idx)
			}
			// Handle empty case: nil slice vs empty slice
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Keys() = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("yield returns false to stop early", func(t *testing.T) {
		mi := mapIndexes[int]{5, 10, 15, 20, 25}
		var got []int
		mi.Keys()(func(idx int) bool {
			got = append(got, idx)
			return len(got) < 3 // Stop after 3 elements
		})
		want := []int{0, 1, 2}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Keys() early stop = %v, want %v", got, want)
		}
	})

	t.Run("yield returns false on first call", func(t *testing.T) {
		mi := mapIndexes[int]{5, 10, 15}
		var got []int
		mi.Keys()(func(idx int) bool {
			got = append(got, idx)
			return false // Stop immediately
		})
		want := []int{0}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Keys() stop immediately = %v, want %v", got, want)
		}
	})
}

func Test_mapIndexes_Range(t *testing.T) {
	type testCase struct {
		name     string
		mi       mapIndexes[int]
		wantIdx  []int
		wantKeys []int
	}
	tests := []testCase{
		{
			name:     "empty slice",
			mi:       mapIndexes[int]{},
			wantIdx:  []int{},
			wantKeys: []int{},
		},
		{
			name:     "single element",
			mi:       mapIndexes[int]{10},
			wantIdx:  []int{0},
			wantKeys: []int{10},
		},
		{
			name:     "multiple elements sorted",
			mi:       mapIndexes[int]{5, 10, 15, 20},
			wantIdx:  []int{0, 1, 2, 3},
			wantKeys: []int{5, 10, 15, 20},
		},
		{
			name:     "negative values",
			mi:       mapIndexes[int]{-10, -5, 0, 5},
			wantIdx:  []int{0, 1, 2, 3},
			wantKeys: []int{-10, -5, 0, 5},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotIdx []int
			var gotKeys []int
			for idx, key := range tt.mi.Range() {
				gotIdx = append(gotIdx, idx)
				gotKeys = append(gotKeys, key)
			}
			// Handle empty case: nil slice vs empty slice
			if len(gotIdx) == 0 && len(tt.wantIdx) == 0 && len(gotKeys) == 0 && len(tt.wantKeys) == 0 {
				return
			}
			if !reflect.DeepEqual(gotIdx, tt.wantIdx) {
				t.Errorf("Range() idx = %v, want %v", gotIdx, tt.wantIdx)
			}
			if !reflect.DeepEqual(gotKeys, tt.wantKeys) {
				t.Errorf("Range() keys = %v, want %v", gotKeys, tt.wantKeys)
			}
		})
	}

	t.Run("yield returns false to stop early", func(t *testing.T) {
		mi := mapIndexes[int]{5, 10, 15, 20, 25}
		var gotIdx []int
		var gotKeys []int
		mi.Range()(func(idx int, key int) bool {
			gotIdx = append(gotIdx, idx)
			gotKeys = append(gotKeys, key)
			return len(gotIdx) < 3 // Stop after 3 elements
		})
		wantIdx := []int{0, 1, 2}
		wantKeys := []int{5, 10, 15}
		if !reflect.DeepEqual(gotIdx, wantIdx) {
			t.Errorf("Range() early stop idx = %v, want %v", gotIdx, wantIdx)
		}
		if !reflect.DeepEqual(gotKeys, wantKeys) {
			t.Errorf("Range() early stop keys = %v, want %v", gotKeys, wantKeys)
		}
	})

	t.Run("yield returns false on first call", func(t *testing.T) {
		mi := mapIndexes[int]{5, 10, 15}
		var gotIdx []int
		var gotKeys []int
		mi.Range()(func(idx int, key int) bool {
			gotIdx = append(gotIdx, idx)
			gotKeys = append(gotKeys, key)
			return false // Stop immediately
		})
		wantIdx := []int{0}
		wantKeys := []int{5}
		if !reflect.DeepEqual(gotIdx, wantIdx) {
			t.Errorf("Range() stop immediately idx = %v, want %v", gotIdx, wantIdx)
		}
		if !reflect.DeepEqual(gotKeys, wantKeys) {
			t.Errorf("Range() stop immediately keys = %v, want %v", gotKeys, wantKeys)
		}
	})
}

func Test_mapIndexes_Remove(t *testing.T) {
	type args struct {
		key int
	}
	type testCase struct {
		name     string
		mi       mapIndexes[int]
		args     args
		wantLen  int
		wantKeys []int
	}
	tests := []testCase{
		{
			name:     "remove from empty slice",
			mi:       mapIndexes[int]{},
			args:     args{key: 5},
			wantLen:  0,
			wantKeys: []int{},
		},
		{
			name:     "remove first element",
			mi:       mapIndexes[int]{10, 20, 30},
			args:     args{key: 10},
			wantLen:  2,
			wantKeys: []int{20, 30},
		},
		{
			name:     "remove last element",
			mi:       mapIndexes[int]{10, 20, 30},
			args:     args{key: 30},
			wantLen:  2,
			wantKeys: []int{10, 20},
		},
		{
			name:     "remove middle element",
			mi:       mapIndexes[int]{10, 20, 30},
			args:     args{key: 20},
			wantLen:  2,
			wantKeys: []int{10, 30},
		},
		{
			name:     "remove non-existent element",
			mi:       mapIndexes[int]{10, 20, 30},
			args:     args{key: 25},
			wantLen:  3,
			wantKeys: []int{10, 20, 30},
		},
		{
			name:     "remove only element",
			mi:       mapIndexes[int]{10},
			args:     args{key: 10},
			wantLen:  0,
			wantKeys: []int{},
		},
		{
			name:     "remove negative value",
			mi:       mapIndexes[int]{-10, -5, 0, 5},
			args:     args{key: -5},
			wantLen:  3,
			wantKeys: []int{-10, 0, 5},
		},
		{
			name:     "remove duplicate value first occurrence",
			mi:       mapIndexes[int]{10, 20, 20, 30},
			args:     args{key: 20},
			wantLen:  3,
			wantKeys: []int{10, 20, 30},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mi.Remove(tt.args.key)
			if len(tt.mi) != tt.wantLen {
				t.Errorf("Remove() len = %v, want %v", len(tt.mi), tt.wantLen)
			}
			if !reflect.DeepEqual([]int(tt.mi), tt.wantKeys) {
				t.Errorf("Remove() keys = %v, want %v", []int(tt.mi), tt.wantKeys)
			}
		})
	}
}

func Test_bSearchFirstFreeIndex(t *testing.T) {
	type args struct {
		indexes *mapIndexes[int]
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "empty slice",
			args: args{indexes: &mapIndexes[int]{}},
			want: 0,
		},
		{
			name: "single element index 0",
			args: args{indexes: &mapIndexes[int]{0}},
			want: 1,
		},
		{
			name: "single element index greater than 0",
			args: args{indexes: &mapIndexes[int]{5}},
			want: 0,
		},
		{
			name: "consecutive indexes starting from 0",
			args: args{indexes: &mapIndexes[int]{0, 1, 2, 3, 4}},
			want: 5,
		},
		{
			name: "first index missing",
			args: args{indexes: &mapIndexes[int]{1, 2, 3, 4}},
			want: 0,
		},
		{
			name: "gap in middle",
			args: args{indexes: &mapIndexes[int]{0, 1, 3, 4, 5}},
			want: 2,
		},
		{
			name: "gap at beginning",
			args: args{indexes: &mapIndexes[int]{2, 3, 4, 5}},
			want: 0,
		},
		{
			name: "gap at end",
			args: args{indexes: &mapIndexes[int]{0, 1, 2, 4}},
			want: 3,
		},
		{
			name: "multiple gaps return first",
			args: args{indexes: &mapIndexes[int]{0, 2, 4, 6}},
			want: 1,
		},
		{
			name: "all indexes shifted by 1",
			args: args{indexes: &mapIndexes[int]{1, 2, 3}},
			want: 0,
		},
		{
			name: "all indexes shifted by 2",
			args: args{indexes: &mapIndexes[int]{2, 3, 4}},
			want: 0,
		},
		{
			name: "two elements no gap",
			args: args{indexes: &mapIndexes[int]{0, 1}},
			want: 2,
		},
		{
			name: "two elements with gap",
			args: args{indexes: &mapIndexes[int]{0, 2}},
			want: 1,
		},
		{
			name: "large consecutive array",
			args: args{indexes: &mapIndexes[int]{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}},
			want: 10,
		},
		{
			name: "large array with gap in middle",
			args: args{indexes: &mapIndexes[int]{0, 1, 2, 3, 5, 6, 7, 8, 9}},
			want: 4,
		},
		{
			name: "sparse indexes",
			args: args{indexes: &mapIndexes[int]{10, 20, 30}},
			want: 0,
		},
		{
			name: "gap after first element",
			args: args{indexes: &mapIndexes[int]{0, 10, 20}},
			want: 1,
		},
		{
			name: "single gap at position 1",
			args: args{indexes: &mapIndexes[int]{0, 2, 3, 4}},
			want: 1,
		},
		{
			name: "very large single index",
			args: args{indexes: &mapIndexes[int]{1000000}},
			want: 0,
		},
		{
			name: "three elements no gap",
			args: args{indexes: &mapIndexes[int]{0, 1, 2}},
			want: 3,
		},
		{
			name: "gap at last position",
			args: args{indexes: &mapIndexes[int]{0, 1, 2, 3, 5}},
			want: 4,
		},
		{
			name: "three elements gap at end",
			args: args{indexes: &mapIndexes[int]{0, 1, 3}},
			want: 2,
		},
		{
			name: "three elements gap at start",
			args: args{indexes: &mapIndexes[int]{1, 2, 3}},
			want: 0,
		},
		{
			name: "consecutive starting from non-zero",
			args: args{indexes: &mapIndexes[int]{5, 6, 7, 8}},
			want: 0,
		},
		{
			name: "gap immediately after zero",
			args: args{indexes: &mapIndexes[int]{0, 5, 6, 7}},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := bSearchFirstFreeIndex(tt.args.indexes); got != tt.want {
				t.Errorf("bSearchFirstFreeIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_lastFreeIndex(t *testing.T) {
	type args struct {
		indexes *mapIndexes[int]
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "single element zero",
			args: args{indexes: &mapIndexes[int]{0}},
			want: 1,
		},
		{
			name: "single element non-zero",
			args: args{indexes: &mapIndexes[int]{5}},
			want: 6,
		},
		{
			name: "consecutive from zero",
			args: args{indexes: &mapIndexes[int]{0, 1, 2, 3, 4}},
			want: 5,
		},
		{
			name: "consecutive from non-zero",
			args: args{indexes: &mapIndexes[int]{10, 11, 12}},
			want: 13,
		},
		{
			name: "with gaps",
			args: args{indexes: &mapIndexes[int]{0, 2, 4, 6}},
			want: 7,
		},
		{
			name: "negative values",
			args: args{indexes: &mapIndexes[int]{-5, -3, -1}},
			want: 0,
		},
		{
			name: "large last value",
			args: args{indexes: &mapIndexes[int]{0, 1, 1000}},
			want: 1001,
		},
		{
			name: "two elements",
			args: args{indexes: &mapIndexes[int]{10, 20}},
			want: 21,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lastFreeIndex(tt.args.indexes); got != tt.want {
				t.Errorf("lastFreeIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mapIndexes_Contains(t *testing.T) {
	type args[K cmp.Ordered] struct {
		key K
	}
	type testCase[K cmp.Ordered] struct {
		name        string
		mi          mapIndexes[K]
		args        args[K]
		wantPresent bool
	}
	tests := []testCase[int]{
		{
			name:        "empty slice - not present",
			mi:          mapIndexes[int]{},
			args:        args[int]{key: 5},
			wantPresent: false,
		},
		{
			name:        "single element - present",
			mi:          mapIndexes[int]{5},
			args:        args[int]{key: 5},
			wantPresent: true,
		},
		{
			name:        "single element - not present",
			mi:          mapIndexes[int]{5},
			args:        args[int]{key: 3},
			wantPresent: false,
		},
		{
			name:        "multiple elements - present at beginning",
			mi:          mapIndexes[int]{1, 3, 5, 7, 9},
			args:        args[int]{key: 1},
			wantPresent: true,
		},
		{
			name:        "multiple elements - present at end",
			mi:          mapIndexes[int]{1, 3, 5, 7, 9},
			args:        args[int]{key: 9},
			wantPresent: true,
		},
		{
			name:        "multiple elements - present in middle",
			mi:          mapIndexes[int]{1, 3, 5, 7, 9},
			args:        args[int]{key: 5},
			wantPresent: true,
		},
		{
			name:        "multiple elements - not present less than all",
			mi:          mapIndexes[int]{1, 3, 5, 7, 9},
			args:        args[int]{key: 0},
			wantPresent: false,
		},
		{
			name:        "multiple elements - not present greater than all",
			mi:          mapIndexes[int]{1, 3, 5, 7, 9},
			args:        args[int]{key: 10},
			wantPresent: false,
		},
		{
			name:        "multiple elements - not present between elements",
			mi:          mapIndexes[int]{1, 3, 5, 7, 9},
			args:        args[int]{key: 4},
			wantPresent: false,
		},
		{
			name:        "negative values - present",
			mi:          mapIndexes[int]{-5, -3, -1, 2, 4},
			args:        args[int]{key: -3},
			wantPresent: true,
		},
		{
			name:        "negative values - not present",
			mi:          mapIndexes[int]{-5, -3, -1, 2, 4},
			args:        args[int]{key: -2},
			wantPresent: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotPresent := tt.mi.Contains(tt.args.key); gotPresent != tt.wantPresent {
				t.Errorf("Contains() = %v, want %v", gotPresent, tt.wantPresent)
			}
		})
	}
}

func Test_mapIndexes_IndexOf(t *testing.T) {
	type args[K cmp.Ordered] struct {
		key K
	}
	type testCase[K cmp.Ordered] struct {
		name string
		mi   mapIndexes[K]
		args args[K]
		want int
	}
	tests := []testCase[int]{
		{
			name: "empty slice - not found",
			mi:   mapIndexes[int]{},
			args: args[int]{key: 5},
			want: -1,
		},
		{
			name: "single element - found",
			mi:   mapIndexes[int]{5},
			args: args[int]{key: 5},
			want: 0,
		},
		{
			name: "single element - not found",
			mi:   mapIndexes[int]{5},
			args: args[int]{key: 3},
			want: -1,
		},
		{
			name: "multiple elements - found at beginning",
			mi:   mapIndexes[int]{1, 3, 5, 7, 9},
			args: args[int]{key: 1},
			want: 0,
		},
		{
			name: "multiple elements - found at end",
			mi:   mapIndexes[int]{1, 3, 5, 7, 9},
			args: args[int]{key: 9},
			want: 4,
		},
		{
			name: "multiple elements - found in middle",
			mi:   mapIndexes[int]{1, 3, 5, 7, 9},
			args: args[int]{key: 5},
			want: 2,
		},
		{
			name: "multiple elements - not found less than all",
			mi:   mapIndexes[int]{1, 3, 5, 7, 9},
			args: args[int]{key: 0},
			want: -1,
		},
		{
			name: "multiple elements - not found greater than all",
			mi:   mapIndexes[int]{1, 3, 5, 7, 9},
			args: args[int]{key: 10},
			want: -1,
		},
		{
			name: "multiple elements - not found between elements",
			mi:   mapIndexes[int]{1, 3, 5, 7, 9},
			args: args[int]{key: 4},
			want: -1,
		},
		{
			name: "negative values - found",
			mi:   mapIndexes[int]{-5, -3, -1, 2, 4},
			args: args[int]{key: -3},
			want: 1,
		},
		{
			name: "negative values - not found",
			mi:   mapIndexes[int]{-5, -3, -1, 2, 4},
			args: args[int]{key: -2},
			want: -1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.mi.IndexOf(tt.args.key); got != tt.want {
				t.Errorf("IndexOf() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mapIndexes_GetKey(t *testing.T) {
	type args struct {
		idx int
	}
	type testCase struct {
		name string
		mi   mapIndexes[int]
		args args
		want int
	}
	tests := []testCase{
		{
			name: "empty slice - index 0 returns zero",
			mi:   mapIndexes[int]{},
			args: args{idx: 0},
			want: 0,
		},
		{
			name: "negative index returns zero",
			mi:   mapIndexes[int]{10, 20, 30},
			args: args{idx: -1},
			want: 0,
		},
		{
			name: "index out of bounds returns zero",
			mi:   mapIndexes[int]{10, 20, 30},
			args: args{idx: 3},
			want: 0,
		},
		{
			name: "first element",
			mi:   mapIndexes[int]{10, 20, 30},
			args: args{idx: 0},
			want: 10,
		},
		{
			name: "last element",
			mi:   mapIndexes[int]{10, 20, 30},
			args: args{idx: 2},
			want: 30,
		},
		{
			name: "middle element",
			mi:   mapIndexes[int]{10, 20, 30},
			args: args{idx: 1},
			want: 20,
		},
		{
			name: "negative values",
			mi:   mapIndexes[int]{-5, -3, -1, 2, 4},
			args: args{idx: 2},
			want: -1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.mi.GetKey(tt.args.idx); got != tt.want {
				t.Errorf("GetKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mapIndexes_Len(t *testing.T) {
	type testCase[K cmp.Ordered] struct {
		name string
		mi   mapIndexes[K]
		want int
	}
	tests := []testCase[int]{
		{
			name: "empty slice",
			mi:   mapIndexes[int]{},
			want: 0,
		},
		{
			name: "single element",
			mi:   mapIndexes[int]{5},
			want: 1,
		},
		{
			name: "multiple elements",
			mi:   mapIndexes[int]{1, 3, 5, 7, 9},
			want: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.mi.Len(); got != tt.want {
				t.Errorf("Len() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_bSearch(t *testing.T) {
	type args[K cmp.Ordered] struct {
		ints []K
		val  K
	}
	type testCase[K cmp.Ordered] struct {
		name  string
		args  args[K]
		want  int
		want1 K
	}
	tests := []testCase[int]{
		{
			name:  "empty slice",
			args:  args[int]{ints: []int{}, val: 5},
			want:  -1,
			want1: 0,
		},
		{
			name:  "single element - found",
			args:  args[int]{ints: []int{5}, val: 5},
			want:  0,
			want1: 5,
		},
		{
			name:  "single element - not found less",
			args:  args[int]{ints: []int{5}, val: 3},
			want:  -1,
			want1: 0,
		},
		{
			name:  "single element - not found greater",
			args:  args[int]{ints: []int{5}, val: 10},
			want:  -1,
			want1: 0,
		},
		{
			name:  "found at beginning",
			args:  args[int]{ints: []int{1, 3, 5, 7, 9}, val: 1},
			want:  0,
			want1: 1,
		},
		{
			name:  "found at end",
			args:  args[int]{ints: []int{1, 3, 5, 7, 9}, val: 9},
			want:  4,
			want1: 9,
		},
		{
			name:  "found in middle",
			args:  args[int]{ints: []int{1, 3, 5, 7, 9}, val: 5},
			want:  2,
			want1: 5,
		},
		{
			name:  "not found less than all",
			args:  args[int]{ints: []int{1, 3, 5, 7, 9}, val: 0},
			want:  -1,
			want1: 0,
		},
		{
			name:  "not found greater than all",
			args:  args[int]{ints: []int{1, 3, 5, 7, 9}, val: 10},
			want:  -1,
			want1: 0,
		},
		{
			name:  "not found between elements",
			args:  args[int]{ints: []int{1, 3, 5, 7, 9}, val: 4},
			want:  -1,
			want1: 0,
		},
		{
			name:  "negative values - found",
			args:  args[int]{ints: []int{-5, -3, -1, 2, 4}, val: -3},
			want:  1,
			want1: -3,
		},
		{
			name:  "negative values - not found",
			args:  args[int]{ints: []int{-5, -3, -1, 2, 4}, val: -2},
			want:  -1,
			want1: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := bSearch(tt.args.ints, tt.args.val)
			if got != tt.want {
				t.Errorf("bSearch() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("bSearch() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
