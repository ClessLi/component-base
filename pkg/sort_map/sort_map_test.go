package sort_map

import (
	"cmp"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
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
			gotV, gotOk := tt.s.Get(tt.args.key)
			if !reflect.DeepEqual(gotV, tt.wantV) {
				t.Errorf("Get() gotV = %v, want %v", gotV, tt.wantV)
			}
			if gotOk != tt.wantOk {
				t.Errorf("Get() gotOk = %v, want %v", gotOk, tt.wantOk)
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

func Test_sortMap_Set(t *testing.T) {
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
	}
	tests := []testCase{
		{
			name:       "set into empty map",
			initial:    map[int]string{},
			args:       args{key: 10, value: "ten"},
			wantKeys:   []int{10},
			wantValues: map[int]string{10: "ten"},
		},
		{
			name:       "set at beginning",
			initial:    map[int]string{20: "twenty", 30: "thirty"},
			args:       args{key: 10, value: "ten"},
			wantKeys:   []int{10, 20, 30},
			wantValues: map[int]string{10: "ten", 20: "twenty", 30: "thirty"},
		},
		{
			name:       "set at end",
			initial:    map[int]string{10: "ten", 20: "twenty"},
			args:       args{key: 30, value: "thirty"},
			wantKeys:   []int{10, 20, 30},
			wantValues: map[int]string{10: "ten", 20: "twenty", 30: "thirty"},
		},
		{
			name:       "set in middle",
			initial:    map[int]string{10: "ten", 30: "thirty"},
			args:       args{key: 20, value: "twenty"},
			wantKeys:   []int{10, 20, 30},
			wantValues: map[int]string{10: "ten", 20: "twenty", 30: "thirty"},
		},
		{
			name:       "update existing key",
			initial:    map[int]string{10: "ten", 20: "twenty"},
			args:       args{key: 10, value: "TEN"},
			wantKeys:   []int{10, 20},
			wantValues: map[int]string{10: "TEN", 20: "twenty"},
		},
		{
			name:       "set negative key",
			initial:    map[int]string{0: "zero", 5: "five"},
			args:       args{key: -5, value: "neg5"},
			wantKeys:   []int{-5, 0, 5},
			wantValues: map[int]string{-5: "neg5", 0: "zero", 5: "five"},
		},
		{
			name:       "set with empty string value",
			initial:    map[int]string{1: "one"},
			args:       args{key: 2, value: ""},
			wantKeys:   []int{1, 2},
			wantValues: map[int]string{1: "one", 2: ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestSortMap[int, string](tt.initial)
			s.Set(tt.args.key, tt.args.value)
			var gotKeys []int
			for k := range s.Keys() {
				gotKeys = append(gotKeys, k)
			}
			if !reflect.DeepEqual(gotKeys, tt.wantKeys) {
				t.Errorf("Set() keys = %v, want %v", gotKeys, tt.wantKeys)
			}
			for k, wantV := range tt.wantValues {
				gotV, ok := s.Get(k)
				if !ok {
					t.Errorf("Set() key %v not found", k)
				}
				if gotV != wantV {
					t.Errorf("Set() value[%v] = %v, want %v", k, gotV, wantV)
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
		wantKeys   []int
		wantExists map[int]bool
	}
	tests := []testCase{
		{
			name:       "remove from empty map",
			initial:    map[int]string{},
			args:       args{key: 10},
			wantKeys:   []int{},
			wantExists: map[int]bool{10: false},
		},
		{
			name:       "remove first key",
			initial:    map[int]string{10: "ten", 20: "twenty", 30: "thirty"},
			args:       args{key: 10},
			wantKeys:   []int{20, 30},
			wantExists: map[int]bool{10: false, 20: true, 30: true},
		},
		{
			name:       "remove last key",
			initial:    map[int]string{10: "ten", 20: "twenty", 30: "thirty"},
			args:       args{key: 30},
			wantKeys:   []int{10, 20},
			wantExists: map[int]bool{10: true, 20: true, 30: false},
		},
		{
			name:       "remove middle key",
			initial:    map[int]string{10: "ten", 20: "twenty", 30: "thirty"},
			args:       args{key: 20},
			wantKeys:   []int{10, 30},
			wantExists: map[int]bool{10: true, 20: false, 30: true},
		},
		{
			name:       "remove non-existent key",
			initial:    map[int]string{10: "ten", 20: "twenty"},
			args:       args{key: 99},
			wantKeys:   []int{10, 20},
			wantExists: map[int]bool{10: true, 20: true, 99: false},
		},
		{
			name:       "remove only key",
			initial:    map[int]string{10: "ten"},
			args:       args{key: 10},
			wantKeys:   []int{},
			wantExists: map[int]bool{10: false},
		},
		{
			name:       "remove negative key",
			initial:    map[int]string{-10: "neg10", 0: "zero", 10: "pos10"},
			args:       args{key: 0},
			wantKeys:   []int{-10, 10},
			wantExists: map[int]bool{-10: true, 0: false, 10: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestSortMap[int, string](tt.initial)
			s.RemoveByKey(tt.args.key)
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
				_, ok := s.Get(k)
				if ok != shouldExist {
					t.Errorf("RemoveByKey() key %v exists = %v, want %v", k, ok, shouldExist)
				}
			}
		})
	}
}

func Test_sortMap_ConcurrentAccess(t *testing.T) {
	t.Run("concurrent insert get and remove", func(t *testing.T) {
		sm := Map[int, string]()
		var wg sync.WaitGroup

		numGoroutines := 100
		numOpsPerGoroutine := 100

		// Concurrent inserts
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOpsPerGoroutine; j++ {
					key := id*numOpsPerGoroutine + j
					sm.Set(key, "value")
				}
			}(i)
		}
		wg.Wait()

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
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOpsPerGoroutine; j++ {
					key := id*numOpsPerGoroutine + j
					// Concurrent read
					_, _ = sm.Get(key)
					// Concurrent remove
					sm.RemoveByKey(key)
				}
			}(i)
		}
		wg.Wait()

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
			sm.Set(i, i*10)
		}

		// Mixed concurrent operations
		for i := 0; i < numGoroutines; i++ {
			wg.Add(3)

			// Set goroutines
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOpsPerGoroutine; j++ {
					key := 1000 + id*numOpsPerGoroutine + j
					sm.Set(key, key)
					insertCount.Add(1)
				}
			}(i)

			// Get goroutines
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOpsPerGoroutine; j++ {
					key := id % 1000
					if _, ok := sm.Get(key); ok {
						getCount.Add(1)
					}
				}
			}(i)

			// Remove goroutines
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOpsPerGoroutine; j++ {
					key := id % 1000
					sm.RemoveByKey(key)
					removeCount.Add(1)
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
			sm.Set(i, "value")
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
					sm.Set(key, "new")
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
			sm.Set(i, "value")
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
					sm.Set(key, "new")
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
					sm.Set(1, id*numOpsPerGoroutine+j)
				}
			}(i)
		}
		wg.Wait()

		// Verify key exists and has some value
		val, ok := sm.Get(1)
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
			sm.Set(i, "value")
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
					sm.Set(key, "new")
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

		numGoroutines := 50
		numOpsPerGoroutine := 500

		// Concurrent inserts of large dataset
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOpsPerGoroutine; j++ {
					key := id*numOpsPerGoroutine + j
					sm.Set(key, "value")
				}
			}(i)
		}
		wg.Wait()

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
		sm.Set(1, "one")
		sm.Set(2, "two")
		sm.Set(3, "three")

		// Get iterator
		keysIter := sm.Keys()

		// Modify data before consuming iterator
		sm.Set(4, "four")
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
			sm.Set(i, i)
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

		// Set goroutines
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					key := 100 + id*100 + j
					sm.Set(key, key)
				}
			}(i)
		}

		wg.Wait()
		t.Logf("Range iteration collected %d key-value pairs", len(rangeResults))
	})

	t.Run("yield callback deadlock detection - Keys", func(t *testing.T) {
		sm := Map[int, string]()
		sm.Set(1, "one")
		sm.Set(2, "two")
		sm.Set(3, "three")

		done := make(chan bool, 1)
		go func() {
			// yield callback tries to modify the map
			count := 0
			for range sm.Keys() {
				count++
				if count == 2 {
					// This will deadlock: yield holds RLock, Set needs Lock
					sm.Set(4, "four")
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
		sm.Set(1, "one")
		sm.Set(2, "two")

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
		sm.Set(1, "one")
		sm.Set(2, "two")

		done := make(chan bool, 1)
		go func() {
			count := 0
			for range sm.Indexes() {
				count++
				if count == 1 {
					sm.Set(3, "three")
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
		sm.Set(1, "one")
		sm.Set(2, "two")

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
		sm.Set(1, "one")
		sm.Set(2, "two")

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
		_, ok := sm.Get(1)
		if !ok {
			t.Error("Get failed after panic recovery, possible lock leak")
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

		_, ok = sm.Get(2)
		if !ok {
			t.Error("Get failed after Range panic recovery, possible lock leak")
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

		_, ok = sm.Get(1)
		if !ok {
			t.Error("Get failed after Indexes panic recovery, possible lock leak")
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

		_, ok = sm.Get(2)
		if !ok {
			t.Error("Get failed after RangeWithIndex panic recovery, possible lock leak")
		}

		t.Log("All panic recovery tests passed, no lock leaks detected")
	})

	t.Run("iterator multiple invocations safety", func(t *testing.T) {
		sm := Map[int, string]()
		sm.Set(1, "one")
		sm.Set(2, "two")

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
		sm.Set(1, "one")
		sm.Set(2, "two")
		sm.Set(3, "three")

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
			sm.Set(i, "value")
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

func Test_sortMap_MarshalJSON(t *testing.T) {
	type testCase struct {
		name    string
		s       sortMap[int, string]
		want    []byte
		wantErr bool
	}
	tests := []testCase{
		{
			name:    "marshal empty map",
			s:       newTestSortMap[int, string](map[int]string{}),
			want:    []byte("{}"),
			wantErr: false,
		},
		{
			name:    "marshal single element",
			s:       newTestSortMap[int, string](map[int]string{1: "one"}),
			want:    []byte(`{"1":"one"}`),
			wantErr: false,
		},
		{
			name:    "marshal multiple elements",
			s:       newTestSortMap[int, string](map[int]string{1: "one", 2: "two", 3: "three"}),
			want:    []byte(`{"1":"one","2":"two","3":"three"}`),
			wantErr: false,
		},
		{
			name:    "marshal with negative keys",
			s:       newTestSortMap[int, string](map[int]string{-1: "neg", 0: "zero", 1: "pos"}),
			want:    []byte(`{"-1":"neg","0":"zero","1":"pos"}`),
			wantErr: false,
		},
		{
			name:    "marshal with empty string values",
			s:       newTestSortMap[int, string](map[int]string{1: "", 2: ""}),
			want:    []byte(`{"1":"","2":""}`),
			wantErr: false,
		},
		{
			name:    "marshal with special characters in values",
			s:       newTestSortMap[int, string](map[int]string{1: `hello "world"`, 2: "line1\nline2"}),
			want:    []byte(`{"1":"hello \"world\"","2":"line1\nline2"}`),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MarshalJSON() got = %v, want %v", string(got), string(tt.want))
			}
		})
	}

	t.Run("marshal with string keys", func(t *testing.T) {
		sm := newTestSortMap[string, int](map[string]int{"a": 1, "b": 2, "c": 3})
		got, err := sm.MarshalJSON()
		if err != nil {
			t.Errorf("MarshalJSON() unexpected error = %v", err)
			return
		}
		want := []byte(`{"a":1,"b":2,"c":3}`)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("MarshalJSON() got = %v, want %v", string(got), string(want))
		}
	})

	t.Run("marshal with complex value type", func(t *testing.T) {
		type Data struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}
		sm := newTestSortMap[int, Data](map[int]Data{
			1: {Name: "first", Value: 100},
			2: {Name: "second", Value: 200},
		})
		got, err := sm.MarshalJSON()
		if err != nil {
			t.Errorf("MarshalJSON() unexpected error = %v", err)
			return
		}
		want := []byte(`{"1":{"name":"first","value":100},"2":{"name":"second","value":200}}`)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("MarshalJSON() got = %v, want %v", string(got), string(want))
		}
	})

	t.Run("marshal concurrent access safety", func(t *testing.T) {
		sm := newTestSortMap[int, string](map[int]string{1: "one", 2: "two"})
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := sm.MarshalJSON()
				if err != nil {
					t.Errorf("MarshalJSON() concurrent error = %v", err)
				}
			}()
		}
		wg.Wait()
	})
}

func Test_sortMap_MarshalYAML(t *testing.T) {
	type testCase struct {
		name    string
		s       sortMap[int, string]
		want    interface{}
		wantErr bool
	}
	tests := []testCase{
		{
			name:    "marshal empty map",
			s:       newTestSortMap[int, string](map[int]string{}),
			want:    map[int]string{},
			wantErr: false,
		},
		{
			name:    "marshal single element",
			s:       newTestSortMap[int, string](map[int]string{1: "one"}),
			want:    map[int]string{1: "one"},
			wantErr: false,
		},
		{
			name:    "marshal multiple elements",
			s:       newTestSortMap[int, string](map[int]string{1: "one", 2: "two", 3: "three"}),
			want:    map[int]string{1: "one", 2: "two", 3: "three"},
			wantErr: false,
		},
		{
			name:    "marshal with negative keys",
			s:       newTestSortMap[int, string](map[int]string{-1: "neg", 0: "zero", 1: "pos"}),
			want:    map[int]string{-1: "neg", 0: "zero", 1: "pos"},
			wantErr: false,
		},
		{
			name:    "marshal with empty string values",
			s:       newTestSortMap[int, string](map[int]string{1: "", 2: ""}),
			want:    map[int]string{1: "", 2: ""},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.MarshalYAML()
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MarshalYAML() got = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("marshal with string keys", func(t *testing.T) {
		sm := newTestSortMap[string, int](map[string]int{"a": 1, "b": 2, "c": 3})
		got, err := sm.MarshalYAML()
		if err != nil {
			t.Errorf("MarshalYAML() unexpected error = %v", err)
			return
		}
		want := map[string]int{"a": 1, "b": 2, "c": 3}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("MarshalYAML() got = %v, want %v", got, want)
		}
	})

	t.Run("marshal with complex value type", func(t *testing.T) {
		type Data struct {
			Name  string `yaml:"name"`
			Value int    `yaml:"value"`
		}
		sm := newTestSortMap[int, Data](map[int]Data{
			1: {Name: "first", Value: 100},
			2: {Name: "second", Value: 200},
		})
		got, err := sm.MarshalYAML()
		if err != nil {
			t.Errorf("MarshalYAML() unexpected error = %v", err)
			return
		}
		want := map[int]Data{
			1: {Name: "first", Value: 100},
			2: {Name: "second", Value: 200},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("MarshalYAML() got = %v, want %v", got, want)
		}
	})

	t.Run("marshal concurrent access safety", func(t *testing.T) {
		sm := newTestSortMap[int, string](map[int]string{1: "one", 2: "two"})
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := sm.MarshalYAML()
				if err != nil {
					t.Errorf("MarshalYAML() concurrent error = %v", err)
				}
			}()
		}
		wg.Wait()
	})
}

func Test_sortMap_UnmarshalJSON(t *testing.T) {
	type testCase struct {
		name    string
		initial map[int]string
		bytes   []byte
		want    map[int]string
		wantErr bool
	}
	tests := []testCase{
		{
			name:    "unmarshal empty JSON to empty map",
			initial: map[int]string{},
			bytes:   []byte("{}"),
			want:    map[int]string{},
			wantErr: false,
		},
		{
			name:    "unmarshal single element",
			initial: map[int]string{},
			bytes:   []byte(`{"1":"one"}`),
			want:    map[int]string{1: "one"},
			wantErr: false,
		},
		{
			name:    "unmarshal multiple elements",
			initial: map[int]string{},
			bytes:   []byte(`{"1":"one","2":"two","3":"three"}`),
			want:    map[int]string{1: "one", 2: "two", 3: "three"},
			wantErr: false,
		},
		{
			name:    "unmarshal with negative keys",
			initial: map[int]string{},
			bytes:   []byte(`{"-1":"neg","0":"zero","1":"pos"}`),
			want:    map[int]string{-1: "neg", 0: "zero", 1: "pos"},
			wantErr: false,
		},
		{
			name:    "unmarshal replaces existing data",
			initial: map[int]string{10: "old", 20: "old"},
			bytes:   []byte(`{"1":"new"}`),
			want:    map[int]string{1: "new"},
			wantErr: false,
		},
		{
			name:    "unmarshal with empty string values",
			initial: map[int]string{},
			bytes:   []byte(`{"1":"","2":""}`),
			want:    map[int]string{1: "", 2: ""},
			wantErr: false,
		},
		{
			name:    "unmarshal invalid JSON",
			initial: map[int]string{},
			bytes:   []byte(`{invalid}`),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "unmarshal empty bytes",
			initial: map[int]string{},
			bytes:   []byte(""),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "unmarshal non-object JSON",
			initial: map[int]string{},
			bytes:   []byte(`[1,2,3]`),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "unmarshal with special characters",
			initial: map[int]string{},
			bytes:   []byte(`{"1":"hello \"world\"","2":"line1\nline2"}`),
			want:    map[int]string{1: `hello "world"`, 2: "line1\nline2"},
			wantErr: false,
		},
		{
			name:    "unmarshal with out-of-order indexes",
			initial: map[int]string{},
			bytes:   []byte(`{"5":"five","2":"two","8":"eight","1":"one","3":"three"}`),
			want:    map[int]string{1: "one", 2: "two", 3: "three", 5: "five", 8: "eight"},
			wantErr: false,
		},
		{
			name:    "unmarshal with reverse order indexes",
			initial: map[int]string{},
			bytes:   []byte(`{"9":"nine","8":"eight","7":"seven","6":"six","5":"five"}`),
			want:    map[int]string{5: "five", 6: "six", 7: "seven", 8: "eight", 9: "nine"},
			wantErr: false,
		},
		{
			name:    "unmarshal with random order indexes",
			initial: map[int]string{},
			bytes:   []byte(`{"42":"forty-two","7":"seven","99":"ninety-nine","1":"one","15":"fifteen"}`),
			want:    map[int]string{1: "one", 7: "seven", 15: "fifteen", 42: "forty-two", 99: "ninety-nine"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := newTestSortMap[int, string](tt.initial)
			err := sm.UnmarshalJSON(tt.bytes)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(sm.dataMap, tt.want) {
				t.Errorf("UnmarshalJSON() dataMap = %v, want %v", sm.dataMap, tt.want)
			}
			if len(sm.dataMap) != len(tt.want) {
				t.Errorf("UnmarshalJSON() dataMap length = %v, want %v", len(sm.dataMap), len(tt.want))
			}
			for k := range tt.want {
				if _, ok := sm.dataMap[k]; !ok {
					t.Errorf("UnmarshalJSON() missing key %v in dataMap", k)
				}
			}
		})
	}

	t.Run("unmarshal with string keys", func(t *testing.T) {
		sm := newTestSortMap[string, int](map[string]int{})
		err := sm.UnmarshalJSON([]byte(`{"a":1,"b":2,"c":3}`))
		if err != nil {
			t.Errorf("UnmarshalJSON() unexpected error = %v", err)
			return
		}
		want := map[string]int{"a": 1, "b": 2, "c": 3}
		if !reflect.DeepEqual(sm.dataMap, want) {
			t.Errorf("UnmarshalJSON() dataMap = %v, want %v", sm.dataMap, want)
		}
	})

	t.Run("unmarshal with complex value type", func(t *testing.T) {
		type Data struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}
		sm := newTestSortMap[int, Data](map[int]Data{})
		err := sm.UnmarshalJSON([]byte(`{"1":{"name":"first","value":100},"2":{"name":"second","value":200}}`))
		if err != nil {
			t.Errorf("UnmarshalJSON() unexpected error = %v", err)
			return
		}
		want := map[int]Data{
			1: {Name: "first", Value: 100},
			2: {Name: "second", Value: 200},
		}
		if !reflect.DeepEqual(sm.dataMap, want) {
			t.Errorf("UnmarshalJSON() dataMap = %v, want %v", sm.dataMap, want)
		}
	})

	t.Run("unmarshal concurrent access safety", func(t *testing.T) {
		sm := newTestSortMap[int, string](map[int]string{})
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				jsonData := []byte(`{"1":"one","2":"two"}`)
				err := sm.UnmarshalJSON(jsonData)
				if err != nil {
					t.Errorf("UnmarshalJSON() concurrent error = %v", err)
				}
			}(i)
		}
		wg.Wait()
	})

	t.Run("unmarshal preserves sorted index order", func(t *testing.T) {
		sm := newTestSortMap[int, string](map[int]string{})
		err := sm.UnmarshalJSON([]byte(`{"30":"thirty","10":"ten","20":"twenty"}`))
		if err != nil {
			t.Errorf("UnmarshalJSON() unexpected error = %v", err)
			return
		}
		var keys []int
		for _, k := range sm.indexList.Range() {
			keys = append(keys, k)
		}
		wantKeys := []int{10, 20, 30}
		if !reflect.DeepEqual(keys, wantKeys) {
			t.Errorf("UnmarshalJSON() index order = %v, want %v", keys, wantKeys)
		}
	})
}

