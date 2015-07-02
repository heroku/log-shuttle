package util

import (
	"bytes"
	"fmt"
	"log"
	"runtime"
)

func makeErr(err error, msg ...interface{}) string {
	var b bytes.Buffer
	_, filename, lineno, ok := runtime.Caller(2)
	if ok {
		b.WriteString(fmt.Sprintf("%v:%v: %v", filename, lineno, err))
	} else {
		b.WriteString(err.Error())
	}
	if len(msg) > 0 {
		b.WriteString(",")
		for _, m := range msg {
			b.WriteString(fmt.Sprintf(" %v", m))
		}
	}
	return b.String()
}

func CheckErr(err error, msg ...interface{}) {
	if err != nil {
		log.SetFlags(0)
		log.Fatalln(makeErr(err, msg...))
	}
}

func WarnErr(err error, msg ...interface{}) error {
	if err != nil {
		f := log.Flags()
		log.SetFlags(0)
		log.Println(makeErr(err, msg...))
		log.SetFlags(f)
	}
	return err
}
