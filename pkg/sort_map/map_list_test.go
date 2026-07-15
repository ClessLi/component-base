package sort_map

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func Test_mapList_UnmarshalJSON(t *testing.T) {
	type testCase struct {
		name    string
		bytes   []byte
		want    map[int]string
		wantErr bool
	}
	tests := []testCase{
		{
			name:    "unmarshal empty JSON",
			bytes:   []byte("{}"),
			want:    map[int]string{},
			wantErr: false,
		},
		{
			name:    "unmarshal single element",
			bytes:   []byte(`{"0":"zero"}`),
			want:    map[int]string{0: "zero"},
			wantErr: false,
		},
		{
			name:    "unmarshal multiple elements",
			bytes:   []byte(`{"0":"zero","1":"one","2":"two"}`),
			want:    map[int]string{0: "zero", 1: "one", 2: "two"},
			wantErr: false,
		},
		{
			name:    "unmarshal with non-sequential indexes",
			bytes:   []byte(`{"0":"zero","5":"five","10":"ten"}`),
			want:    map[int]string{0: "zero", 5: "five", 10: "ten"},
			wantErr: false,
		},
		{
			name:    "unmarshal with out-of-order indexes",
			bytes:   []byte(`{"5":"five","2":"two","8":"eight","1":"one","3":"three"}`),
			want:    map[int]string{1: "one", 2: "two", 3: "three", 5: "five", 8: "eight"},
			wantErr: false,
		},
		{
			name:    "unmarshal with reverse order indexes",
			bytes:   []byte(`{"9":"nine","8":"eight","7":"seven","6":"six","5":"five"}`),
			want:    map[int]string{5: "five", 6: "six", 7: "seven", 8: "eight", 9: "nine"},
			wantErr: false,
		},
		{
			name:    "unmarshal with random order indexes",
			bytes:   []byte(`{"42":"forty-two","7":"seven","99":"ninety-nine","1":"one","15":"fifteen"}`),
			want:    map[int]string{1: "one", 7: "seven", 15: "fifteen", 42: "forty-two", 99: "ninety-nine"},
			wantErr: false,
		},
		{
			name:    "unmarshal with empty string values",
			bytes:   []byte(`{"0":"","1":""}`),
			want:    map[int]string{0: "", 1: ""},
			wantErr: false,
		},
		{
			name:    "unmarshal with special characters",
			bytes:   []byte(`{"0":"hello \"world\"","1":"line1\nline2"}`),
			want:    map[int]string{0: `hello "world"`, 1: "line1\nline2"},
			wantErr: false,
		},
		{
			name:    "reject negative index",
			bytes:   []byte(`{"-1":"negative"}`),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "reject multiple negative indexes",
			bytes:   []byte(`{"-1":"neg1","-2":"neg2"}`),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "reject mixed valid and negative indexes",
			bytes:   []byte(`{"0":"zero","-1":"negative","1":"one"}`),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "unmarshal invalid JSON",
			bytes:   []byte(`{invalid}`),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "unmarshal empty bytes",
			bytes:   []byte(""),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "unmarshal non-object JSON",
			bytes:   []byte(`[1,2,3]`),
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ml := List[string]()
			err := ml.UnmarshalJSON(tt.bytes)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			mlImpl := ml.(*mapList[string])
			if len(mlImpl.dataMap) != len(tt.want) {
				t.Errorf("UnmarshalJSON() dataMap length = %v, want %v", len(mlImpl.dataMap), len(tt.want))
			}
			for k, v := range tt.want {
				gotV, ok := mlImpl.dataMap[k]
				if !ok {
					t.Errorf("UnmarshalJSON() missing key %v in dataMap", k)
				}
				if gotV != v {
					t.Errorf("UnmarshalJSON() dataMap[%v] = %v, want %v", k, gotV, v)
				}
			}
		})
	}

	t.Run("unmarshal with complex value type", func(t *testing.T) {
		type Data struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}
		ml := List[Data]()
		err := ml.UnmarshalJSON([]byte(`{"0":{"name":"first","value":100},"1":{"name":"second","value":200}}`))
		if err != nil {
			t.Errorf("UnmarshalJSON() unexpected error = %v", err)
			return
		}
		mlImpl := ml.(*mapList[Data])
		if len(mlImpl.dataMap) != 2 {
			t.Errorf("UnmarshalJSON() dataMap length = %v, want 2", len(mlImpl.dataMap))
		}
	})

	t.Run("unmarshal preserves sorted index order", func(t *testing.T) {
		ml := List[string]()
		err := ml.UnmarshalJSON([]byte(`{"10":"ten","0":"zero","5":"five"}`))
		if err != nil {
			t.Errorf("UnmarshalJSON() unexpected error = %v", err)
			return
		}
		mlImpl := ml.(*mapList[string])
		var keys []int
		for _, k := range mlImpl.indexList.Range() {
			keys = append(keys, k)
		}
		wantKeys := []int{0, 5, 10}
		if len(keys) != len(wantKeys) {
			t.Errorf("UnmarshalJSON() keys length = %v, want %v", len(keys), len(wantKeys))
		}
		for i, k := range wantKeys {
			if keys[i] != k {
				t.Errorf("UnmarshalJSON() keys[%v] = %v, want %v", i, keys[i], k)
			}
		}
	})
}

