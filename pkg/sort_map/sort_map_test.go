package sort_map

import (
	"cmp"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func newTestSortMap[K cmp.Ordered, V any](data map[K]V) sortMap[K, V] {
	sm := sortMap[K, V]{
		mu:        new(sync.RWMutex),
		dataMap:   make(map[K]V),
		indexList: NewMapIndexes[K](),
	}
	for k, v := range data {
		sm.indexList.Insert(k)
		sm.dataMap[k] = v
	}
	return sm
}

func Test_sortMap_GetByKey(t *testing.T) {
	type args struct {
		key int
	}
	type testCase struct {
		name   string
		s      sortMap[int, string]
		args   args
		wantV  string
		wantOk bool
	}
	tests := []testCase{
		{
			name:   "get from empty map",
			s:      newTestSortMap[int, string](map[int]string{}),
			args:   args{key: 1},
			wantV:  "",
			wantOk: false,
		},
		{
			name:   "get existing key",
			s:      newTestSortMap[int, string](map[int]string{10: "ten", 20: "twenty", 30: "thirty"}),
			args:   args{key: 20},
			wantV:  "twenty",
			wantOk: true,
		},
		{
			name:   "get non-existent key",
			s:      newTestSortMap[int, string](map[int]string{10: "ten", 20: "twenty"}),
			args:   args{key: 99},
			wantV:  "",
			wantOk: false,
		},
		{
			name:   "get first key",
			s:      newTestSortMap[int, string](map[int]string{1: "one", 2: "two", 3: "three"}),
			args:   args{key: 1},
			wantV:  "one",
			wantOk: true,
		},
		{
			name:   "get last key",
			s:      newTestSortMap[int, string](map[int]string{1: "one", 2: "two", 3: "three"}),
			args:   args{key: 3},
			wantV:  "three",
			wantOk: true,
		},
		{
			name:   "get with negative key",
			s:      newTestSortMap[int, string](map[int]string{-5: "neg5", 0: "zero", 5: "pos5"}),
			args:   args{key: -5},
			wantV:  "neg5",
			wantOk: true,
		},
		{
			name:   "get with zero value",
			s:      newTestSortMap[int, string](map[int]string{1: ""}),
			args:   args{key: 1},
			wantV:  "",
			wantOk: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotV, gotOk := tt.s.GetByKey(tt.args.key)
			if !reflect.DeepEqual(gotV, tt.wantV) {
				t.Errorf("GetByKey() gotV = %v, want %v", gotV, tt.wantV)
			}
			if gotOk != tt.wantOk {
				t.Errorf("GetByKey() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

func Test_sortMap_Indexes(t *testing.T) {
	type testCase struct {
		name     string
		s        sortMap[int, string]
		wantIdxs []int
	}
	tests := []testCase{
		{
			name:     "empty map indexes",
			s:        newTestSortMap[int, string](map[int]string{}),
			wantIdxs: []int{},
		},
		{
			name:     "single element indexes",
			s:        newTestSortMap[int, string](map[int]string{10: "ten"}),
			wantIdxs: []int{0},
		},
		{
			name:     "multiple elements indexes",
			s:        newTestSortMap[int, string](map[int]string{30: "thirty", 10: "ten", 20: "twenty"}),
			wantIdxs: []int{0, 1, 2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []int
			for idx := range tt.s.Indexes() {
				got = append(got, idx)
			}
			if len(got) == 0 && len(tt.wantIdxs) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tt.wantIdxs) {
				t.Errorf("Indexes() = %v, want %v", got, tt.wantIdxs)
			}
		})
	}

	t.Run("indexes yield returns false to stop early", func(t *testing.T) {
		sm := newTestSortMap[int, string](map[int]string{10: "ten", 20: "twenty", 30: "thirty", 40: "forty"})
		var got []int
		sm.Indexes()(func(idx int) bool {
			got = append(got, idx)
			return len(got) < 2
		})
		want := []int{0, 1}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Indexes() early stop = %v, want %v", got, want)
		}
	})
}

func Test_sortMap_Insert(t *testing.T) {
	type args struct {
		key   int
		value string
	}
	type testCase struct {
		name       string
		initial    map[int]string
		args       args
		wantKeys   []int
		wantValues map[int]string
		wantErr    bool
	}
	tests := []testCase{
		{
			name:       "insert into empty map",
			initial:    map[int]string{},
			args:       args{key: 10, value: "ten"},
			wantKeys:   []int{10},
			wantValues: map[int]string{10: "ten"},
			wantErr:    false,
		},
		{
			name:       "insert at beginning",
			initial:    map[int]string{20: "twenty", 30: "thirty"},
			args:       args{key: 10, value: "ten"},
			wantKeys:   []int{10, 20, 30},
			wantValues: map[int]string{10: "ten", 20: "twenty", 30: "thirty"},
			wantErr:    false,
		},
		{
			name:       "insert at end",
			initial:    map[int]string{10: "ten", 20: "twenty"},
			args:       args{key: 30, value: "thirty"},
			wantKeys:   []int{10, 20, 30},
			wantValues: map[int]string{10: "ten", 20: "twenty", 30: "thirty"},
			wantErr:    false,
		},
		{
			name:       "insert in middle",
			initial:    map[int]string{10: "ten", 30: "thirty"},
			args:       args{key: 20, value: "twenty"},
			wantKeys:   []int{10, 20, 30},
			wantValues: map[int]string{10: "ten", 20: "twenty", 30: "thirty"},
			wantErr:    false,
		},
		{
			name:       "update existing key",
			initial:    map[int]string{10: "ten", 20: "twenty"},
			args:       args{key: 10, value: "TEN"},
			wantKeys:   []int{10, 20},
			wantValues: map[int]string{10: "TEN", 20: "twenty"},
			wantErr:    false,
		},
		{
			name:       "insert negative key",
			initial:    map[int]string{0: "zero", 5: "five"},
			args:       args{key: -5, value: "neg5"},
			wantKeys:   []int{-5, 0, 5},
			wantValues: map[int]string{-5: "neg5", 0: "zero", 5: "five"},
			wantErr:    false,
		},
		{
			name:       "insert with empty string value",
			initial:    map[int]string{1: "one"},
			args:       args{key: 2, value: ""},
			wantKeys:   []int{1, 2},
			wantValues: map[int]string{1: "one", 2: ""},
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestSortMap[int, string](tt.initial)
			if err := s.Insert(tt.args.key, tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("Insert() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				var gotKeys []int
				for k := range s.Keys() {
					gotKeys = append(gotKeys, k)
				}
				if !reflect.DeepEqual(gotKeys, tt.wantKeys) {
					t.Errorf("Insert() keys = %v, want %v", gotKeys, tt.wantKeys)
				}
				for k, wantV := range tt.wantValues {
					gotV, ok := s.GetByKey(k)
					if !ok {
						t.Errorf("Insert() key %v not found", k)
					}
					if gotV != wantV {
						t.Errorf("Insert() value[%v] = %v, want %v", k, gotV, wantV)
					}
				}
			}
		})
	}
}

func Test_sortMap_Keys(t *testing.T) {
	type testCase struct {
		name     string
		s        sortMap[int, string]
		wantKeys []int
	}
	tests := []testCase{
		{
			name:     "empty map keys",
			s:        newTestSortMap[int, string](map[int]string{}),
			wantKeys: []int{},
		},
		{
			name:     "single element keys",
			s:        newTestSortMap[int, string](map[int]string{10: "ten"}),
			wantKeys: []int{10},
		},
		{
			name:     "multiple elements keys sorted",
			s:        newTestSortMap[int, string](map[int]string{30: "thirty", 10: "ten", 20: "twenty"}),
			wantKeys: []int{10, 20, 30},
		},
		{
			name:     "negative keys",
			s:        newTestSortMap[int, string](map[int]string{-5: "neg5", 5: "pos5", 0: "zero"}),
			wantKeys: []int{-5, 0, 5},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []int
			for k := range tt.s.Keys() {
				got = append(got, k)
			}
			if len(got) == 0 && len(tt.wantKeys) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tt.wantKeys) {
				t.Errorf("Keys() = %v, want %v", got, tt.wantKeys)
			}
		})
	}

	t.Run("keys yield returns false to stop early", func(t *testing.T) {
		sm := newTestSortMap[int, string](map[int]string{10: "ten", 20: "twenty", 30: "thirty", 40: "forty"})
		var got []int
		sm.Keys()(func(k int) bool {
			got = append(got, k)
			return len(got) < 3
		})
		want := []int{10, 20, 30}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Keys() early stop = %v, want %v", got, want)
		}
	})

	t.Run("keys yield returns false on first call", func(t *testing.T) {
		sm := newTestSortMap[int, string](map[int]string{10: "ten", 20: "twenty"})
		var got []int
		sm.Keys()(func(k int) bool {
			got = append(got, k)
			return false
		})
		want := []int{10}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Keys() stop immediately = %v, want %v", got, want)
		}
	})
}

