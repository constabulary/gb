package vendor

import "testing"
import "reflect"

func set(args ...string) map[string]bool {
	r := make(map[string]bool)
	for _, a := range args {
		r[a] = true
	}
	return r
}

func TestUnion(t *testing.T) {
	tests := []struct {
		a, b map[string]bool
		want map[string]bool
	}{{
		a: nil, b: nil,
		want: set(),
	}, {
		a: nil, b: set("b"),
		want: set("b"),
	}, {
		a: set("a"), b: nil,
		want: set("a"),
	}, {
		a: set("a"), b: set("b"),
		want: set("b", "a"),
	}, {
		a: set("c"), b: set("c"),
		want: set("c"),
	}}

	for _, tt := range tests {
		got := union(tt.a, tt.b)
		if !reflect.DeepEqual(tt.want, got) {
			t.Errorf("union(%v, %v) want: %v, got %v", tt.a, tt.b, tt.want, got)
		}
	}
}

func TestIntersection(t *testing.T) {
	tests := []struct {
		a, b map[string]bool
		want map[string]bool
	}{{
		a: nil, b: nil,
		want: set(),
	}, {
		a: nil, b: set("b"),
		want: set(),
	}, {
		a: set("a"), b: nil,
		want: set(),
	}, {
		a: set("a"), b: set("b"),
		want: set(),
	}, {
		a: set("c"), b: set("c"),
		want: set("c"),
	}, {
		a: set("a", "c"), b: set("b", "c"),
		want: set("c"),
	}}

	for _, tt := range tests {
		got := intersection(tt.a, tt.b)
		if !reflect.DeepEqual(tt.want, got) {
			t.Errorf("intersection(%v, %v) want: %v, got %v", tt.a, tt.b, tt.want, got)
		}
	}
}

func TestDifference(t *testing.T) {
	tests := []struct {
		a, b map[string]bool
		want map[string]bool
	}{{
		a: nil, b: nil,
		want: set(),
	}, {
		a: nil, b: set("b"),
		want: set("b"),
	}, {
		a: set("a"), b: nil,
		want: set("a"),
	}, {
		a: set("a"), b: set("b"),
		want: set("a", "b"),
	}, {
		a: set("c"), b: set("c"),
		want: set(),
	}, {
		a: set("a", "c"), b: set("b", "c"),
		want: set("a", "b"),
	}}

	for _, tt := range tests {
		got := difference(tt.a, tt.b)
		if !reflect.DeepEqual(tt.want, got) {
			t.Errorf("difference(%v, %v) want: %v, got %v", tt.a, tt.b, tt.want, got)
		}
	}
}
