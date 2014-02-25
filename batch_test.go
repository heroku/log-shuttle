package main

import (
	"testing"
	"time"
)

func TestBatchMsgAgeRange(t *testing.T) {
	batch := NewBatch(&config)
	logline1 := LogLine{line: []byte("Hi there"), when: time.Now()}
	batch.Write(logline1)
	logline2 := LogLine{line: []byte("Hi there"), when: time.Now()}
	batch.Write(logline2)
	if mar := batch.MsgAgeRange(); mar < 0 {
		t.Errorf("MsgAgeRange() is < 0, expected > 0 : %v", mar)
	}
}

func TestBatchUUID(t *testing.T) {
	batch := NewBatch(&config)
	if batch.UUID == nil {
		t.Errorf("Batch's UUID is nil, expected non nil")
	}
	if batch.UUID.String() == "" {
		t.Errorf("Batch's UUID is empty, expect non empty")
	}
}
