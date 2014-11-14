package shuttle

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/bmizerany/aws4"
	"io"
	"net/http"
)

type KinesisFormatter struct {
	header     *bytes.Buffer
	formatters []io.Reader
	footer     *bytes.Reader
	rdr        io.Reader
	keys       *aws4.Keys
	host       string
}

func NewKinesisFormatter(b Batch, eData []errData, config *ShuttleConfig) HttpFormatter {
	fmt.Println("Kinesis")
	kf := &KinesisFormatter{
		header:     bytes.NewBuffer(make([]byte, 0, 500)),
		formatters: make([]io.Reader, 0, b.MsgCount()+len(eData)),
		footer:     bytes.NewReader([]byte{'"', '}'}),
		keys: &aws4.Keys{
			AccessKey: config.AwsAccessKey,
			SecretKey: config.AwsSecretKey,
		},
		host: config.AwsHost,
	}
	kf.header.WriteString("{")
	kf.header.WriteString(fmt.Sprintf("\"StreamName\":\"%s\",", config.KinesisStreamName))
	kf.header.WriteString(fmt.Sprintf("\"PartitionKey\":\"%s\",", config.Appname))
	kf.header.WriteString("\"Data\":\"")

	for _, edata := range eData {
		kf.formatters = append(kf.formatters, NewLogplexErrorFormatter(edata, *config))
	}

	for _, l := range b.logLines {
		kf.formatters = append(kf.formatters, NewLogplexLineFormatter(l, config))
	}

	return kf
}

func (kf *KinesisFormatter) Request(url string) (*http.Request, error) {
	req, err := http.NewRequest("POST", url, kf)
	if err != nil {
		return nil, err
	}

	//TODO: Setup content types and stuff
	// See: http://docs.aws.amazon.com/kinesis/latest/APIReference/API_PutRecord.html

	req.Header.Add("Content-Type", "application/x-amz-json-1.1")
	req.Header.Add("X-Amz-Target", "Kinesis_20131202.PutRecord")
	req.Host = kf.host

	err = aws4.Sign(kf.keys, req)
	if err != nil {
		return nil, err
	}

	return req, nil

}

func (kf *KinesisFormatter) Read(p []byte) (n int, err error) {
	if kf.rdr == nil {
		dataReader, dataWriter := io.Pipe()
		kf.rdr = io.MultiReader(kf.header, dataReader, kf.footer)
		go func() {
			encoder := base64.NewEncoder(base64.StdEncoding, dataWriter)
			//TODO: Handle errors somehow?
			io.Copy(encoder, io.MultiReader(kf.formatters...))
			encoder.Close()
			dataWriter.Close()
		}()
	}

	// header get's read completely
	// io.Pipe(

	return kf.rdr.Read(p)
}

func (kf *KinesisFormatter) MsgCount() int {
	return 0
}
