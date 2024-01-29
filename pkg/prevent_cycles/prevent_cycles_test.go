package prevent_cycles

import (
	"testing"
)

func Test_preventer_CheckLoop(t *testing.T) {
	p := NewStringPreventer()
	err := p.CheckLoop("aaa", "bbb")
	if err != nil {
		t.Fatal(err)
	}
	err = p.CheckLoop("bbb", "ccc")
	if err != nil {
		t.Fatal(err)
	}

	type args[T any] struct {
		src T
		dst T
	}
	type testCase[K comparable, T any] struct {
		name    string
		s       *preventer[K, T]
		args    args[T]
		wantErr bool
	}
	tests := []testCase[string, string]{
		{
			name: "test1",
			s:    p.(*preventer[string, string]),
			args: args[string]{"aaa", "ccc"},
		},
		{
			name:    "test2",
			s:       p.(*preventer[string, string]),
			args:    args[string]{"ccc", "aaa"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.s.CheckLoop(tt.args.src, tt.args.dst); (err != nil) != tt.wantErr {
				t.Errorf("CheckLoop() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
