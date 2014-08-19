package shuttle

import (
	"encoding/json"
	"time"
)

type LogLine struct {
	line []byte
	when time.Time
}

func (ll LogLine) Length() int {
	return len(ll.line)
}

type LogLineEncoder LogLine

type JSONLogLine struct {
	Msg     string    `json:"msg"`
	When    time.Time `json:"when"`
	ProcId  string    `json:"proc_id"`
	AppName string    `json:"app_name"`
}

func (lle LogLineEncoder) Encode() (msg []byte, err error) {
	return json.Marshal(
		JSONLogLine{
			Msg:  string(lle.line),
			When: lle.when,
		},
	)
}
