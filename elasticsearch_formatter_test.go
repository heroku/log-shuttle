package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/nu7hatch/gouuid"
)

func TestNewElasticSearchDocument_PrivalParsing(t *testing.T) {
	facilityOne, levelOne, versionOne := parsePrivalVersion(primaryVersionOne)
	if facilityOne != primaryVersionOneFacility {
		t.Fatalf("actual=%q\n", facilityOne)
	}
	if levelOne != primaryVersionOneLevel {
		t.Fatalf("actual=%q\n", levelOne)
	}
	if versionOne != primaryVersionOneVersion {
		t.Fatalf("actual=%q\n", versionOne)
	}

	facilityTwo, levelTwo, versionTwo := parsePrivalVersion(primaryVersionTwo)
	if facilityTwo != primaryVersionTwoFacility {
		t.Fatalf("actual=%q\n", facilityTwo)
	}
	if levelTwo != primaryVersionTwoLevel {
		t.Fatalf("actual=%q\n", levelTwo)
	}
	if versionTwo != primaryVersionTwoVersion {
		t.Fatalf("actual=%q\n", versionTwo)
	}
}

func TestNewElasticSearchDocument(t *testing.T) {
	config.SkipHeaders = true
	defer func() { config.SkipHeaders = false }()

	expected := &ElasticSearchDocument{
		Facility: "user",
		Level:    "notice",
		Version:  1,
		Time:     "2013-09-25T01:16:49.371356+00:00",
		Hostname: "host",
		Name:     "token",
		Procid:   "web.1",
		Msgid:    "-",
		Msg:      "[meta sequenceId=\"1\"] message 1\n",
	}

	d := NewElasticSearchDocument(LogLineOneWithHeaders, &config)

	if *expected != *d {
		t.Fatalf("actual=%q\n", d)
	}
}

func doTestElasticSearchIndexFormatter(t *testing.T, fmtr *ElasticSearchIndexFormatter, expAction ElasticSearchIndexAction, expDoc ElasticSearchDocument) {
	var actualAction ElasticSearchIndexAction
	var actualDocument ElasticSearchDocument

	out, err := ioutil.ReadAll(fmtr)
	if err != nil {
		t.Fatalf("error on read=%q\n", err)
	}

	// Attempt to Decode the output and compare against the original
	decoder := json.NewDecoder(bytes.NewBuffer(out))
	if err = decoder.Decode(&actualAction); err != nil {
		t.Fatalf("error=%q\n", err)
	}

	if actualAction != expAction {
		t.Errorf("actual=%q\nexpected=%q\n", actualAction, expAction)
	}

	if err = decoder.Decode(&actualDocument); err != nil {
		t.Fatalf("error=%q\n", err)
	}

	if actualDocument != expDoc {
		t.Errorf("actual=%q\nexpected=%q\n", actualDocument, expDoc)
	}
}

func TestElasticSearchIndexFormatter_WithHeaders(t *testing.T) {
	config.SkipHeaders = true
	defer func() { config.SkipHeaders = false }()

	llOne := LogLineOneWithHeaders
	index := 0 // index of thing in batch
	batchId, err := uuid.NewV4()
	if err != nil {
		t.Fatalf("error: %q\n", err)
	}

	action := ElasticSearchIndexAction{
		Index: ElasticSearchIndexActionBody{
			Id:        fmt.Sprintf("%s:%d", batchId.String(), index),
			Timestamp: llOne.when.UTC().Format(LOGPLEX_BATCH_TIME_FORMAT),
		},
	}

	document := ElasticSearchDocument{
		Facility: "user",
		Level:    "notice",
		Version:  1,
		Time:     "2013-09-25T01:16:49.371356+00:00",
		Hostname: "host",
		Name:     "token",
		Procid:   "web.1",
		Msgid:    "-",
		Msg:      "[meta sequenceId=\"1\"] message 1\n",
	}

	f, _ := NewElasticSearchIndexFormatter(llOne, &config, batchId, index)
	doTestElasticSearchIndexFormatter(t, f, action, document)
}

