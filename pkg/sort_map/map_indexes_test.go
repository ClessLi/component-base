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
		name string
		mi   mapIndexes[K]
		args args[K]
	}
	tests := []testCase[int]{
		{
			name: "insert into empty slice",
			mi:   mapIndexes[int]{},
			args: args[int]{key: 5},
		},
		{
			name: "insert at beginning",
			mi:   mapIndexes[int]{10, 20, 30},
			args: args[int]{key: 5},
		},
		{
			name: "insert at end",
			mi:   mapIndexes[int]{10, 20, 30},
			args: args[int]{key: 40},
		},
		{
			name: "insert in middle",
			mi:   mapIndexes[int]{10, 30, 50},
			args: args[int]{key: 25},
		},
		{
			name: "insert duplicate value",
			mi:   mapIndexes[int]{10, 20, 30},
			args: args[int]{key: 20},
		},
		{
			name: "insert between first two elements",
			mi:   mapIndexes[int]{10, 20},
			args: args[int]{key: 15},
		},
		{
			name: "insert between last two elements",
			mi:   mapIndexes[int]{10, 20, 30},
			args: args[int]{key: 25},
		},
		{
			name: "insert negative value at beginning",
			mi:   mapIndexes[int]{-5, 0, 5},
			args: args[int]{key: -10},
		},
		{
			name: "insert negative value in middle",
			mi:   mapIndexes[int]{-10, 0, 10},
			args: args[int]{key: -5},
		},
		{
			name: "insert single element greater",
			mi:   mapIndexes[int]{10},
			args: args[int]{key: 5},
		},
		{
			name: "insert single element less",
			mi:   mapIndexes[int]{5},
			args: args[int]{key: 10},
		},
		{
			name: "insert into large sorted array",
			mi:   mapIndexes[int]{2, 4, 6, 8, 10, 12, 14, 16, 18, 20},
			args: args[int]{key: 11},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mi.Insert(tt.args.key)
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
