package shuttle

import (
	"sync"
	"time"
)

type Counter struct {
	value        int
	allTimeValue int
	lastRAR      time.Time
	sync.Mutex
}

func NewCounter(initial int) *Counter {
	return &Counter{value: initial, allTimeValue: initial, lastRAR: time.Now()}
}

func (c *Counter) Read() int {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	return c.value
}

func (c *Counter) AllTime() int {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	return c.allTimeValue
}

func (c *Counter) ReadAndReset() (int, time.Time) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	defer func() {
		c.value = 0
		c.lastRAR = time.Now()
	}()
	return c.value, c.lastRAR
}

func (c *Counter) Add(u int) int {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	c.allTimeValue += u
	c.value += u
	return c.value
}
