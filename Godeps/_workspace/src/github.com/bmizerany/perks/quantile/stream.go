// Package quantile computes approximate quantiles over an unbounded data
// stream within low memory and CPU bounds.
//
// A small amount of accuracy is traded to achieve the above properties.
//
// Multiple streams can be merged before calling Query to generate a single set
// of results. This is meaningful when the streams represent the same type of
// data. See Merge and Samples.
//
// For more detailed information about the algorithm used, see:
//
// Effective Computation of Biased Quantiles over Data Streams
//
// http://www.cs.rutgers.edu/~muthu/bquant.pdf
package quantile

import (
	"container/list"
	"math"
	"sort"
)

// Sample holds an observed value and meta information for compression. JSON
// tags have been added for convenience.
type Sample struct {
	Value float64 `json:",string"`
	Width float64 `json:",string"`
	Delta float64 `json:",string"`
}

// Samples represents a slice of samples. It implements sort.Interface.
type Samples []Sample

func (a Samples) Len() int {
	return len(a)
}

func (a Samples) Less(i, j int) bool {
	return a[i].Value < a[j].Value
}

func (a Samples) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

type invariant func(s *stream, r float64) float64

// NewBiased returns an initialized Stream for high-biased quantiles (e.g.
// 50th, 90th, 99th) not known a priori with ﬁner error guarantees for the
// higher ranks of the data distribution.
// See http://www.cs.rutgers.edu/~muthu/bquant.pdf for time, space, and error properties.
func NewBiased() *Stream {
	ƒ := func(s *stream, r float64) float64 {
		return 2 * s.epsilon * r
	}
	return newStream(ƒ)
}

// NewTargeted returns an initialized Stream concerned with a particular set of
// quantile values that are supplied a priori. Knowing these a priori reduces
// space and computation time.
// See http://www.cs.rutgers.edu/~muthu/bquant.pdf for time, space, and error properties.
func NewTargeted(quantiles ...float64) *Stream {
	ƒ := func(s *stream, r float64) float64 {
		var m float64 = math.MaxFloat64
		var f float64
		for _, q := range quantiles {
			if q*s.n <= r {
				f = (2 * s.epsilon * r) / q
			} else {
				f = (2 * s.epsilon * (s.n - r)) / (1 - q)
			}
			m = math.Min(m, f)
		}
		return m
	}
	return newStream(ƒ)
}

// Stream computes quantiles for a stream of float64s. It is not thread-safe by
// design. Take care when using across multiple goroutines.
type Stream struct {
	*stream
	b      Samples
	sorted bool
}

func newStream(ƒ invariant) *Stream {
	const defaultEpsilon = 0.01
	x := &stream{epsilon: defaultEpsilon, ƒ: ƒ, l: list.New()}
	return &Stream{x, make(Samples, 0, 500), true}
}

// Insert inserts v into the stream.
func (s *Stream) Insert(v float64) {
	s.insert(Sample{Value: v, Width: 1})
}

func (s *Stream) insert(sample Sample) {
	s.b = append(s.b, sample)
	s.sorted = false
	if len(s.b) == cap(s.b) {
		s.flush()
		s.compress()
	}
}

// Query returns the computed qth percentiles value. If s was created with
// NewTargeted, and q is not in the set of quantiles provided a priori, Query
// will return an unspecified result.
func (s *Stream) Query(q float64) float64 {
	if !s.flushed() {
		// Fast path when there hasn't been enough data for a flush;
		// this also yeilds better accuracy for small sets of data.
		l := len(s.b)
		if l == 0 {
			return 0
		}
		i := int(float64(l) * q)
		if i > 0 {
			i -= 1
		}
		s.maybeSort()
		return s.b[i].Value
	}
	s.flush()
	return s.stream.query(q)
}

// Merge merges samples into the underlying streams samples. This is handy when
// merging multiple streams from separate threads, database shards, etc.
func (s *Stream) Merge(samples Samples) {
	sort.Sort(samples)
	s.stream.merge(samples)
}

// Reset reinitializes and clears the list reusing the samples buffer memory.
func (s *Stream) Reset() {
	s.stream.reset()
	s.b = s.b[:0]
}

// Samples returns stream samples held by s.
func (s *Stream) Samples() Samples {
	if !s.flushed() {
		return s.b
	}
	s.flush()
	s.compress()
	return s.stream.samples()
}

// Count returns the total number of samples observed in the stream
// since initialization.
func (s *Stream) Count() int {
	return len(s.b) + s.stream.count()
}

func (s *Stream) flush() {
	s.maybeSort()
	s.stream.merge(s.b)
	s.b = s.b[:0]
}

func (s *Stream) maybeSort() {
	if !s.sorted {
		s.sorted = true
		sort.Sort(s.b)
	}
}

func (s *Stream) flushed() bool {
	return s.stream.l.Len() > 0
}

type stream struct {
	epsilon float64
	n       float64
	l       *list.List
	ƒ       invariant
}

// SetEpsilon sets the error epsilon for the Stream. The default epsilon is
// 0.01 and is usually satisfactory. If needed, this must be called before all
// Inserts.
// To learn more, see: http://www.cs.rutgers.edu/~muthu/bquant.pdf
func (s *stream) SetEpsilon(epsilon float64) {
	s.epsilon = epsilon
}

func (s *stream) reset() {
	s.l.Init()
	s.n = 0
}

func (s *stream) insert(v float64) {
	fn := s.mergeFunc()
	fn(v, 1)
}

func (s *stream) merge(samples Samples) {
	fn := s.mergeFunc()
	for _, s := range samples {
		fn(s.Value, s.Width)
	}
}

func (s *stream) mergeFunc() func(v, w float64) {
	// NOTE: I used a goto over defer because it bought me a few extra
	// nanoseconds. I know. I know.
	var r float64
	e := s.l.Front()
	return func(v, w float64) {
		for ; e != nil; e = e.Next() {
			c := e.Value.(*Sample)
			if c.Value > v {
				sm := &Sample{v, w, math.Floor(s.ƒ(s, r)) - 1}
				s.l.InsertBefore(sm, e)
				goto inserted
			}
			r += c.Width
		}
		s.l.PushBack(&Sample{v, w, 0})
	inserted:
		s.n += w
	}
}

func (s *stream) count() int {
	return int(s.n)
}

func (s *stream) query(q float64) float64 {
	e := s.l.Front()
	t := math.Ceil(q * s.n)
	t += math.Ceil(s.ƒ(s, t) / 2)
	p := e.Value.(*Sample)
	e = e.Next()
	r := float64(0)
	for e != nil {
		c := e.Value.(*Sample)
		if r+c.Width+c.Delta > t {
			return p.Value
		}
		r += p.Width
		p = c
		e = e.Next()
	}
	return p.Value
}

func (s *stream) compress() {
	if s.l.Len() < 2 {
		return
	}
	e := s.l.Back()
	x := e.Value.(*Sample)
	r := s.n - 1 - x.Width
	e = e.Prev()
	for e != nil {
		c := e.Value.(*Sample)
		if c.Width+x.Width+x.Delta <= s.ƒ(s, r) {
			x.Width += c.Width
			o := e
			e = e.Prev()
			s.l.Remove(o)
		} else {
			x = c
			e = e.Prev()
		}
		r -= c.Width
	}
}

func (s *stream) samples() Samples {
	samples := make(Samples, 0, s.l.Len())
	for e := s.l.Front(); e != nil; e = e.Next() {
		samples = append(samples, *e.Value.(*Sample))
	}
	return samples
}
