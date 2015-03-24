package shuttle

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"testing"
)

func TestKinesisRecord_MarshalJSONToWriter(t *testing.T) {
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
