package sort_map

import (
	"bytes"
	"cmp"
	"encoding/json"
	"iter"
	"slices"
	"strings"
	"sync"
	"testing"

	"gopkg.in/yaml.v3"
)

// Compile-time proof that the exported concrete types satisfy the JSON/YAML serializer
// interfaces relied on by the round-trip tests: the value types carry the value-receiver
// Marshal methods, the pointer types additionally carry Unmarshal.
var (
	_ yaml.Marshaler   = SortMap[string, string]{}
	_ yaml.Unmarshaler = (*SortMap[string, string])(nil)
	_ json.Marshaler   = SortMap[string, string]{}
	_ json.Unmarshaler = (*SortMap[string, string])(nil)
	_ yaml.Marshaler   = MapList[int]{}
	_ yaml.Unmarshaler = (*MapList[int])(nil)
	_ json.Marshaler   = MapList[int]{}
	_ json.Unmarshaler = (*MapList[int])(nil)
)

// nestedTask simulates how business objects (e.g. a task definition) embed the concrete
// SortMap/MapList containers as fields and decode the whole object with a single yaml.Unmarshal call.
type nestedTask struct {
	Name     string                  `yaml:"name" json:"name"`
	Params   SortMap[string, string] `yaml:"params" json:"params"`
	Ext      *SortMap[string, int]   `yaml:"ext" json:"ext"`
	Slots    MapList[string]         `yaml:"slots" json:"slots"`
	PtrSlots *MapList[int]           `yaml:"ptr_slots" json:"ptr_slots"`
}

const nestedTaskYAML = `name: demo
params:
  b: "2"
  a: "1"
ext:
  x: 10
  y: 20
slots:
  5: five
  0: zero
ptr_slots:
  1: 100
`

func collectKeys[K cmp.Ordered](seq iter.Seq[K]) []K {
	var out []K
	for k := range seq {
		out = append(out, k)
	}
	return out
}

func TestYAMLNested_DecodeIntoConcreteTypeFieldsTriggersUnmarshalYAML(t *testing.T) {
	task := nestedTask{}
	if err := yaml.Unmarshal([]byte(nestedTaskYAML), &task); err != nil {
		t.Fatalf("yaml.Unmarshal() unexpected error = %v", err)
	}

	// The fields hold SortMap/MapList values: a generic (untyped) decode could never
	// produce them, so their presence proves UnmarshalYAML was dispatched by yaml.v3.
	if task.Name != "demo" {
		t.Errorf("Name = %q, want %q", task.Name, "demo")
	}
	if got := task.Params.Len(); got != 2 {
		t.Fatalf("Params.Len() = %d, want 2", got)
	}
	v, ok := task.Params.Get("a")
	if !ok || v != "1" {
		t.Errorf(`Params.Get("a") = %q, %v, want "1", true`, v, ok)
	}
	if got, want := collectKeys(task.Params.Keys()), []string{"a", "b"}; !slices.Equal(got, want) {
		t.Errorf("Params key order = %v, want %v", got, want)
	}

	if task.Ext == nil {
		t.Fatal("Ext should be allocated by the decoder")
	}
	if ev, ok := task.Ext.Get("x"); !ok || ev != 10 {
		t.Errorf(`Ext.Get("x") = %d, %v, want 10, true`, ev, ok)
	}
	if got, want := collectKeys(task.Ext.Keys()), []string{"x", "y"}; !slices.Equal(got, want) {
		t.Errorf("Ext key order = %v, want %v", got, want)
	}

	if got := task.Slots.Len(); got != 2 {
		t.Fatalf("Slots.Len() = %d, want 2", got)
	}
	var slotIndexes []int
	for i, _ := range task.Slots.Range() {
		slotIndexes = append(slotIndexes, i)
	}
	if want := []int{0, 5}; !slices.Equal(slotIndexes, want) {
		t.Errorf("Slots index order = %v, want %v", slotIndexes, want)
	}
	if sv, ok := task.Slots.Get(0); !ok || sv != "zero" {
		t.Errorf(`Slots.Get(0) = %q, %v, want "zero", true`, sv, ok)
	}

	if task.PtrSlots == nil {
		t.Fatal("PtrSlots should be allocated by the decoder")
	}
	if pv, ok := task.PtrSlots.Get(1); !ok || pv != 100 {
		t.Errorf("PtrSlots.Get(1) = %d, %v, want 100, true", pv, ok)
	}
}

