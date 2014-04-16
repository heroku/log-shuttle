package main

import "testing"

func TestProgramStatsSnapshot(t *testing.T) {
	ps := NewProgramStats("tcp,:9000", 0)

	ps.Input <- NewNamedValue("test", 1.0)
	snapshot := ps.Snapshot(false)

	//Test some of the values, but not all
	v, ok := snapshot["alltime.drops.count"]
	if !ok {
		t.Fatal("Unable to find log-shuttle.alltime.drops.count")
	}
	if v != 0 {
		t.Errorf("alltime.drops.count expected to be 0, got: %d\n", v)
	}

	v, ok = snapshot["test.p50"]
	if !ok {
		t.Fatal("Unable to find log-shuttle.test.p50 in snapshot")
	}

	if v != 1.0 {
		t.Errorf("Value of count (%d) is incorrect, expecting 1.0\n", v)
	}
	close(ps.Input)
}
