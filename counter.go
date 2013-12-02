package main

import (
	"sync/atomic"
)

type Counter struct {
	value uint64
}

func (c *Counter) Read() uint64 {
	return c.value
}

func (c *Counter) ReadAndReset() uint64 {
	for {
		oldCount := c.value
		if atomic.CompareAndSwapUint64(&c.value, oldCount, 0) {
			return oldCount
		}
	}
}

func (c *Counter) Add(u uint64) uint64 {
	return atomic.AddUint64(&c.value, u)
}
