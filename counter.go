package shuttle

import (
	"sync"
	"time"
)

// Counter is used to track 2 values for a given metric. The first item is the
// "all time" metric counterand the second is the last value since the metric was
// ReadAndReset. Counters are safe for concurrent use.
type Counter struct {
	value        int
	allTimeValue int
	lastRAR      time.Time
	sync.Mutex
}

// NewCounter returns a new Counter initialized to the initial value
func NewCounter(initial int) *Counter {
	return &Counter{value: initial, allTimeValue: initial, lastRAR: time.Now()}
}

// Read returns the current value of the Counter
func (c *Counter) Read() int {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	return c.value
}

// AllTime returns the current alltime value of the Counter
func (c *Counter) AllTime() int {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	return c.allTimeValue
}

// ReadAndReset returns the current value and the last time it was reset, then
// resets the value and the last reset time to time.Now()
func (c *Counter) ReadAndReset() (int, time.Time) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	defer func() {
		c.value = 0
		c.lastRAR = time.Now()
	}()
	return c.value, c.lastRAR
}

// Add increments the counter (alltime and current), returning the new value
func (c *Counter) Add(u int) int {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	c.allTimeValue += u
	c.value += u
	return c.value
}
