package shuttle

import (
	"log"
	"net"
	"time"
)

type Socket struct {
	net.Listener
	Shuttle *Shuttle
}

func NewSocket(s *Shuttle) *Socket {
	l, err := net.Listen("unix", s.config.Socket)
	if err != nil {
		log.Fatalln(err)
	}

	return &Socket{
		Listener: l,
		Shuttle:  s,
	}
}

func (k *Socket) Start() {
	for {
		c, err := k.Accept()
		if err != nil {
			if err, ok := err.(*net.OpError); ok && err.Timeout() {
				continue
			}
			return
		}
		c.SetDeadline(time.Now().Add(1 * time.Second))
		k.Shuttle.ReadLogLines(c)
	}
}
