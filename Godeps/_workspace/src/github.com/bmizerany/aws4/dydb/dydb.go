// This is an experimental library for use with DynamoDB. It uses
// github.com/bmizerany/aws4 to sign requests. See Example for use.
package dydb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/bmizerany/aws4"
	"net/http"
	"strings"
)

// A ResponseError is returned by Decode when an error communicating with
// DynamoDB occurs.
type ResponseError struct {
	StatusCode int
	Type       string
	Message    string
}

// IsException returns true if err is a ResponseError whos TypeName() equals
// name; false otherwise.
func IsException(err error, name string) bool {
	if e, ok := err.(*ResponseError); ok {
		return e.TypeName() == name
	}
	return false
}

func (e *ResponseError) Error() string {
	return fmt.Sprintf("dydb: %d - %s - %q", e.StatusCode, e.TypeName(), e.Message)
}

// TypeName returns the error Type without the namespace.
func (e *ResponseError) TypeName() string {
	i := strings.Index(e.Type, "#")
	if i < 0 {
		return ""
	}
	return e.Type[i+1:]
}

type errorDecoder struct {
	err error
}

func (e *errorDecoder) Decode(v interface{}) error {
	return e.err
}

type Decoder interface {
	Decode(v interface{}) error
}

const (
	DefaultURL     = "https://dynamodb.us-east-1.amazonaws.com/"
	DefaultVersion = "20120810"
)

type DB struct {
	// The version of DynamoDB to use. If empty string, DefaultVersion is
	// used.
	Version string

	// If nil, aws4.DefaultClient is used.
	Client *aws4.Client

	// If empty, DefaultURL is used.
	URL string
}

// Exec is like Query, but discards the response. It returns the error if there
// was one.
func (db *DB) Exec(action string, v interface{}) error {
	var x struct{}
	return db.Query(action, v).Decode(&x)
}

// Query executes an action with a JSON-encoded v as the body.  A nil v is
// represented as the JSON value {}. If an error occurs while communicating
// with DynamoDB, Query returns a Decoder that returns only the error,
// otherwise a json.Decoder is returned.
func (db *DB) Query(action string, v interface{}) Decoder {
	cl := db.Client
	if cl == nil {
		cl = aws4.DefaultClient
	}

	url := db.URL
	if url == "" {
		url = DefaultURL
	}

	ver := db.Version
	if ver == "" {
		ver = DefaultVersion
	}

	if v == nil {
		v = struct{}{}
	}

	b, err := json.Marshal(v)
	if err != nil {
		return &errorDecoder{err: err}
	}

	r, err := http.NewRequest("POST", url, bytes.NewBuffer(b))
	if err != nil {
		return &errorDecoder{err: err}
	}
	r.Header.Set("Content-Type", "application/x-amz-json-1.0")
	r.Header.Set("X-Amz-Target", "DynamoDB_"+ver+"."+action)

	resp, err := cl.Do(r)
	if err != nil {
		return &errorDecoder{err: err}
	}

	if code := resp.StatusCode; code != 200 {
		// Read the whole body in so that Keep-Alives may be released back to the pool.
		var e struct {
			Message string
			Type    string `json:"__type"`
		}
		json.NewDecoder(resp.Body).Decode(&e)
		return &errorDecoder{err: &ResponseError{code, e.Type, e.Message}}
	}
	return json.NewDecoder(resp.Body)
}