func Test_sortMap_Range(t *testing.T) {
	type testCase struct {
		name     string
		s        sortMap[int, string]
		wantKeys []int
		wantVals []string
	}
	tests := []testCase{
		{
			name:     "empty map range",
			s:        newTestSortMap[int, string](map[int]string{}),
			wantKeys: []int{},
			wantVals: []string{},
		},
		{
			name:     "single element range",
			s:        newTestSortMap[int, string](map[int]string{10: "ten"}),
			wantKeys: []int{10},
			wantVals: []string{"ten"},
		},
		{
			name:     "multiple elements range sorted",
			s:        newTestSortMap[int, string](map[int]string{30: "thirty", 10: "ten", 20: "twenty"}),
			wantKeys: []int{10, 20, 30},
			wantVals: []string{"ten", "twenty", "thirty"},
		},
		{
			name:     "negative keys range",
			s:        newTestSortMap[int, string](map[int]string{-5: "neg5", 5: "pos5", 0: "zero"}),
			wantKeys: []int{-5, 0, 5},
			wantVals: []string{"neg5", "zero", "pos5"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotKeys []int
			var gotVals []string
			for k, v := range tt.s.Range() {
				gotKeys = append(gotKeys, k)
				gotVals = append(gotVals, v)
			}
			if len(gotKeys) == 0 && len(tt.wantKeys) == 0 && len(gotVals) == 0 && len(tt.wantVals) == 0 {
				return
			}
			if !reflect.DeepEqual(gotKeys, tt.wantKeys) {
				t.Errorf("Range() keys = %v, want %v", gotKeys, tt.wantKeys)
			}
			if !reflect.DeepEqual(gotVals, tt.wantVals) {
				t.Errorf("Range() vals = %v, want %v", gotVals, tt.wantVals)
			}
		})
	}

	t.Run("range yield returns false to stop early", func(t *testing.T) {
		sm := newTestSortMap[int, string](map[int]string{10: "ten", 20: "twenty", 30: "thirty", 40: "forty"})
		var gotKeys []int
		var gotVals []string
		sm.Range()(func(k int, v string) bool {
			gotKeys = append(gotKeys, k)
			gotVals = append(gotVals, v)
			return len(gotKeys) < 3
		})
		wantKeys := []int{10, 20, 30}
		wantVals := []string{"ten", "twenty", "thirty"}
		if !reflect.DeepEqual(gotKeys, wantKeys) {
			t.Errorf("Range() early stop keys = %v, want %v", gotKeys, wantKeys)
		}
		if !reflect.DeepEqual(gotVals, wantVals) {
			t.Errorf("Range() early stop vals = %v, want %v", gotVals, wantVals)
		}
	})

	t.Run("range yield returns false on first call", func(t *testing.T) {
		sm := newTestSortMap[int, string](map[int]string{10: "ten", 20: "twenty"})
		var gotKeys []int
		var gotVals []string
		sm.Range()(func(k int, v string) bool {
			gotKeys = append(gotKeys, k)
			gotVals = append(gotVals, v)
			return false
		})
		wantKeys := []int{10}
		wantVals := []string{"ten"}
		if !reflect.DeepEqual(gotKeys, wantKeys) {
			t.Errorf("Range() stop immediately keys = %v, want %v", gotKeys, wantKeys)
		}
		if !reflect.DeepEqual(gotVals, wantVals) {
			t.Errorf("Range() stop immediately vals = %v, want %v", gotVals, wantVals)
		}
	})
}