func TestYAMLNested_ReuseAlreadyInitializedFieldReplacesContent(t *testing.T) {
	task := nestedTask{}
	task.Params.Set("old", "x")
	task.Slots.Append("stale")

	if err := yaml.Unmarshal([]byte("params:\n  a: \"1\"\nslots:\n  3: three\n"), &task); err != nil {
		t.Fatalf("yaml.Unmarshal() unexpected error = %v", err)
	}

	if _, ok := task.Params.Get("old"); ok {
		t.Error(`Params should not keep key "old" after a full replace`)
	}
	if got := task.Params.Len(); got != 1 {
		t.Errorf("Params.Len() = %d, want 1", got)
	}
	if _, ok := task.Slots.Get(0); ok {
		t.Error("Slots should not keep index 0 after a full replace")
	}
	if sv, ok := task.Slots.Get(3); !ok || sv != "three" {
		t.Errorf(`Slots.Get(3) = %q, %v, want "three", true`, sv, ok)
	}
}

func TestYAMLNested_EmptyInputAndMissingFields(t *testing.T) {
	// Completely empty YAML: nothing is touched, no error.
	task := nestedTask{Name: "keep"}
	if err := yaml.Unmarshal([]byte(""), &task); err != nil {
		t.Fatalf("yaml.Unmarshal(empty) unexpected error = %v", err)
	}
	if task.Name != "keep" {
		t.Errorf("Name = %q, want %q", task.Name, "keep")
	}

	// Missing container fields keep the zero value; zero-value reads must not panic.
	if err := yaml.Unmarshal([]byte("name: only\n"), &task); err != nil {
		t.Fatalf("yaml.Unmarshal() unexpected error = %v", err)
	}
	if got := task.Params.Len(); got != 0 {
		t.Errorf("Params.Len() = %d, want 0", got)
	}
	if got := collectKeys(task.Params.Keys()); len(got) != 0 {
		t.Errorf("Params.Keys() = %v, want empty", got)
	}
	if task.Ext != nil {
		t.Error("Ext should stay nil when absent")
	}
	if got := task.Slots.Len(); got != 0 {
		t.Errorf("Slots.Len() = %d, want 0", got)
	}
	for range task.Slots.Indexes() {
		t.Error("Slots.Indexes() should yield nothing on zero value")
	}

	// Empty flow mapping resets/initializes to an empty container without error.
	if err := yaml.Unmarshal([]byte("params: {}\n"), &task); err != nil {
		t.Fatalf("yaml.Unmarshal(params: {}) unexpected error = %v", err)
	}
	if got := task.Params.Len(); got != 0 {
		t.Errorf("Params.Len() = %d, want 0", got)
	}
}