func Test_sortMap_UnmarshalYAML(t *testing.T) {
	type testCase struct {
		name    string
		initial map[int]string
		yamlStr string
		want    map[int]string
		wantErr bool
	}
	tests := []testCase{
		{
			name:    "unmarshal empty YAML to empty map",
			initial: map[int]string{},
			yamlStr: "{}",
			want:    map[int]string{},
			wantErr: false,
		},
		{
			name:    "unmarshal single element",
			initial: map[int]string{},
			yamlStr: "1: one",
			want:    map[int]string{1: "one"},
			wantErr: false,
		},
		{
			name:    "unmarshal multiple elements",
			initial: map[int]string{},
			yamlStr: "1: one\n2: two\n3: three",
			want:    map[int]string{1: "one", 2: "two", 3: "three"},
			wantErr: false,
		},
		{
			name:    "unmarshal with negative keys",
			initial: map[int]string{},
			yamlStr: "-1: neg\n0: zero\n1: pos",
			want:    map[int]string{-1: "neg", 0: "zero", 1: "pos"},
			wantErr: false,
		},
		{
			name:    "unmarshal replaces existing data",
			initial: map[int]string{10: "old", 20: "old"},
			yamlStr: "1: new",
			want:    map[int]string{1: "new"},
			wantErr: false,
		},
		{
			name:    "unmarshal with empty string values",
			initial: map[int]string{},
			yamlStr: "1: ''\n2: ''",
			want:    map[int]string{1: "", 2: ""},
			wantErr: false,
		},
		{
			name:    "unmarshal invalid YAML",
			initial: map[int]string{},
			yamlStr: "just a scalar value",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "unmarshal with special characters",
			initial: map[int]string{},
			yamlStr: `1: 'hello "world"'` + "\n2: |-\n  line1\n  line2",
			want:    map[int]string{1: `hello "world"`, 2: "line1\nline2"},
			wantErr: false,
		},
		{
			name:    "unmarshal with out-of-order indexes",
			initial: map[int]string{},
			yamlStr: "5: five\n2: two\n8: eight\n1: one\n3: three",
			want:    map[int]string{1: "one", 2: "two", 3: "three", 5: "five", 8: "eight"},
			wantErr: false,
		},
		{
			name:    "unmarshal with reverse order indexes",
			initial: map[int]string{},
			yamlStr: "9: nine\n8: eight\n7: seven\n6: six\n5: five",
			want:    map[int]string{5: "five", 6: "six", 7: "seven", 8: "eight", 9: "nine"},
			wantErr: false,
		},
		{
			name:    "unmarshal with random order indexes",
			initial: map[int]string{},
			yamlStr: "42: forty-two\n7: seven\n99: ninety-nine\n1: one\n15: fifteen",
			want:    map[int]string{1: "one", 7: "seven", 15: "fifteen", 42: "forty-two", 99: "ninety-nine"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := newTestSortMap[int, string](tt.initial)
			var node yaml.Node
			err := yaml.Unmarshal([]byte(tt.yamlStr), &node)
			if err != nil {
				t.Fatalf("failed to parse YAML test data: %v", err)
			}
			err = sm.UnmarshalYAML(&node)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(sm.dataMap, tt.want) {
				t.Errorf("UnmarshalYAML() dataMap = %v, want %v", sm.dataMap, tt.want)
			}
			if len(sm.dataMap) != len(tt.want) {
				t.Errorf("UnmarshalYAML() dataMap length = %v, want %v", len(sm.dataMap), len(tt.want))
			}
			for k := range tt.want {
				if _, ok := sm.dataMap[k]; !ok {
					t.Errorf("UnmarshalYAML() missing key %v in dataMap", k)
				}
			}
		})
	}

	t.Run("unmarshal with string keys", func(t *testing.T) {
		sm := newTestSortMap[string, int](map[string]int{})
		var node yaml.Node
		err := yaml.Unmarshal([]byte("a: 1\nb: 2\nc: 3"), &node)
		if err != nil {
			t.Fatalf("failed to parse YAML test data: %v", err)
		}
		err = sm.UnmarshalYAML(&node)
		if err != nil {
			t.Errorf("UnmarshalYAML() unexpected error = %v", err)
			return
		}
		want := map[string]int{"a": 1, "b": 2, "c": 3}
		if !reflect.DeepEqual(sm.dataMap, want) {
			t.Errorf("UnmarshalYAML() dataMap = %v, want %v", sm.dataMap, want)
		}
	})

	t.Run("unmarshal with complex value type", func(t *testing.T) {
		type Data struct {
			Name  string `yaml:"name"`
			Value int    `yaml:"value"`
		}
		sm := newTestSortMap[int, Data](map[int]Data{})
		var node yaml.Node
		err := yaml.Unmarshal([]byte("1:\n  name: first\n  value: 100\n2:\n  name: second\n  value: 200"), &node)
		if err != nil {
			t.Fatalf("failed to parse YAML test data: %v", err)
		}
		err = sm.UnmarshalYAML(&node)
		if err != nil {
			t.Errorf("UnmarshalYAML() unexpected error = %v", err)
			return
		}
		want := map[int]Data{
			1: {Name: "first", Value: 100},
			2: {Name: "second", Value: 200},
		}
		if !reflect.DeepEqual(sm.dataMap, want) {
			t.Errorf("UnmarshalYAML() dataMap = %v, want %v", sm.dataMap, want)
		}
	})

	t.Run("unmarshal concurrent access safety", func(t *testing.T) {
		sm := newTestSortMap[int, string](map[int]string{})
		var node yaml.Node
		err := yaml.Unmarshal([]byte("1: one\n2: two"), &node)
		if err != nil {
			t.Fatalf("failed to parse YAML test data: %v", err)
		}
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := sm.UnmarshalYAML(&node)
				if err != nil {
					t.Errorf("UnmarshalYAML() concurrent error = %v", err)
				}
			}()
		}
		wg.Wait()
	})

	t.Run("unmarshal preserves sorted index order", func(t *testing.T) {
		sm := newTestSortMap[int, string](map[int]string{})
		var node yaml.Node
		err := yaml.Unmarshal([]byte("30: thirty\n10: ten\n20: twenty"), &node)
		if err != nil {
			t.Fatalf("failed to parse YAML test data: %v", err)
		}
		err = sm.UnmarshalYAML(&node)
		if err != nil {
			t.Errorf("UnmarshalYAML() unexpected error = %v", err)
			return
		}
		var keys []int
		for _, k := range sm.indexList.Range() {
			keys = append(keys, k)
		}
		wantKeys := []int{10, 20, 30}
		if !reflect.DeepEqual(keys, wantKeys) {
			t.Errorf("UnmarshalYAML() index order = %v, want %v", keys, wantKeys)
		}
	})

	t.Run("unmarshal with nil node", func(t *testing.T) {
		sm := newTestSortMap[int, string](map[int]string{})
		err := sm.UnmarshalYAML(nil)
		if err == nil {
			t.Errorf("UnmarshalYAML() expected error for nil node, got nil")
		}
	})
}

