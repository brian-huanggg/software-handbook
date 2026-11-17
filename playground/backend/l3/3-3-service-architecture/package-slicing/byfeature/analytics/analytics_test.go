package analytics

import (
	"reflect"
	"testing"
)

func TestTop(t *testing.T) {
	r := New()
	for _, c := range []string{"a", "b", "b", "c", "b", "a"} {
		r.Record(c)
	}
	want := []Count{{"b", 3}, {"a", 2}}
	if got := r.Top(2); !reflect.DeepEqual(got, want) {
		t.Fatalf("Top(2) = %v, want %v", got, want)
	}
}