func TestYAMLNested_NullNodes(t *testing.T) {
	// null on pointer fields: decoded as nil pointers, no error.
	task := nestedTask{}
	if err := yaml.Unmarshal([]byte("ext: null\nptr_slots: null\n"), &task); err != nil {
		t.Fatalf("yaml.Unmarshal() unexpected error = %v", err)
	}
	if task.Ext != nil {
		t.Error("Ext should be nil for a null node")
	}
	if task.PtrSlots != nil {
		t.Error("PtrSlots should be nil for a null node")
	}

	// null on a value-typed container field: yaml.v3 short-circuits null in prepare() and
	// then routes it through d.null(), which only zeroes Interface/Ptr/Map/Slice values.
	// A struct-typed field is left untouched with no error, and UnmarshalYAML is never
	// dispatched for null. So value fields keep their (zero) content; use pointer fields
	// when a null must be observable (see the pointer assertions above).
	nullOnValue := nestedTask{}
	if err := yaml.Unmarshal([]byte("params: null\nslots: null\n"), &nullOnValue); err != nil {
		t.Fatalf("yaml.Unmarshal(null into value-typed fields) unexpected error = %v", err)
	}
	if got := nullOnValue.Params.Len(); got != 0 {
		t.Errorf("Params.Len() after null on value field = %d, want 0", got)
	}
	if got := nullOnValue.Slots.Len(); got != 0 {
		t.Errorf("Slots.Len() after null on value field = %d, want 0", got)
	}

	// Direct UnmarshalYAML with an explicit null node resets to an empty container.
	sm := newSortMap[string, string]()
	sm.Set("k", "v")
	var nullNode yaml.Node
	if err := yaml.Unmarshal([]byte("null"), &nullNode); err != nil {
		t.Fatalf("failed to parse null node: %v", err)
	}
	if err := sm.UnmarshalYAML(&nullNode); err != nil {
		t.Fatalf("UnmarshalYAML(null node) unexpected error = %v", err)
	}
	if got := sm.Len(); got != 0 {
		t.Errorf("Params.Len() after null = %d, want 0", got)
	}

	ml := &MapList[int]{}
	ml.Append(1)
	if err := ml.UnmarshalYAML(&nullNode); err != nil {
		t.Fatalf("MapList.UnmarshalYAML(null node) unexpected error = %v", err)
	}
	if got := ml.Len(); got != 0 {
		t.Errorf("MapList.Len() after null = %d, want 0", got)
	}

	// A zero-value Node (empty document) behaves like a null node as well.
	sm2 := newSortMap[string, string]()
	sm2.Set("k", "v")
	if err := sm2.UnmarshalYAML(&yaml.Node{}); err != nil {
		t.Fatalf("UnmarshalYAML(zero node) unexpected error = %v", err)
	}
	if got := sm2.Len(); got != 0 {
		t.Errorf("Params.Len() after zero node = %d, want 0", got)
	}

	// Truly nil nodes still error (for both types).
	if err := sm2.UnmarshalYAML(nil); err == nil {
		t.Error("SortMap.UnmarshalYAML(nil) should return an error")
	}
	if err := (&MapList[int]{}).UnmarshalYAML(nil); err == nil {
		t.Error("MapList.UnmarshalYAML(nil) should return an error")
	}
}

func TestYAMLNested_MapListNodeKinds(t *testing.T) {
	t.Run("sequence expands to 0..n-1", func(t *testing.T) {
		task := nestedTask{}
		if err := yaml.Unmarshal([]byte("slots:\n  - a\n  - b\n"), &task); err != nil {
			t.Fatalf("yaml.Unmarshal() unexpected error = %v", err)
		}
		if got := task.Slots.Len(); got != 2 {
			t.Fatalf("Slots.Len() = %d, want 2", got)
		}
		if v, ok := task.Slots.Get(0); !ok || v != "a" {
			t.Errorf(`Slots.Get(0) = %q, %v, want "a", true`, v, ok)
		}
		if v, ok := task.Slots.Get(1); !ok || v != "b" {
			t.Errorf(`Slots.Get(1) = %q, %v, want "b", true`, v, ok)
		}
	})

	t.Run("non-sequential out-of-order indexes keep sorted order", func(t *testing.T) {
		task := nestedTask{}
		if err := yaml.Unmarshal([]byte("slots:\n  10: ten\n  0: zero\n  5: five\n"), &task); err != nil {
			t.Fatalf("yaml.Unmarshal() unexpected error = %v", err)
		}
		var idx []int
		for i, _ := range task.Slots.Range() {
			idx = append(idx, i)
		}
		if want := []int{0, 5, 10}; !slices.Equal(idx, want) {
			t.Errorf("Slots index order = %v, want %v", idx, want)
		}
	})

	t.Run("negative index rejected with the offending value", func(t *testing.T) {
		err := yaml.Unmarshal([]byte("slots:\n  0: ok\n  -1: bad\n"), &nestedTask{})
		if err == nil {
			t.Fatal("expected an error for a negative list index")
		}
		if !strings.Contains(err.Error(), "-1") {
			t.Errorf("error %v should contain the offending index -1", err)
		}
	})

	t.Run("scalar node rejected clearly", func(t *testing.T) {
		err := yaml.Unmarshal([]byte("slots: hello\n"), &nestedTask{})
		if err == nil {
			t.Fatal("expected an error for a scalar into a map list")
		}
		if !strings.Contains(err.Error(), "mapping or sequence") {
			t.Errorf("error %v should mention the expected node kinds", err)
		}
	})

	t.Run("negative keys remain legal for SortMap with int keys", func(t *testing.T) {
		sm := SortMap[int, string]{}
		if err := yaml.Unmarshal([]byte("-1: neg\n2: two\n"), &sm); err != nil {
			t.Fatalf("yaml.Unmarshal() unexpected error = %v", err)
		}
		if got, want := collectKeys(sm.Keys()), []int{-1, 2}; !slices.Equal(got, want) {
			t.Errorf("key order = %v, want %v", got, want)
		}
	})

	t.Run("sequence rejected for SortMap", func(t *testing.T) {
		sm := SortMap[string, string]{}
		err := yaml.Unmarshal([]byte("- a\n- b\n"), &sm)
		if err == nil {
			t.Fatal("expected an error for a sequence into a sort map")
		}
		if !strings.Contains(err.Error(), "mapping") {
			t.Errorf("error %v should mention the expected mapping node", err)
		}
	})
}