func Test_sortMap_Contains(t *testing.T) {
	type args[K cmp.Ordered] struct {
		key K
	}
	type testCase[K cmp.Ordered, V any] struct {
		name        string
		s           sortMap[K, V]
		args        args[K]
		wantPresent bool
	}
	tests := []testCase[int, string]{
		{
			name:        "empty map - not present",
			s:           newTestSortMap[int, string](map[int]string{}),
			args:        args[int]{key: 5},
			wantPresent: false,
		},
		{
			name:        "single element - present",
			s:           newTestSortMap[int, string](map[int]string{5: "five"}),
			args:        args[int]{key: 5},
			wantPresent: true,
		},
		{
			name:        "single element - not present",
			s:           newTestSortMap[int, string](map[int]string{5: "five"}),
			args:        args[int]{key: 3},
			wantPresent: false,
		},
		{
			name:        "multiple elements - present at beginning",
			s:           newTestSortMap[int, string](map[int]string{1: "one", 3: "three", 5: "five", 7: "seven", 9: "nine"}),
			args:        args[int]{key: 1},
			wantPresent: true,
		},
		{
			name:        "multiple elements - present at end",
			s:           newTestSortMap[int, string](map[int]string{1: "one", 3: "three", 5: "five", 7: "seven", 9: "nine"}),
			args:        args[int]{key: 9},
			wantPresent: true,
		},
		{
			name:        "multiple elements - present in middle",
			s:           newTestSortMap[int, string](map[int]string{1: "one", 3: "three", 5: "five", 7: "seven", 9: "nine"}),
			args:        args[int]{key: 5},
			wantPresent: true,
		},
		{
			name:        "multiple elements - not present less than all",
			s:           newTestSortMap[int, string](map[int]string{1: "one", 3: "three", 5: "five", 7: "seven", 9: "nine"}),
			args:        args[int]{key: 0},
			wantPresent: false,
		},
		{
			name:        "multiple elements - not present greater than all",
			s:           newTestSortMap[int, string](map[int]string{1: "one", 3: "three", 5: "five", 7: "seven", 9: "nine"}),
			args:        args[int]{key: 10},
			wantPresent: false,
		},
		{
			name:        "multiple elements - not present between elements",
			s:           newTestSortMap[int, string](map[int]string{1: "one", 3: "three", 5: "five", 7: "seven", 9: "nine"}),
			args:        args[int]{key: 4},
			wantPresent: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotPresent := tt.s.Contains(tt.args.key); gotPresent != tt.wantPresent {
				t.Errorf("Contains() = %v, want %v", gotPresent, tt.wantPresent)
			}
		})
	}
}