func Test_mapList_UnmarshalYAML(t *testing.T) {
	type testCase struct {
		name    string
		yamlStr string
		want    map[int]string
		wantErr bool
	}
	tests := []testCase{
		{
			name:    "unmarshal empty YAML",
			yamlStr: "{}",
			want:    map[int]string{},
			wantErr: false,
		},
		{
			name:    "unmarshal single element",
			yamlStr: "0: zero",
			want:    map[int]string{0: "zero"},
			wantErr: false,
		},
		{
			name:    "unmarshal multiple elements",
			yamlStr: "0: zero\n1: one\n2: two",
			want:    map[int]string{0: "zero", 1: "one", 2: "two"},
			wantErr: false,
		},
		{
			name:    "unmarshal with non-sequential indexes",
			yamlStr: "0: zero\n5: five\n10: ten",
			want:    map[int]string{0: "zero", 5: "five", 10: "ten"},
			wantErr: false,
		},
		{
			name:    "unmarshal with out-of-order indexes",
			yamlStr: "5: five\n2: two\n8: eight\n1: one\n3: three",
			want:    map[int]string{1: "one", 2: "two", 3: "three", 5: "five", 8: "eight"},
			wantErr: false,
		},
		{
			name:    "unmarshal with reverse order indexes",
			yamlStr: "9: nine\n8: eight\n7: seven\n6: six\n5: five",
			want:    map[int]string{5: "five", 6: "six", 7: "seven", 8: "eight", 9: "nine"},
			wantErr: false,
		},
		{
			name:    "unmarshal with random order indexes",
			yamlStr: "42: forty-two\n7: seven\n99: ninety-nine\n1: one\n15: fifteen",
			want:    map[int]string{1: "one", 7: "seven", 15: "fifteen", 42: "forty-two", 99: "ninety-nine"},
			wantErr: false,
		},
		{
			name:    "unmarshal with empty string values",
			yamlStr: "0: ''\n1: ''",
			want:    map[int]string{0: "", 1: ""},
			wantErr: false,
		},
		{
			name:    "unmarshal with special characters",
			yamlStr: `0: 'hello "world"'` + "\n1: |-\n  line1\n  line2",
			want:    map[int]string{0: `hello "world"`, 1: "line1\nline2"},
			wantErr: false,
		},
		{
			name:    "reject negative index",
			yamlStr: "-1: negative",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "reject multiple negative indexes",
			yamlStr: "-1: neg1\n-2: neg2",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "reject mixed valid and negative indexes",
			yamlStr: "0: zero\n-1: negative\n1: one",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "unmarshal invalid YAML",
			yamlStr: "just a scalar value",
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ml := List[string]()
			var node yaml.Node
			err := yaml.Unmarshal([]byte(tt.yamlStr), &node)
			if err != nil {
				t.Fatalf("failed to parse YAML test data: %v", err)
			}
			err = ml.UnmarshalYAML(&node)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			mlImpl := ml.(*mapList[string])
			if len(mlImpl.dataMap) != len(tt.want) {
				t.Errorf("UnmarshalYAML() dataMap length = %v, want %v", len(mlImpl.dataMap), len(tt.want))
			}
			for k, v := range tt.want {
				gotV, ok := mlImpl.dataMap[k]
				if !ok {
					t.Errorf("UnmarshalYAML() missing key %v in dataMap", k)
				}
				if gotV != v {
					t.Errorf("UnmarshalYAML() dataMap[%v] = %v, want %v", k, gotV, v)
				}
			}
		})
	}

	t.Run("unmarshal with complex value type", func(t *testing.T) {
		type Data struct {
			Name  string `yaml:"name"`
			Value int    `yaml:"value"`
		}
		ml := List[Data]()
		var node yaml.Node
		err := yaml.Unmarshal([]byte("0:\n  name: first\n  value: 100\n1:\n  name: second\n  value: 200"), &node)
		if err != nil {
			t.Fatalf("failed to parse YAML test data: %v", err)
		}
		err = ml.UnmarshalYAML(&node)
		if err != nil {
			t.Errorf("UnmarshalYAML() unexpected error = %v", err)
			return
		}
		mlImpl := ml.(*mapList[Data])
		if len(mlImpl.dataMap) != 2 {
			t.Errorf("UnmarshalYAML() dataMap length = %v, want 2", len(mlImpl.dataMap))
		}
	})

	t.Run("unmarshal preserves sorted index order", func(t *testing.T) {
		ml := List[string]()
		var node yaml.Node
		err := yaml.Unmarshal([]byte("10: ten\n0: zero\n5: five"), &node)
		if err != nil {
			t.Fatalf("failed to parse YAML test data: %v", err)
		}
		err = ml.UnmarshalYAML(&node)
		if err != nil {
			t.Errorf("UnmarshalYAML() unexpected error = %v", err)
			return
		}
		mlImpl := ml.(*mapList[string])
		var keys []int
		for _, k := range mlImpl.indexList.Range() {
			keys = append(keys, k)
		}
		wantKeys := []int{0, 5, 10}
		if len(keys) != len(wantKeys) {
			t.Errorf("UnmarshalYAML() keys length = %v, want %v", len(keys), len(wantKeys))
		}
		for i, k := range wantKeys {
			if keys[i] != k {
				t.Errorf("UnmarshalYAML() keys[%v] = %v, want %v", i, keys[i], k)
			}
		}
	})

	t.Run("unmarshal with nil node", func(t *testing.T) {
		ml := List[string]()
		err := ml.UnmarshalYAML(nil)
		if err == nil {
			t.Errorf("UnmarshalYAML() expected error for nil node, got nil")
		}
	})
}

