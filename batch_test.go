package main

import (
	"regexp"
	"testing"
	"time"
)

func TestBatchMsgAgeRange(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	batch := NewBatch(&config)
	if batch.UUID == nil {
		t.Errorf("Batch's UUID is nil, expected non nil")
	}
	if batch.UUID.String() == "" {
		t.Errorf("Batch's UUID is empty, expect non empty")
	}
}

func TestBatchReset(t *testing.T) {
	t.Parallel()
	batch := NewBatch(&config)

	if batch.MsgCount != 0 {
		t.Errorf("batch.MsgCount != 0, == %q", batch.MsgCount)
	}

	if batchDataLength := len(batch.String()); batchDataLength != 0 {
		t.Errorf("Length of Batch Data isn't 0, is %q", batchDataLength)
	}

	batch.Write(LogLine{[]byte("Hello"), time.Now()})

	if batch.MsgCount != 1 {
		t.Errorf("batch.MsgCount != 1, == %q", batch.MsgCount)
	}

	batch.Reset()

	if batch.MsgCount != 0 {
		t.Errorf("batch.MsgCount != 0, == %q", batch.MsgCount)
	}

	if batchDataLength := len(batch.String()); batchDataLength != 0 {
		t.Errorf("Length of Batch Data isn't 0, is %q", batchDataLength)
	}
}

func TestBatchDrop(t *testing.T) {
	t.Parallel()

	const drops = 5

	batch := NewBatch(&config)
	if batch.Drops != 0 {
		t.Errorf("batch.Drops != 0, == %q", batch.Drops)
	}

	batch.WriteDrops(drops, time.Now())
	if batch.Drops != drops {
		t.Errorf("batch.Drops != %q, == %q", drops, batch.Drops)
	}
	if batch.MsgCount != 1 {
		t.Errorf("batch.MsgCount != 1, == %q", batch.MsgCount)
	}

	dropMsg := batch.Bytes()
	dropMsgCheck := regexp.MustCompile(`138 <172>1 [0-9T:\+\-\.]+ heroku token log-shuttle - - Error L12: 5 messages dropped since [0-9T:\+\-\.]+\.`)
	if !dropMsgCheck.Match(dropMsg) {
		t.Errorf("drop message isn't correct: %q", dropMsg)
	}

	batch.Reset()
	if batch.MsgCount != 0 {
		t.Errorf("batch.MsgCount != 0, == %q", batch.MsgCount)
	}
	if batch.Drops != 0 {
		t.Errorf("batch.Drops != 0, == %q", batch.Drops)
	}
	if batchDataLength := len(batch.String()); batchDataLength != 0 {
		t.Errorf("Length of Batch Data isn't 0, is %q", batchDataLength)
	}

}

func TestBatchLost(t *testing.T) {
	t.Parallel()

	const lost = 5

	batch := NewBatch(&config)
	if batch.Lost != 0 {
		t.Errorf("batch.Lost != 0, == %q", batch.Lost)
	}
	batch.WriteLost(lost, time.Now())
	if batch.Lost != lost {
		t.Errorf("batch.Lost != %q, == %q", lost, batch.Lost)
	}
	if batch.MsgCount != 1 {
		t.Errorf("batch.MsgCount != 1, == %q", batch.MsgCount)
	}

	lostMsg := batch.Bytes()
	lostMsgCheck := regexp.MustCompile(`135 <172>1 [0-9T:\+\-\.]+ heroku token log-shuttle - - Error L13: 5 messages lost since [0-9T:\+\-\.]+\.`)
	if !lostMsgCheck.Match(lostMsg) {
		t.Errorf("Lost message isn't correct: %q", lostMsg)
	}

	batch.Reset()
	if batch.MsgCount != 0 {
		t.Errorf("batch.MsgCount != 0, == %q", batch.MsgCount)
	}
	if batch.Lost != 0 {
		t.Errorf("batch.Lost != 0, == %q", batch.Lost)
	}
	if batchDataLength := len(batch.String()); batchDataLength != 0 {
		t.Errorf("Length of Batch Data isn't 0, is %q", batchDataLength)
	}
}