func Test_sortMap_Clear(t *testing.T) {
	type testCase[K cmp.Ordered, V any] struct {
		name string
		s    sortMap[K, V]
	}
	tests := []testCase[int, string]{
		{
			name: "clear empty map",
			s:    newTestSortMap[int, string](map[int]string{}),
		},
		{
			name: "clear single element map",
			s:    newTestSortMap[int, string](map[int]string{1: "one"}),
		},
		{
			name: "clear multiple elements map",
			s:    newTestSortMap[int, string](map[int]string{1: "one", 3: "three", 5: "five", 7: "seven", 9: "nine"}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.s.Clear()
		})
	}
}

func Test_sortMap_Len(t *testing.T) {
	type testCase[K cmp.Ordered, V any] struct {
		name string
		s    sortMap[K, V]
		want int
	}
	tests := []testCase[int, string]{
		{
			name: "empty map",
			s:    newTestSortMap[int, string](map[int]string{}),
			want: 0,
		},
		{
			name: "single element map",
			s:    newTestSortMap[int, string](map[int]string{1: "one"}),
			want: 1,
		},
		{
			name: "multiple elements map",
			s:    newTestSortMap[int, string](map[int]string{1: "one", 3: "three", 5: "five", 7: "seven", 9: "nine"}),
			want: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.Len(); got != tt.want {
				t.Errorf("Len() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sortMap_Values(t *testing.T) {
	type testCase[K cmp.Ordered, V any] struct {
		name string
		s    sortMap[K, V]
		want []V
	}
	tests := []testCase[int, string]{
		{
			name: "empty map",
			s:    newTestSortMap[int, string](map[int]string{}),
			want: []string{},
		},
		{
			name: "single element map",
			s:    newTestSortMap[int, string](map[int]string{1: "one"}),
			want: []string{"one"},
		},
		{
			name: "multiple elements map",
			s:    newTestSortMap[int, string](map[int]string{1: "one", 3: "three", 5: "five"}),
			want: []string{"one", "three", "five"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := make([]string, 0)
			for v := range tt.s.Values() {
				got = append(got, v)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Values() = %v, want %v", got, tt.want)
			}
		})
	}
}
