package shuttle

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"testing"
)

func TestKinesisRecord_MarshalJSONToWriter(t *testing.T) {
	config := newTestConfig()
	llf := NewLogplexLineFormatter(LogLineOne, &config)
	b := new(bytes.Buffer)
	r := KinesisRecord{llf}

	_, err := r.WriteTo(b)
	if err != nil {
		t.Fatal("Unexpected error calling MarshalJSONToWriter: ", err)
	}

	tr := struct {
		Data         []byte
		PartitionKey string
	}{}

	t.Logf("%+q\n", b.Bytes())

	if err := json.Unmarshal(b.Bytes(), &tr); err != nil {
		t.Fatal("Unexpected error unmashalling KinesisRecord: ", err)
	}

	t.Logf("%+q\n", tr)

	if tr.PartitionKey == "" {
		t.Fatal("Expected PartitonKey to not be empty, but was.")
	}

	llf.Reset()

	d, _ := ioutil.ReadAll(llf)

	if string(tr.Data) != string(d) {
		t.Logf("tr.Data: %q\n, d: %q\n", string(tr.Data), string(d))
		t.Fatal("Expected Data to be equal to reading all the formatted bytes of the log line, but wasn't.")
	}
}

func TestKinesisRecordSharding(t *testing.T) {
	config := newTestConfig()
	config.KinesisShards = 2
	llf := NewLogplexLineFormatter(LogLineOne, &config)
	llf2 := NewLogplexLineFormatter(LogLineTwo, &config)
	b := new(bytes.Buffer)
	r := KinesisRecord{llf}
	r2 := KinesisRecord{llf2}

	_, err := r.WriteTo(b)
	if err != nil {
		t.Fatal("Unexpected error calling MarshalJSONToWriter: ", err)
	}

	tr := struct {
		Data         []byte
		PartitionKey string
	}{}

	t.Logf("%+q\n", b.Bytes())

	if err := json.Unmarshal(b.Bytes(), &tr); err != nil {
		t.Fatal("Unexpected error unmashalling KinesisRecord: ", err)
	}

	t.Logf("%+q\n", tr)

	if tr.PartitionKey != "shuttle1" {
		t.Fatal("Expected PartitonKey to not be empty, but was.")
	}

	b = new(bytes.Buffer)
	_, err = r2.WriteTo(b)
	if err != nil {
		t.Fatal("Unexpected error calling MarshalJSONToWriter: ", err)
	}

	tr = struct {
		Data         []byte
		PartitionKey string
	}{}

	t.Logf("%+q\n", b.Bytes())

	if err := json.Unmarshal(b.Bytes(), &tr); err != nil {
		t.Fatal("Unexpected error unmashalling KinesisRecord: ", err)
	}

	t.Logf("%+q\n", tr)

	if tr.PartitionKey != "shuttle2" {
		t.Fatal("Expected PartitonKey to not be empty, but was.")
	}
}