func TestElasticSearchIndexFormatter_Basic(t *testing.T) {
	llOne := LogLineOne
	index := 0 // index of thing in batch
	batchId, err := uuid.NewV4()
	if err != nil {
		t.Fatalf("error: %q\n", err)
	}

	action := ElasticSearchIndexAction{
		Index: ElasticSearchIndexActionBody{
			Id:        fmt.Sprintf("%s:%d", batchId.String(), index),
			Timestamp: llOne.when.UTC().Format(LOGPLEX_BATCH_TIME_FORMAT),
		},
	}

	// expected
	document := ElasticSearchDocument{
		Facility: "local7",
		Level:    "info",
		Version:  1,
		Time:     llOne.when.UTC().Format(LOGPLEX_BATCH_TIME_FORMAT),
		Hostname: "shuttle",
		Name:     "token",
		Procid:   "shuttle",
		Msgid:    "-",
		Msg:      "- Hello World\n",
	}

	f, _ := NewElasticSearchIndexFormatter(llOne, &config, batchId, index)

	doTestElasticSearchIndexFormatter(t, f, action, document)
}

func TestElasticSearchBatchFormatter(t *testing.T) {
	b := NewBatch(1)
	b.Add(LogLineOne)
	b.Add(LogLineTwo)
	br, _ := NewElasticSearchBatchFormatter(b, noErrData, &config)
	d, err := ioutil.ReadAll(br)
	if err != nil {
		t.Fatalf("Error reading everything from batch: %q", err)
	}

	bs := bytes.Split(d, []byte{'\n'})
	if len(bs) != 5 {
		t.Fatalf("Line count != 5, is: %d\n", len(bs))
	}

	for i, l := range bs {
		if i == 4 {
			if len(l) > 0 {
				t.Fatal("Payload should end with \\n, got: \"%s\"\n", string(l))
			}
		} else if len(l) < 2 { // should at least be {}, e.g. valid JSON
			t.Fatal("Expected len >= 2, is: %d\n", len(l))
		}
	}

	t.Logf("%q", string(d))
}

func TestElasticSearchBatchFormatter_MsgCount(t *testing.T) {
	b := NewBatch(1)
	b.Add(LogLineOne)  // 1 frame
	b.Add(LongLogLine) // 1 frame 

	br, _ := NewElasticSearchBatchFormatter(b, noErrData, &config)

	if msgCount := br.MsgCount(); msgCount != 2 {
		t.Fatalf("Formatter's MsgCount != 2, is: %d\n", msgCount)
	}
}

func BenchmarkElasticSearchIndexFormatter(b *testing.B) {
	bid, _ := uuid.NewV4()

	for i := 0; i < b.N; i++ {
		lf, _ := NewElasticSearchIndexFormatter(LogLineOne, &config, bid, 1)
		_, err := ioutil.ReadAll(lf)
		if err != nil {
			b.Fatalf("Error reading everything from line: %q", err)
		}
	}
}

func BenchmarkElasticSearchIndexFormatter_WithHeaders(b *testing.B) {
	config.SkipHeaders = true
	defer func () { config.SkipHeaders = false }()

	bid, _ := uuid.NewV4()

	for i := 0; i < b.N; i++ {
		lf, _ := NewElasticSearchIndexFormatter(LogLineOneWithHeaders, &config, bid, 1)
		_, err := ioutil.ReadAll(lf)
		if err != nil {
			b.Fatalf("Error reading everything from line: %q", err)
		}
	}
}

func BenchmarkElasticSearchBatchFormatter(b *testing.B) {
	batch := NewBatch(50)
	for i := 0; i < 25; i++ {
		batch.Add(LogLineOne)
		batch.Add(LogLineTwo)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bf, _ := NewElasticSearchBatchFormatter(batch, noErrData, &config)
		_, err := ioutil.ReadAll(bf)
		if err != nil {
			b.Fatalf("Error reading everything from line: %q", err)
		}
	}
}

func BenchmarkElasticSearchBatchFormatter_WithHeaders(b *testing.B) {
	config.SkipHeaders = true
	defer func () { config.SkipHeaders = false }()

	batch := NewBatch(50)
	for i := 0; i < 25; i++ {
		batch.Add(LogLineOneWithHeaders)
		batch.Add(LogLineTwoWithHeaders)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bf, _ := NewElasticSearchBatchFormatter(batch, noErrData, &config)
		_, err := ioutil.ReadAll(bf)
		if err != nil {
			b.Fatalf("Error reading everything from line: %q", err)
		}
	}
}