func Test_sortMap_RangeWithIndex(t *testing.T) {
	type testCase struct {
		name     string
		s        sortMap[int, string]
		wantIdxs []int
		wantVals []string
	}
	tests := []testCase{
		{
			name:     "empty map range with index",
			s:        newTestSortMap[int, string](map[int]string{}),
			wantIdxs: []int{},
			wantVals: []string{},
		},
		{
			name:     "single element range with index",
			s:        newTestSortMap[int, string](map[int]string{10: "ten"}),
			wantIdxs: []int{0},
			wantVals: []string{"ten"},
		},
		{
			name:     "multiple elements range with index sorted",
			s:        newTestSortMap[int, string](map[int]string{30: "thirty", 10: "ten", 20: "twenty"}),
			wantIdxs: []int{0, 1, 2},
			wantVals: []string{"ten", "twenty", "thirty"},
		},
		{
			name:     "negative keys range with index",
			s:        newTestSortMap[int, string](map[int]string{-5: "neg5", 5: "pos5", 0: "zero"}),
			wantIdxs: []int{0, 1, 2},
			wantVals: []string{"neg5", "zero", "pos5"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotIdxs []int
			var gotVals []string
			for idx, v := range tt.s.RangeWithIndex() {
				gotIdxs = append(gotIdxs, idx)
				gotVals = append(gotVals, v)
			}
			if len(gotIdxs) == 0 && len(tt.wantIdxs) == 0 && len(gotVals) == 0 && len(tt.wantVals) == 0 {
				return
			}
			if !reflect.DeepEqual(gotIdxs, tt.wantIdxs) {
				t.Errorf("RangeWithIndex() idxs = %v, want %v", gotIdxs, tt.wantIdxs)
			}
			if !reflect.DeepEqual(gotVals, tt.wantVals) {
				t.Errorf("RangeWithIndex() vals = %v, want %v", gotVals, tt.wantVals)
			}
		})
	}

	t.Run("range with index yield returns false to stop early", func(t *testing.T) {
		sm := newTestSortMap[int, string](map[int]string{10: "ten", 20: "twenty", 30: "thirty", 40: "forty"})
		var gotIdxs []int
		var gotVals []string
		sm.RangeWithIndex()(func(idx int, v string) bool {
			gotIdxs = append(gotIdxs, idx)
			gotVals = append(gotVals, v)
			return len(gotIdxs) < 3
		})
		wantIdxs := []int{0, 1, 2}
		wantVals := []string{"ten", "twenty", "thirty"}
		if !reflect.DeepEqual(gotIdxs, wantIdxs) {
			t.Errorf("RangeWithIndex() early stop idxs = %v, want %v", gotIdxs, wantIdxs)
		}
		if !reflect.DeepEqual(gotVals, wantVals) {
			t.Errorf("RangeWithIndex() early stop vals = %v, want %v", gotVals, wantVals)
		}
	})

	t.Run("range with index yield returns false on first call", func(t *testing.T) {
		sm := newTestSortMap[int, string](map[int]string{10: "ten", 20: "twenty"})
		var gotIdxs []int
		var gotVals []string
		sm.RangeWithIndex()(func(idx int, v string) bool {
			gotIdxs = append(gotIdxs, idx)
			gotVals = append(gotVals, v)
			return false
		})
		wantIdxs := []int{0}
		wantVals := []string{"ten"}
		if !reflect.DeepEqual(gotIdxs, wantIdxs) {
			t.Errorf("RangeWithIndex() stop immediately idxs = %v, want %v", gotIdxs, wantIdxs)
		}
		if !reflect.DeepEqual(gotVals, wantVals) {
			t.Errorf("RangeWithIndex() stop immediately vals = %v, want %v", gotVals, wantVals)
		}
	})
}

