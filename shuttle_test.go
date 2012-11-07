package main

import (
	"fmt"
	"testing"
)

func TestPrepare(t *testing.T) {
	res := prepare([]string{"hello", "goodbye"})
	expected := "92 <0>1 2012-10-18T05:56:10Z 1234 5678 - hello\n <0>1 2012-10-18T05:56:10Z 1234 5678 - goodbye\n"
	if res != expected {
		fmt.Printf("val=%v", res)
		t.Error(res)
	}
}
