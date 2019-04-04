package heartbeat

import (
	"time"
)

type signal struct {
	done    chan struct{}
	data    chan struct{}
	timeout time.Duration
}

var empty struct{}

func (s *signal) tick() {
	select {
	case <-s.done:
	case s.data <- empty:
	}
}

func (s *signal) next() bool {
	select {
	case _, ok := <-s.data:
		return ok
	case <-time.After(s.timeout):
		return false
	}
}

func New(timeout time.Duration) (wait func(), tick func()) {
	signal := &signal{
		done:    make(chan struct{}),
		data:    make(chan struct{}),
		timeout: timeout,
	}

	wait = func() {
		<-signal.done
	}

	tick = func() {
		signal.tick()
	}

	go func() {
		defer close(signal.done)
		for {
			if !signal.next() {
				break
			}
		}
	}()

	return wait, tick
}
