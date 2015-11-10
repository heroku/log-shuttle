package shuttle

import (
	"encoding/base64"
	"io"
	"strconv"
)

var shard int = 1

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
	var b string

	// Add an integer in the PartitionKey to enable distribution
	// over multiple shards in the Kinesis stream.
	b = `{"PartitionKey":"` + getPartitionKey(r) + `","Data":"`
	t, err = w.Write([]byte(b))
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

func getShard(r KinesisRecord) (nextShard string) {
	// For consistency with older versions, if there is only one shard,
	// we will just use the app name, and skip any unnecessary work.
	totalShards := r.llf.shards
	if totalShards == 1 {
		return ""
	}
	if shard > totalShards {
		shard = totalShards
	}
	nextShard = strconv.Itoa(shard)
	shard++
	return
}

func getPartitionKey(r KinesisRecord) string {
	return r.llf.AppName() + getShard(r)
}
