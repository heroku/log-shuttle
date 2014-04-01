package main

import (
	"testing"
)

func TestProgramStats_Snapshot(t *testing.T) {
	statsChannel := make(chan NamedValue)
	lost := new(Counter)
	drops := new(Counter)
	ps := NewProgramStats("tcp,:9000", lost, drops, statsChannel)

	statsChannel <- NewNamedValue("test", 1.0)
	snapshot := ps.Snapshot(false)

	//Test some of the values, but not all
	v, ok := snapshot["log-shuttle.alltime.drops.count"]
	if !ok {
		t.Error("Unable to find log-shuttle.alltime.drops.count")
	}
	if v != 0 {
		t.Errorf("alltime.drops.count expected to be 0, got: %d\n", v)
	}

	v, ok = snapshot["log-shuttle.test.p50.seconds"]
	if !ok {
		t.Error("Unable to find log-shuttle.test.p50.seconds in snapshot")
	}

	if v.(float64) != 1.0 {
		t.Errorf("Value of count (%d) is incorrect, expecting 1.0\n", v)
	}
	close(statsChannel)
}
