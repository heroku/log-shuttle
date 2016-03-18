package shuttle

import (
	"encoding/base64"
	"io"
	"strconv"
)

const (
	partitionKeyHeader = `{"PartitionKey":"`
	partitionKeyFooter = `","Data":"`
)

// KinesisRecord is used to marshal LoglexLineFormatters to Kinesis Records for
// the PutRecords API Call
type KinesisRecord struct {
	llf   *LogplexLineFormatter
	shard int
}

// WriteTo writes the LogplexLineFormatter to the provided writer
// in Kinesis' PutRecordsFormat. Conforms to the WriterTo interface.
func (r KinesisRecord) WriteTo(w io.Writer) (n int64, err error) {
	var t int
	var t64 int64
	var b string

	// Add an integer in the PartitionKey to enable distribution
	// over multiple shards in the Kinesis stream.
	b = partitionKeyHeader + r.partitionKey() + partitionKeyFooter
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

// There is no guarantee that this partitionKey will hash to a different shard.
// Consider this a simplistic form of "sharding" for now.
func (r KinesisRecord) partitionKey() string {
	if r.shard == 0 {
		return r.llf.AppName()
	}
	return r.llf.AppName() + strconv.Itoa(r.shard)
}
