package shuttle

import (
	"testing"
)

func TestCounter(t *testing.T) {
	counter := Counter{}
	if c := counter.Read(); c != 0 {
		t.Fatalf("counter should be 0, but was %d", c)
	}
	counter.Add(1)
	if c := counter.Read(); c != 1 {
		t.Fatalf("counter should be 1, but was %d", c)
	}
	counter.Add(2)
	if c, _ := counter.ReadAndReset(); c != 3 {
		t.Fatalf("counter should have been 3, but was %d", c)
	}
	if c := counter.Read(); c != 0 {
		t.Fatalf("counter should be have been 0 after read/reset, but was %d", c)
	}
}
