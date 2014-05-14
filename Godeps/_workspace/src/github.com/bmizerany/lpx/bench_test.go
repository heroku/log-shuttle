package lpx

import (
	"bufio"
	"bytes"
	"testing"
)

func BenchmarkNext(b *testing.B) {
	const data = `66 <174>1 2012-07-22T00:06:26-00:00 somehost Go console 2 Hi from Go
67 <174>1 2013-07-22T00:06:26-00:00 somehost Go console 10 Hi from Py
`
	b.StopTimer()
	b.SetBytes(int64(len(data)))
	for i := 0; i < b.N; i++ {
		r := NewReader(bufio.NewReader(bytes.NewBufferString(data)))
		b.StartTimer()
		for r.Next() {
		}
		b.StopTimer()
	}
}
