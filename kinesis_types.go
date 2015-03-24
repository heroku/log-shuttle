package shuttle

import (
	"encoding/base64"
	"io"
	"io/ioutil"
)

// KinesisRecord is used to marshal LoglexLineFormatters to Kinesis Records for
// the PutRecords API Call
type KinesisRecord struct {
	llf *LogplexLineFormatter
}

// WriteTo writes the LogplexLineFormatter to the provided writer
// in Kinesis' PutRecordsFormat. Conforms to the WriterTo interface.
//
// Since the data should be relatively small just read the LogplexLineFormatter
// into ram making a bunch of temporary garbage.  AFAICT it's just not worth it
// to try and string these together with io.Pipes and the like. :-(
func (r KinesisRecord) WriteTo(w io.Writer) (n int64, err error) {
	var t int

	b, err := ioutil.ReadAll(r.llf)
	if err != nil {
		return
	}

	t, err = w.Write([]byte(`{"PartitionKey":"` + r.llf.AppName() + `","Data":"`))
	n += int64(t)
	if err != nil {
		return
	}

	e := base64.NewEncoder(base64.StdEncoding, w)

	t, err = e.Write(b)
	n += int64(t)
	if err != nil {
		return
	}
	e.Close()

	t, err = w.Write([]byte(`"}`))
	n += int64(t)
	return
}
