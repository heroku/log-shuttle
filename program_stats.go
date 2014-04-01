package main

import (
	"fmt"
	"github.com/bmizerany/perks/quantile"
	"net"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

func Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func cleanUpSocket(path string) error {
	if Exists(path) {
		return os.Remove(path)
	}
	return nil
}

type NamedValue struct {
	value float64
	name  string
}

func NewNamedValue(name string, value float64) NamedValue {
	return NamedValue{name: name, value: value}
}

type ProgramStats struct {
	lost, drops      *Counter
	stats            map[string]*quantile.Stream
	input            <-chan NamedValue
	lastPoll         time.Time
	exposeStats      bool
	network, address string
	sync.Mutex
}

// Returns a new ProgramStats instance aggregating stats from the input channel
// You will need to Listen() seperately if you need / want to export stats
// polling
func NewProgramStats(listen string, lost, drops *Counter, input <-chan NamedValue) *ProgramStats {
	var network, address string
	if len(listen) == 0 {
		network = ""
		address = ""
	} else {
		netDeets := strings.Split(listen, ",")
		switch len(netDeets) {
		case 2:
			network = netDeets[0]
			address = netDeets[1]
		default:
			ErrLogger.Fatalf("Invalid -stats-addr (%s). Must be of form <net>,<addr> (e.g. unix,/tmp/ff or tcp,:8080)\n", listen)
		}
	}

	ps := ProgramStats{
		input:       input,
		lost:        lost,
		drops:       drops,
		lastPoll:    time.Now(),
		network:     network,
		address:     address,
		exposeStats: network != "",
		stats:       make(map[string]*quantile.Stream),
	}

	go ps.aggregateValues()

	return &ps
}

// Listen for stats requests if we should
func (stats *ProgramStats) Listen() {
	if stats.exposeStats {
		unixSocket := stats.network == "unix"

		if unixSocket {
			err := cleanUpSocket(stats.address)
			if err != nil {
				ErrLogger.Fatalf("Unable to remove old stats socket (%s): %s\n", stats.address, err)
			}
		}

		listener, err := net.Listen(stats.network, stats.address)
		if err != nil {
			ErrLogger.Fatalf("Unable to listen on %s,%s: %s\n", stats.network, stats.address, err)
		}

		go func() {
			stats.accept(listener)
			stats.cleanup(listener)
		}()
	}
}

// Cleanup after ourselves
// TODO(edwardam): Chances are that we won't get here because we'll exit before
// this
func (stats *ProgramStats) cleanup(listener net.Listener) {
	if listener != nil {
		listener.Close()
	}

	if stats.network == "unix" {
		if Exists(stats.address) {
			err := os.Remove(stats.address)
			if err != nil {
				ErrLogger.Printf("Unable to remove socket (%s): %s\n", stats.address, err)
			}
		}
	}
}

// Aggregate the values by name as them come in via the input channel
func (stats *ProgramStats) aggregateValues() {
	for namedValue := range stats.input {
		stats.Mutex.Lock()

		sample, ok := stats.stats[namedValue.name]
		// Zero value not good enough, so initialize
		if !ok {
			sample = quantile.NewTargeted(0.50, 0.95, 0.99)
		}

		sample.Insert(namedValue.value)
		stats.stats[namedValue.name] = sample
		stats.Mutex.Unlock()
	}
}

// Accept connections and handle them
func (stats *ProgramStats) accept(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			ErrLogger.Printf("Error accepting connection: %s\n", err)
			break
		}
		go stats.handleConnection(conn)
	}
}

// we create a buffer (output) in order to sort the output
func (stats *ProgramStats) handleConnection(conn net.Conn) {
	defer conn.Close()

	snapshot := stats.Snapshot(false)
	output := make([]string, 0, len(snapshot))

	for key, value := range snapshot {
		output = append(output, fmt.Sprintf("%s: %v\n", key, value))
	}

	sort.Strings(output)

	for i := range output {
		_, err := conn.Write([]byte(output[i]))
		if err != nil {
			ErrLogger.Printf("Error writting stats out: %s\n", err)
		}
	}
}

// Produces a point in time snapshot of the quantiles/other stats
// If reset is true, then will call Reset() on each of the quantiles
func (stats *ProgramStats) Snapshot(reset bool) map[string]interface{} {
	snapshot := make(map[string]interface{})
	// We don't need locks for these values
	snapshot["log-shuttle.alltime.drops.count"] = stats.drops.AllTime()
	snapshot["log-shuttle.alltime.lost.count"] = stats.lost.AllTime()

	stats.Mutex.Lock()
	defer stats.Mutex.Unlock()
	snapshot["log-shuttle.last.stats.connection.since.seconds"] = time.Now().Sub(stats.lastPoll).Seconds()

	for name, stream := range stats.stats {
		base := "log-shuttle." + name + "."
		snapshot[base+"count"] = stream.Count()
		snapshot[base+"p50.seconds"] = stream.Query(0.50)
		snapshot[base+"p95.seconds"] = stream.Query(0.95)
		snapshot[base+"p99.seconds"] = stream.Query(0.99)
		if reset {
			stream.Reset()
		}
	}

	return snapshot
}
