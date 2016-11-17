package shuttle

import "io"

// MultiShuttle duplicates log lines read from a single log source to a
// collection of underlying Shuttle instances to drain logs to multiple
// destinations.
type MultiShuttle struct {
	shuttles []*Shuttle
	reader   io.ReadCloser
	writers  []*io.PipeWriter
}

// NewMultiShuttle initializes and returns a new multishuttle.
func NewMultiShuttle() *MultiShuttle {
	return &MultiShuttle{
		shuttles: make([]*Shuttle, 0),
		writers:  make([]*io.PipeWriter, 0),
	}
}

// AddShuttle registers a shuttle to process log lines.
func (s *MultiShuttle) AddShuttle(shuttle *Shuttle) {
	s.shuttles = append(s.shuttles, shuttle)
}

// LoadReader into the multishuttle to make each line read from the log source
// available to each of the registered shuttles. AddShuttle should not be
// called after this function has been invoked.
func (s *MultiShuttle) LoadReader(rdr io.ReadCloser) {
	s.reader = rdr
	for _, shuttle := range s.shuttles {
		pr, pw := io.Pipe()
		s.writers = append(s.writers, pw)
		shuttle.LoadReader(pr)
	}
}

// Launch all the shuttles previously registered with AddShuttle. AddShuttle
// should not be called after this function has been invoked.
func (s *MultiShuttle) Launch() {
	for _, shuttle := range s.shuttles {
		shuttle.Launch()
	}

	go func() {
		defer func() {
			for _, pw := range s.writers {
				pw.Close()
			}
		}()

		_, err := io.Copy(s.newMultiWriter(), s.reader)
		if err != nil {
			// TODO(jkakar) What should we do when an error occurs? log.Fatal
			// and bail out?
		}
	}()
}

func (s *MultiShuttle) newMultiWriter() io.Writer {
	ww := make([]io.Writer, 0)
	for _, w := range s.writers {
		ww = append(ww, w)
	}
	return io.MultiWriter(ww...)
}

// WaitForReadersToFinish waits for each registered shuttle to finish reading.
func (s *MultiShuttle) WaitForReadersToFinish() {
	for _, shuttle := range s.shuttles {
		shuttle.WaitForReadersToFinish()
	}
}

// Land gracefully terminates all registered shuttles.
func (s *MultiShuttle) Land() {
	for _, shuttle := range s.shuttles {
		shuttle.Land()
	}
}