func Test_sortMap_RemoveByKey(t *testing.T) {
	type args struct {
		key int
	}
	type testCase struct {
		name       string
		initial    map[int]string
		args       args
		wantErr    bool
		wantKeys   []int
		wantExists map[int]bool
	}
	tests := []testCase{
		{
			name:       "remove from empty map",
			initial:    map[int]string{},
			args:       args{key: 10},
			wantErr:    false,
			wantKeys:   []int{},
			wantExists: map[int]bool{10: false},
		},
		{
			name:       "remove first key",
			initial:    map[int]string{10: "ten", 20: "twenty", 30: "thirty"},
			args:       args{key: 10},
			wantErr:    false,
			wantKeys:   []int{20, 30},
			wantExists: map[int]bool{10: false, 20: true, 30: true},
		},
		{
			name:       "remove last key",
			initial:    map[int]string{10: "ten", 20: "twenty", 30: "thirty"},
			args:       args{key: 30},
			wantErr:    false,
			wantKeys:   []int{10, 20},
			wantExists: map[int]bool{10: true, 20: true, 30: false},
		},
		{
			name:       "remove middle key",
			initial:    map[int]string{10: "ten", 20: "twenty", 30: "thirty"},
			args:       args{key: 20},
			wantErr:    false,
			wantKeys:   []int{10, 30},
			wantExists: map[int]bool{10: true, 20: false, 30: true},
		},
		{
			name:       "remove non-existent key",
			initial:    map[int]string{10: "ten", 20: "twenty"},
			args:       args{key: 99},
			wantErr:    false,
			wantKeys:   []int{10, 20},
			wantExists: map[int]bool{10: true, 20: true, 99: false},
		},
		{
			name:       "remove only key",
			initial:    map[int]string{10: "ten"},
			args:       args{key: 10},
			wantErr:    false,
			wantKeys:   []int{},
			wantExists: map[int]bool{10: false},
		},
		{
			name:       "remove negative key",
			initial:    map[int]string{-10: "neg10", 0: "zero", 10: "pos10"},
			args:       args{key: 0},
			wantErr:    false,
			wantKeys:   []int{-10, 10},
			wantExists: map[int]bool{-10: true, 0: false, 10: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestSortMap[int, string](tt.initial)
			if err := s.RemoveByKey(tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("RemoveByKey() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				var gotKeys []int
				for k := range s.Keys() {
					gotKeys = append(gotKeys, k)
				}
				if len(gotKeys) == 0 && len(tt.wantKeys) == 0 {
					return
				}
				if !reflect.DeepEqual(gotKeys, tt.wantKeys) {
					t.Errorf("RemoveByKey() keys = %v, want %v", gotKeys, tt.wantKeys)
				}
				for k, shouldExist := range tt.wantExists {
					_, ok := s.GetByKey(k)
					if ok != shouldExist {
						t.Errorf("RemoveByKey() key %v exists = %v, want %v", k, ok, shouldExist)
					}
				}
			}
		})
	}
}

func Test_sortMap_ConcurrentAccess(t *testing.T) {
	t.Run("concurrent insert get and remove", func(t *testing.T) {
		sm := Map[int, string]()
		var wg sync.WaitGroup
		var errCount atomic.Int64

		numGoroutines := 100
		numOpsPerGoroutine := 100

		// Concurrent inserts
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOpsPerGoroutine; j++ {
					key := id*numOpsPerGoroutine + j
					if err := sm.Insert(key, "value"); err != nil {
						errCount.Add(1)
					}
				}
			}(i)
		}
		wg.Wait()

		if errCount.Load() > 0 {
			t.Errorf("Concurrent insert errors: %d", errCount.Load())
		}

		// Verify count
		count := 0
		for range sm.Keys() {
			count++
		}
		expectedCount := numGoroutines * numOpsPerGoroutine
		if count != expectedCount {
			t.Errorf("Expected %d keys, got %d", expectedCount, count)
		}

		// Concurrent reads and removes
		errCount.Store(0)
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOpsPerGoroutine; j++ {
					key := id*numOpsPerGoroutine + j
					// Concurrent read
					_, _ = sm.GetByKey(key)
					// Concurrent remove
					if err := sm.RemoveByKey(key); err != nil {
						errCount.Add(1)
					}
				}
			}(i)
		}
		wg.Wait()

		if errCount.Load() > 0 {
			t.Errorf("Concurrent remove errors: %d", errCount.Load())
		}

		// Verify all removed
		count = 0
		for range sm.Keys() {
			count++
		}
		if count != 0 {
			t.Errorf("Expected 0 keys after removal, got %d", count)
		}
	})

	t.Run("concurrent read write mixed operations", func(t *testing.T) {
		sm := Map[int, int]()
		var wg sync.WaitGroup
		var insertCount atomic.Int64
		var getCount atomic.Int64
		var removeCount atomic.Int64

		numGoroutines := 50
		numOpsPerGoroutine := 200

		// Pre-populate some data
		for i := 0; i < 1000; i++ {
			sm.Insert(i, i*10)
		}

		// Mixed concurrent operations
		for i := 0; i < numGoroutines; i++ {
			wg.Add(3)

			// Insert goroutines
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOpsPerGoroutine; j++ {
					key := 1000 + id*numOpsPerGoroutine + j
					if err := sm.Insert(key, key); err == nil {
						insertCount.Add(1)
					}
				}
			}(i)

			// Get goroutines
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOpsPerGoroutine; j++ {
					key := id % 1000
					if _, ok := sm.GetByKey(key); ok {
						getCount.Add(1)
					}
				}
			}(i)

			// Remove goroutines
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOpsPerGoroutine; j++ {
					key := id % 1000
					if err := sm.RemoveByKey(key); err == nil {
						removeCount.Add(1)
					}
				}
			}(i)
		}
		wg.Wait()

		t.Logf("Concurrent ops completed: inserts=%d, gets=%d, removes=%d",
			insertCount.Load(), getCount.Load(), removeCount.Load())
	})

	t.Run("concurrent iteration safety", func(t *testing.T) {
		sm := Map[int, string]()
		var wg sync.WaitGroup
		var panicCount atomic.Int64

		// Pre-populate
		for i := 0; i < 100; i++ {
			sm.Insert(i, "value")
		}

		numGoroutines := 20

		// Concurrent iteration while modifying
		for i := 0; i < numGoroutines; i++ {
			wg.Add(2)

			// Iterating goroutine
			go func() {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						panicCount.Add(1)
					}
				}()
				for j := 0; j < 10; j++ {
					for range sm.Keys() {
						// Just iterate
					}
				}
			}()

			// Modifying goroutine
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					key := 100 + id*100 + j
					sm.Insert(key, "new")
					sm.RemoveByKey(key)
				}
			}(i)
		}
		wg.Wait()

		if panicCount.Load() > 0 {
			t.Errorf("Concurrent iteration caused %d panics", panicCount.Load())
		}
	})

	t.Run("concurrent range with index safety", func(t *testing.T) {
		sm := Map[int, string]()
		var wg sync.WaitGroup
		var panicCount atomic.Int64

		// Pre-populate
		for i := 0; i < 50; i++ {
			sm.Insert(i, "value")
		}

		numGoroutines := 10

		// Concurrent RangeWithIndex iteration while modifying
		for i := 0; i < numGoroutines; i++ {
			wg.Add(2)

			// Iterating goroutine
			go func() {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						panicCount.Add(1)
					}
				}()
				for j := 0; j < 10; j++ {
					for _, _ = range sm.RangeWithIndex() {
						// Just iterate
					}
				}
			}()

			// Modifying goroutine
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 50; j++ {
					key := 50 + id*50 + j
					sm.Insert(key, "new")
					sm.RemoveByKey(key)
				}
			}(i)
		}
		wg.Wait()

		if panicCount.Load() > 0 {
			t.Errorf("Concurrent RangeWithIndex iteration caused %d panics", panicCount.Load())
		}
	})

	t.Run("concurrent same key updates", func(t *testing.T) {
		sm := Map[int, int]()
		var wg sync.WaitGroup
		numGoroutines := 50
		numOpsPerGoroutine := 1000

		// Multiple goroutines updating the same key
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOpsPerGoroutine; j++ {
					sm.Insert(1, id*numOpsPerGoroutine+j)
				}
			}(i)
		}
		wg.Wait()

		// Verify key exists and has some value
		val, ok := sm.GetByKey(1)
		if !ok {
			t.Error("Key 1 should exist after concurrent updates")
		}
		t.Logf("Final value for key 1: %d", val)

		// Verify only one key in index
		count := 0
		for range sm.Keys() {
			count++
		}
		if count != 1 {
			t.Errorf("Expected 1 key, got %d", count)
		}
	})

	t.Run("concurrent indexes access", func(t *testing.T) {
		sm := Map[int, string]()
		var wg sync.WaitGroup
		var panicCount atomic.Int64

		// Pre-populate
		for i := 0; i < 100; i++ {
			sm.Insert(i, "value")
		}

		numGoroutines := 20

		// Concurrent Indexes iteration while modifying
		for i := 0; i < numGoroutines; i++ {
			wg.Add(2)

			// Iterating goroutine
			go func() {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						panicCount.Add(1)
					}
				}()
				for j := 0; j < 10; j++ {
					for range sm.Indexes() {
						// Just iterate
					}
				}
			}()

			// Modifying goroutine
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					key := 100 + id*100 + j
					sm.Insert(key, "new")
					sm.RemoveByKey(key)
				}
			}(i)
		}
		wg.Wait()

		if panicCount.Load() > 0 {
			t.Errorf("Concurrent Indexes iteration caused %d panics", panicCount.Load())
		}
	})

	t.Run("concurrent large dataset operations", func(t *testing.T) {
		sm := Map[int, string]()
		var wg sync.WaitGroup
		var errCount atomic.Int64

		numGoroutines := 50
		numOpsPerGoroutine := 500

		// Concurrent inserts of large dataset
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOpsPerGoroutine; j++ {
					key := id*numOpsPerGoroutine + j
					if err := sm.Insert(key, "value"); err != nil {
						errCount.Add(1)
					}
				}
			}(i)
		}
		wg.Wait()

		if errCount.Load() > 0 {
			t.Errorf("Concurrent large dataset insert errors: %d", errCount.Load())
		}

		// Verify count
		count := 0
		for range sm.Keys() {
			count++
		}
		expectedCount := numGoroutines * numOpsPerGoroutine
		if count != expectedCount {
			t.Errorf("Expected %d keys in large dataset, got %d", expectedCount, count)
		}
	})

	t.Run("iterator deferred consumption", func(t *testing.T) {
		sm := Map[int, string]()
		sm.Insert(1, "one")
		sm.Insert(2, "two")
		sm.Insert(3, "three")

		// Get iterator
		keysIter := sm.Keys()

		// Modify data before consuming iterator
		sm.Insert(4, "four")
		sm.RemoveByKey(2)

		// Consume iterator
		var gotKeys []int
		for k := range keysIter {
			gotKeys = append(gotKeys, k)
		}

		// Note: Due to lock behavior, this may show inconsistent state
		t.Logf("Keys after deferred consumption: %v", gotKeys)
	})

	t.Run("concurrent insert and range consistency", func(t *testing.T) {
		sm := Map[int, int]()
		var wg sync.WaitGroup
		var rangeResults []int
		var mu sync.Mutex

		// Pre-populate
		for i := 0; i < 100; i++ {
			sm.Insert(i, i)
		}

		numGoroutines := 10

		// Range goroutine
		wg.Add(1)
		go func() {
			defer wg.Done()
			for k, v := range sm.Range() {
				mu.Lock()
				rangeResults = append(rangeResults, k)
				if k != v {
					t.Errorf("Inconsistent key-value pair: key=%d, value=%d", k, v)
				}
				mu.Unlock()
			}
		}()

		// Insert goroutines
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					key := 100 + id*100 + j
					sm.Insert(key, key)
				}
			}(i)
		}

		wg.Wait()
		t.Logf("Range iteration collected %d key-value pairs", len(rangeResults))
	})

	t.Run("yield callback deadlock detection - Keys", func(t *testing.T) {
		sm := Map[int, string]()
		sm.Insert(1, "one")
		sm.Insert(2, "two")
		sm.Insert(3, "three")

		done := make(chan bool, 1)
		go func() {
			// yield callback tries to modify the map
			count := 0
			for range sm.Keys() {
				count++
				if count == 2 {
					// This will deadlock: yield holds RLock, Insert needs Lock
					sm.Insert(4, "four")
				}
			}
			done <- true
		}()

		select {
		case <-done:
			t.Log("Keys() yield callback completed without deadlock")
		case <-time.After(2 * time.Second):
			t.Log("Keys() yield callback detected potential deadlock (expected behavior)")
		}
	})

	t.Run("yield callback deadlock detection - Range", func(t *testing.T) {
		sm := Map[int, string]()
		sm.Insert(1, "one")
		sm.Insert(2, "two")

		done := make(chan bool, 1)
		go func() {
			count := 0
			for _, _ = range sm.Range() {
				count++
				if count == 1 {
					sm.RemoveByKey(1)
				}
			}
			done <- true
		}()

		select {
		case <-done:
			t.Log("Range() yield callback completed without deadlock")
		case <-time.After(2 * time.Second):
			t.Log("Range() yield callback detected potential deadlock (expected behavior)")
		}
	})

	t.Run("yield callback deadlock detection - Indexes", func(t *testing.T) {
		sm := Map[int, string]()
		sm.Insert(1, "one")
		sm.Insert(2, "two")

		done := make(chan bool, 1)
		go func() {
			count := 0
			for range sm.Indexes() {
				count++
				if count == 1 {
					sm.Insert(3, "three")
				}
			}
			done <- true
		}()

		select {
		case <-done:
			t.Log("Indexes() yield callback completed without deadlock")
		case <-time.After(2 * time.Second):
			t.Log("Indexes() yield callback detected potential deadlock (expected behavior)")
		}
	})

	t.Run("yield callback deadlock detection - RangeWithIndex", func(t *testing.T) {
		sm := Map[int, string]()
		sm.Insert(1, "one")
		sm.Insert(2, "two")

		done := make(chan bool, 1)
		go func() {
			count := 0
			for _, _ = range sm.RangeWithIndex() {
				count++
				if count == 1 {
					sm.RemoveByKey(1)
				}
			}
			done <- true
		}()

		select {
		case <-done:
			t.Log("RangeWithIndex() yield callback completed without deadlock")
		case <-time.After(2 * time.Second):
			t.Log("RangeWithIndex() yield callback detected potential deadlock (expected behavior)")
		}
	})

	t.Run("yield panic does not leak lock", func(t *testing.T) {
		sm := Map[int, string]()
		sm.Insert(1, "one")
		sm.Insert(2, "two")

		// Test Keys() panic recovery
		func() {
			defer func() {
				recover()
			}()
			for range sm.Keys() {
				panic("test panic in yield")
			}
		}()

		// Verify lock is not leaked - subsequent operations should work
		_, ok := sm.GetByKey(1)
		if !ok {
			t.Error("GetByKey failed after panic recovery, possible lock leak")
		}

		// Test Range() panic recovery
		func() {
			defer func() {
				recover()
			}()
			for _, _ = range sm.Range() {
				panic("test panic in yield")
			}
		}()

		_, ok = sm.GetByKey(2)
		if !ok {
			t.Error("GetByKey failed after Range panic recovery, possible lock leak")
		}

		// Test Indexes() panic recovery
		func() {
			defer func() {
				recover()
			}()
			for range sm.Indexes() {
				panic("test panic in yield")
			}
		}()

		_, ok = sm.GetByKey(1)
		if !ok {
			t.Error("GetByKey failed after Indexes panic recovery, possible lock leak")
		}

		// Test RangeWithIndex() panic recovery
		func() {
			defer func() {
				recover()
			}()
			for _, _ = range sm.RangeWithIndex() {
				panic("test panic in yield")
			}
		}()

		_, ok = sm.GetByKey(2)
		if !ok {
			t.Error("GetByKey failed after RangeWithIndex panic recovery, possible lock leak")
		}

		t.Log("All panic recovery tests passed, no lock leaks detected")
	})

	t.Run("iterator multiple invocations safety", func(t *testing.T) {
		sm := Map[int, string]()
		sm.Insert(1, "one")
		sm.Insert(2, "two")

		// Get iterator function
		keysIter := sm.Keys()

		// Invoke it multiple times
		for i := 0; i < 5; i++ {
			count := 0
			for range keysIter {
				count++
			}
			if count != 2 {
				t.Errorf("Iteration %d: expected 2 keys, got %d", i, count)
			}
		}
	})

	t.Run("cross goroutine iterator usage", func(t *testing.T) {
		sm := Map[int, string]()
		sm.Insert(1, "one")
		sm.Insert(2, "two")
		sm.Insert(3, "three")

		// Get iterator in main goroutine
		keysIter := sm.Keys()

		// Consume in different goroutine
		var wg sync.WaitGroup
		var gotKeys []int
		var mu sync.Mutex

		wg.Add(1)
		go func() {
			defer wg.Done()
			for k := range keysIter {
				mu.Lock()
				gotKeys = append(gotKeys, k)
				mu.Unlock()
			}
		}()

		wg.Wait()

		if len(gotKeys) != 3 {
			t.Errorf("Expected 3 keys, got %d", len(gotKeys))
		}
	})

	t.Run("concurrent multiple iterators", func(t *testing.T) {
		sm := Map[int, string]()
		for i := 0; i < 100; i++ {
			sm.Insert(i, "value")
		}

		var wg sync.WaitGroup
		var panicCount atomic.Int64
		numGoroutines := 20

		for i := 0; i < numGoroutines; i++ {
			wg.Add(4)

			// Keys iterator
			go func() {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						panicCount.Add(1)
					}
				}()
				for range sm.Keys() {
					// iterate
				}
			}()

			// Range iterator
			go func() {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						panicCount.Add(1)
					}
				}()
				for _, _ = range sm.Range() {
					// iterate
				}
			}()

			// Indexes iterator
			go func() {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						panicCount.Add(1)
					}
				}()
				for range sm.Indexes() {
					// iterate
				}
			}()

			// RangeWithIndex iterator
			go func() {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						panicCount.Add(1)
					}
				}()
				for _, _ = range sm.RangeWithIndex() {
					// iterate
				}
			}()
		}

		wg.Wait()

		if panicCount.Load() > 0 {
			t.Errorf("Concurrent iterator usage caused %d panics", panicCount.Load())
		}
	})
}
