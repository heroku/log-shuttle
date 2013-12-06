package main

import (
	"sync"
)

type Counter struct {
	value        uint64
	allTimeValue uint64
	sync.Mutex
}

func NewCounter(initial uint64) *Counter {
	return &Counter{value: initial, allTimeValue: initial}
}

func (c *Counter) Read() uint64 {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	return c.value
}

func (c *Counter) AllTime() uint64 {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	return c.allTimeValue
}

func (c *Counter) ReadAndReset() uint64 {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	defer func() { c.value = 0 }()
	return c.value
}

func (c *Counter) Add(u uint64) uint64 {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	c.allTimeValue += u
	c.value += u
	return c.value
}
