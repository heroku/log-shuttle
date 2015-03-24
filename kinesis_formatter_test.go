package shuttle

import (
	"compress/gzip"
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

func TestKinesisFormatterRequest(t *testing.T) {
	config.LogsURL = "https://key:secret@kinesis.us-east-1.amazonaws.com/Stream"
	b := NewBatch(1)
	b.Add(LogLineOne)
	b.Add(LogLineTwo)
	kf := NewKinesisFormatter(b, noErrData, &config)
	r, err := kf.Request()
	if err != nil {
		t.Fatal("Unexpected error calling Request: ", err)
	}

	// Read the body of the request
	d, err := ioutil.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("Error reading everything from the request: %q", err)
	}

	t.Logf("%q", string(d))
}

func TestKinesisGzip(t *testing.T) {
	config.LogsURL = "https://key:secret@foo/Stream"
	b := NewBatch(1)
	b.Add(LogLineOne)
	kf := NewKinesisFormatter(b, noErrData, &config)

	gf := NewGzipFormatter(kf)

	// decompress the bytes and verify the message
	gunzipper, err := gzip.NewReader(gf)
	if err != nil {
		t.Fatal("Error making a reader: ", err)
	}

	// read the uncompressed bytes
	uncompressed, err := ioutil.ReadAll(gunzipper)
	if err != nil {
		t.Fatal("Errors reading the compressed bytes: ", err)
	}

	t.Log("Data: ", string(uncompressed))
}