func Test_mapList_Remove(t *testing.T) {
	type args struct {
		index int
	}
	type testCase[V any] struct {
		name         string
		m            *mapList[V]
		args         args
		wantLenAfter int
	}
	tests := []testCase[string]{
		{
			name:         "remove from empty list",
			m:            List[string]().(*mapList[string]),
			args:         args{index: 0},
			wantLenAfter: 0,
		},
		{
			name:         "remove negative index - no-op",
			m:            func() *mapList[string] { ml := List[string]().(*mapList[string]); ml.Append("a"); return ml }(),
			args:         args{index: -1},
			wantLenAfter: 1,
		},
		{
			name: "remove existing index",
			m: func() *mapList[string] {
				ml := List[string]().(*mapList[string])
				ml.Append("a")
				ml.Append("b")
				ml.Append("c")
				return ml
			}(),
			args:         args{index: 1},
			wantLenAfter: 2,
		},
		{
			name: "remove non-existing index",
			m: func() *mapList[string] {
				ml := List[string]().(*mapList[string])
				ml.Append("a")
				ml.Append("b")
				return ml
			}(),
			args:         args{index: 10},
			wantLenAfter: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.m.Remove(tt.args.index)
			if got := tt.m.Len(); got != tt.wantLenAfter {
				t.Errorf("Len after Remove() = %v, want %v", got, tt.wantLenAfter)
			}
		})
	}
}

func Test_mapList_Contains(t *testing.T) {
	type args struct {
		index int
	}
	type testCase struct {
		name        string
		m           MapList[string]
		args        args
		wantPresent bool
	}
	tests := []testCase{
		{
			name:        "empty list - not present",
			m:           List[string](),
			args:        args{index: 0},
			wantPresent: false,
		},
		{
			name:        "negative index - not present",
			m:           func() MapList[string] { ml := List[string](); ml.Append("a"); return ml }(),
			args:        args{index: -1},
			wantPresent: false,
		},
		{
			name:        "index out of bounds - not present",
			m:           func() MapList[string] { ml := List[string](); ml.Append("a"); ml.Append("b"); return ml }(),
			args:        args{index: 10},
			wantPresent: false,
		},
		{
			name: "valid index - present",
			m: func() MapList[string] {
				ml := List[string]()
				ml.Append("a")
				ml.Append("b")
				ml.Append("c")
				return ml
			}(),
			args:        args{index: 1},
			wantPresent: true,
		},
		{
			name:        "first index - present",
			m:           func() MapList[string] { ml := List[string](); ml.Append("a"); ml.Append("b"); return ml }(),
			args:        args{index: 0},
			wantPresent: true,
		},
		{
			name: "last index - present",
			m: func() MapList[string] {
				ml := List[string]()
				ml.Append("a")
				ml.Append("b")
				ml.Append("c")
				return ml
			}(),
			args:        args{index: 2},
			wantPresent: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotPresent := tt.m.Contains(tt.args.index); gotPresent != tt.wantPresent {
				t.Errorf("Contains() = %v, want %v", gotPresent, tt.wantPresent)
			}
		})
	}
}
