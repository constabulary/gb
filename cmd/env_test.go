package cmd

import "testing"

func TestMergeEnv(t *testing.T) {
	envTests := []struct {
		env  []string
		args map[string]string
		want []string
	}{
		{
			env:  nil,
			args: nil,
			want: nil,
		},
		{
			env:  []string{`FOO=BAR`, `BAZ="QUXX"`},
			args: nil,
			want: []string{`FOO=BAR`, `BAZ="QUXX"`},
		},
		{
			env:  []string{`FOO=BAR`, `BAZ="QUXX"`},
			args: map[string]string{"BLORT": "false", "BAZ": "QUXX"},
			want: []string{`FOO=BAR`, `BAZ=QUXX`, `BLORT=false`},
		},
	}

	for _, tt := range envTests {
		got := MergeEnv(tt.env, tt.args)
		compare(t, tt.want, got)
	}
}

func compare(t *testing.T, want, got []string) {
	w, g := set(want), set(got)
	for k := range w {
		if w[k] != g[k] {
			t.Errorf("want %v, got %v", k, g[k])
		}
	}
}

func set(v []string) map[string]bool {
	m := make(map[string]bool)
	for _, s := range v {
		m[s] = true
	}
	return m
}
