package kinesis

import "encoding/json"

type Record struct {
	Data            json.Marshaler
	ExplicitHashKey string
	PartitionKey    string
}
