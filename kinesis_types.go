package shuttle

import (
	"encoding/base64"
	"io"
)

// KinesisRecord is used to marshal LoglexLineFormatters to Kinesis Records for
// the PutRecords API Call
type KinesisRecord struct {
	llf *LogplexLineFormatter
}

// WriteTo writes the LogplexLineFormatter to the provided writer
// in Kinesis' PutRecordsFormat. Conforms to the WriterTo interface.
func (r KinesisRecord) WriteTo(w io.Writer) (n int64, err error) {
	var t int
	var t64 int64

	t, err = w.Write([]byte(`{"PartitionKey":"` + r.llf.AppName() + `","Data":"`))
	n += int64(t)
	if err != nil {
		return
	}

	e := base64.NewEncoder(base64.StdEncoding, w)

	t64, err = io.Copy(e, r.llf)
	n += encodedLen(t64)
	if err != nil {
		return
	}

	if err = e.Close(); err != nil {
		return
	}

	t, err = w.Write([]byte(`"}`))
	n += int64(t)
	return
}

// The same as Encoding.EncodedLen, but for int64
func encodedLen(n int64) int64 {
	return (n + 2) / 3 * 4
}
