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

func TestKinesisGzip(t *testing.T) {
	config.LogsURL = "https://key:secret@foo/Stream"
	b := NewBatch(1)
	b.Add(LogLineOne)
	kf := NewKinesisFormatter(b, noErrData, &config)

	// Expecting a panic here?
	gf := NewGzipFormatter(kf)

	d, err := ioutil.ReadAll(gf)
	if err != nil {
		t.Fatal("Didn't expect an error reading all data from the gzip formatter, but got: ", err)
	}

	t.Log("Data: ", d)
}