func TestYAMLNested_JSONYAMLRoundTripConsistency(t *testing.T) {
	task := nestedTask{}
	if err := yaml.Unmarshal([]byte(nestedTaskYAML), &task); err != nil {
		t.Fatalf("yaml.Unmarshal() unexpected error = %v", err)
	}

	ydata, err := yaml.Marshal(&task)
	if err != nil {
		t.Fatalf("yaml.Marshal() unexpected error = %v", err)
	}
	jdata, err := json.Marshal(&task)
	if err != nil {
		t.Fatalf("json.Marshal() unexpected error = %v", err)
	}

	viaJSON := nestedTask{}
	if err := json.Unmarshal(jdata, &viaJSON); err != nil {
		t.Fatalf("json.Unmarshal() unexpected error = %v", err)
	}
	ydata2, err := yaml.Marshal(&viaJSON)
	if err != nil {
		t.Fatalf("yaml.Marshal() after JSON round trip unexpected error = %v", err)
	}
	if !bytes.Equal(ydata, ydata2) {
		t.Errorf("YAML output differs after a JSON round trip:\nfirst:\n%s\nsecond:\n%s", ydata, ydata2)
	}

	if got, want := collectKeys(viaJSON.Params.Keys()), []string{"a", "b"}; !slices.Equal(got, want) {
		t.Errorf("key order after JSON round trip = %v, want %v", got, want)
	}
	var idx []int
	for i, _ := range viaJSON.Slots.Range() {
		idx = append(idx, i)
	}
	if want := []int{0, 5}; !slices.Equal(idx, want) {
		t.Errorf("Slots index order after JSON round trip = %v, want %v", idx, want)
	}

	// YAML -> JSON -> YAML direct byte comparison.
	viaYAML := nestedTask{}
	if err := yaml.Unmarshal(ydata, &viaYAML); err != nil {
		t.Fatalf("yaml.Unmarshal() unexpected error = %v", err)
	}
	jdata2, err := json.Marshal(&viaYAML)
	if err != nil {
		t.Fatalf("json.Marshal() unexpected error = %v", err)
	}
	if !bytes.Equal(jdata, jdata2) {
		t.Errorf("JSON output differs after a YAML round trip:\nfirst: %s\nsecond: %s", jdata, jdata2)
	}

	// Negative keys survive SortMap[int] round trips in both encoders.
	sm := SortMap[int, string]{}
	sm.Set(-1, "neg")
	sm.Set(2, "two")
	smYAML, err := yaml.Marshal(&sm)
	if err != nil {
		t.Fatalf("yaml.Marshal() unexpected error = %v", err)
	}
	smBack := SortMap[int, string]{}
	if err := yaml.Unmarshal(smYAML, &smBack); err != nil {
		t.Fatalf("yaml.Unmarshal() of marshaled negative keys unexpected error = %v", err)
	}
	if v, ok := smBack.Get(-1); !ok || v != "neg" {
		t.Errorf(`negative key lost after round trip: Get(-1) = %q, %v`, v, ok)
	}

	// MapList keeps rejecting negative indexes consistently in both encoders.
	mlErr := json.Unmarshal([]byte(`{"-1":"x"}`), &MapList[string]{})
	if mlErr == nil || !strings.Contains(mlErr.Error(), "-1") {
		t.Errorf("json negative index error = %v, want it to contain -1", mlErr)
	}
	yamlErr := yaml.Unmarshal([]byte("-1: x\n"), &MapList[string]{})
	if yamlErr == nil || !strings.Contains(yamlErr.Error(), "-1") {
		t.Errorf("yaml negative index error = %v, want it to contain -1", yamlErr)
	}
}

