package quantile

import (
	"testing"
)

func BenchmarkInsert(b *testing.B) {
	b.StopTimer()
	s := NewTargeted(0.01, 0.5, 0.9, 0.99)
	b.StartTimer()
	for i := float64(0); i < float64(b.N); i++ {
		s.Insert(i)
	}
}
