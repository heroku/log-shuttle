package main

import (
	"fmt"
	"io"
	"time"
)

var layout = "<%s>%s %s %s %s %s %s %s"

func SyslogFmt(w io.Writer, logs []string, conf *ShuttleConfig) {
	for i := range logs {
		var packet string
		if conf.SkipHeaders {
			packet = logs[i]
		} else {
			packet = fmt.Sprintf(layout,
				conf.Prival,
				conf.Version,
				time.Now().UTC().Format("2006-01-02T15:04:05.000000+00:00"),
				conf.Hostname,
				conf.Appname,
				conf.Procid,
				conf.Msgid,
				logs[i])
		}
		fmt.Fprintf(w, "%d %s", len(packet), packet)
	}
}
