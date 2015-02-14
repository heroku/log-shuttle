package shuttle

import (
	"encoding/base64"
	"io"
	"io/ioutil"
)

// Record is used to marshal LoglexLineFormatters to Kinesis Records for the
// PutRecords API Call
type KinesisRecord struct {
	llf *LogplexLineFormatter
}

// MarshalJSONToWriter marshals the LogplexLineFormatter to the provided writer
// in Kinesis' PutRecordsFormat
//
// Since the data should be relatively small just read the LogplexLineFormatter
// into ram making a bunch of temporary garbage.  AFAICT it's just not worth it
// to try and string these together with io.Pipes and the like. :-(
func (r KinesisRecord) MarshalJSONToWriter(w io.Writer) error {
	t, err := ioutil.ReadAll(r.llf)
	if err != nil {
		return err
	}

	if _, err = w.Write([]byte(`{"PartitionKey":"` + r.llf.AppName() + `","Data":"`)); err != nil {
		return err
	}

	e := base64.NewEncoder(base64.StdEncoding, w)

	if _, err = e.Write(t); err != nil {
		return err
	}
	e.Close()

	_, err = w.Write([]byte(`"}`))
	return err
}
