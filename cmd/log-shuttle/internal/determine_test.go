package internal

import "testing"

func TestDetermineAWSRegion(t *testing.T) {
	for _, tc := range []struct {
		input    string
		expected string
		err      bool
	}{
		{input: "foo.bar.com", expected: "", err: true},
		{input: "", expected: "", err: true},
		{input: "logs.foo.amazonaws.com", expected: "foo", err: false},
		{input: "logs.us-east-2.amazonaws.com", expected: "us-east-2", err: false},
		{input: "logs.us-east-2.amazonaws.com.cn", expected: "us-east-2", err: false},
		{input: "ec2-54-203-136-119.us-west-2.compute.amazonaws.com", expected: "us-west-2", err: false},
	} {
		t.Run(tc.input, func(t *testing.T) {
			out, err := DetermineAWSRegion(tc.input)
			if out != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, out)
			}
			if !tc.err && err != nil {
				t.Errorf("expected no error, got %q", err)
			}

		})
	}
}

func TestDetermineAWSService(t *testing.T) {
	for _, tc := range []struct {
		input    string
		expected string
		err      bool
	}{
		{input: "foo.bar.com", expected: "", err: true},
		{input: "", expected: "", err: true},
		{input: "ec2-54-203-136-119.us-west-2.compute.amazonaws.com", expected: "", err: true}, // not a valid service
		{input: "logs.foo.amazonaws.com", expected: "logs", err: false},
		{input: "kinesis.us-east-2.amazonaws.com", expected: "kinesis", err: false},
		{input: "kinesis.us-east-2.amazonaws.com.cn", expected: "kinesis", err: false},
	} {
		t.Run(tc.input, func(t *testing.T) {
			out, err := DetermineAWSService(tc.input)
			if out != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, out)
			}
			if !tc.err && err != nil {
				t.Errorf("expected no error, got %q", err)
			}

		})
	}
}
