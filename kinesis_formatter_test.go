package shuttle

import (
	"io/ioutil"
	"testing"
)

func TestKinesisFormatter(t *testing.T) {
	config.LogsURL = "https://key:secret@foo/Stream"
	b := NewBatch(1)
	b.Add(LogLineOne)
	b.Add(LogLineTwo)
	br := NewKinesisFormatter(b, noErrData, &config)
	d, err := ioutil.ReadAll(br)
	if err != nil {
		t.Fatalf("Error reading everything from batch: %q", err)
	}

	t.Logf("%q", string(d))
}