func TestYAMLNested_ValueAndPointerFieldMarshalConsistency(t *testing.T) {
	// A value-typed container field must marshal exactly like a pointer field holding the
	// same content (yaml.v3 does not fall back to Addr() for Marshaler probing).
	sm := SortMap[string, int]{}
	sm.Set("a", 1)
	sm.Set("b", 2)

	valueHolder := struct {
		V SortMap[string, int]  `yaml:"v" json:"v"`
		P *SortMap[string, int] `yaml:"p" json:"p"`
	}{V: sm, P: &sm}

	ydata, err := yaml.Marshal(&valueHolder)
	if err != nil {
		t.Fatalf("yaml.Marshal() unexpected error = %v", err)
	}
	if !strings.Contains(string(ydata), "v:") || !strings.Contains(string(ydata), "a: 1") {
		t.Errorf("value-typed field was not marshaled with its content:\n%s", ydata)
	}

	jdata, err := json.Marshal(&valueHolder)
	if err != nil {
		t.Fatalf("json.Marshal() unexpected error = %v", err)
	}
	if !strings.Contains(string(jdata), `"v":{"a":1,"b":2}`) {
		t.Errorf("value-typed field was not marshaled with its content: %s", jdata)
	}
}

func TestMapList_ConcurrentAppendInsertToEmptyRegression(t *testing.T) {
	ml := List[int]()
	const appliers, inserters, setters = 100, 100, 50

	// Pre-reserve the explicit Set targets (1000..1049) so concurrent Set only overwrites
	// existing indexes and never allocates. Append uses lastFreeIndex (largest+1), so it
	// then always yields >=1050 while InsertToEmpty fills the smallest free indexes (0..99).
	// The three ranges stay disjoint, making the final size deterministic regardless of
	// goroutine scheduling; without this, an Append could land on a not-yet-applied Set
	// target and silently overwrite it (a test-design collision, not an implementation bug).
	for id := 0; id < setters; id++ {
		ml.Set(1000+id, -1)
	}

	var mu sync.Mutex
	seen := make(map[int]int, appliers+inserters)
	duplicates := 0

	mark := func(idx int) {
		mu.Lock()
		defer mu.Unlock()
		if prev, ok := seen[idx]; ok {
			duplicates++
			t.Errorf("index %d allocated twice (by ops %d and %d)", idx, prev, len(seen))
		}
		seen[idx] = len(seen)
	}

	var wg sync.WaitGroup
	for i := 0; i < appliers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mark(ml.Append(i))
		}()
	}
	for i := 0; i < inserters; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mark(ml.InsertToEmpty(i))
		}()
	}
	for i := 0; i < setters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ml.Set(1000+id, id)
		}(i)
	}
	wg.Wait()

	if duplicates != 0 {
		t.Errorf("%d duplicate indexes were allocated", duplicates)
	}
	if got := ml.Len(); got != appliers+inserters+setters {
		t.Errorf("Len() = %d, want %d", got, appliers+inserters+setters)
	}
}

func TestMapList_ConcurrentUseAfterDecodeInitOnZeroValue(t *testing.T) {
	var task nestedTask
	if err := yaml.Unmarshal([]byte("slots: {}\n"), &task); err != nil {
		t.Fatalf("yaml.Unmarshal() unexpected error = %v", err)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	total := 0
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			task.Slots.Append("x")
		}()
		go func() {
			defer wg.Done()
			n := 0
			for range task.Slots.Range() {
				n++
			}
			mu.Lock()
			total += n
			mu.Unlock()
		}()
	}
	wg.Wait()

	if got := task.Slots.Len(); got != 50 {
		t.Errorf("Slots.Len() = %d, want 50", got)
	}
}
