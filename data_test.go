package main

import (
	"regexp"
	"time"
)

var (
	NewLine                          = byte('\n')
	LogLineOne                       = LogLine{line: []byte("Hello World\n"), when: time.Now()}
	logplexTestLineOnePattern        = regexp.MustCompile(`78 <190>1 [0-9T:\+\-\.]+ shuttle token shuttle - - Hello World\n`)
	LogLineTwo                       = LogLine{line: []byte("The Second Test Line \n"), when: time.Now()}
	logplexTestLineTwoPattern        = regexp.MustCompile(`88 <190>1 [0-9T:\+\-\.]+ shuttle token shuttle - - The Second Test Line \n`)
	LongLogLine                      = LogLine{when: time.Now()}
	LogLineOneWithHeaders            = LogLine{line: []byte("<13>1 2013-09-25T01:16:49.371356+00:00 host token web.1 - [meta sequenceId=\"1\"] message 1\n"), when: time.Now()}
	LogLineTwoWithHeaders            = LogLine{line: []byte("<13>1 2013-09-25T01:16:49.402923+00:00 host token web.1 - [meta sequenceId=\"2\"] other message\n"), when: time.Now()}
	logplexLineOneWithHeadersPattern = regexp.MustCompile(`90 <13>1 2013-09-25T01:16:49\.371356\+00:00 host token web\.1 - \[meta sequenceId="1"\] message 1\n`)
	logplexLineTwoWithHeadersPattern = regexp.MustCompile(`94 <13>1 2013-09-25T01:16:49\.402923\+00:00 host token web\.1 - \[meta sequenceId="2"\] other message\n`)
	noErrData                        = make([]errData, 0)
)

var (
	primaryVersionOne = "<190>1"
  primaryVersionOneFacility = "local7"
  primaryVersionOneLevel = "info"
  primaryVersionOneVersion = 1
	primaryVersionTwo = "<0>1"
  primaryVersionTwoFacility = "kern"
  primaryVersionTwoLevel = "emerg"
  primaryVersionTwoVersion = 1
)

func init() {
	for i := 0; i < 2980; i++ {
		LongLogLine.line = append(LongLogLine.line, []byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9'}...)
	}
	LongLogLine.line = append(LongLogLine.line, NewLine)
}

