package middleware

import (
	"time"

	"github.com/nulloop/chu"
)

// DetectInactivity accepts a timeout and returns wait and middleware
// wait can be used to wait function will be blocked until timeout passes.
// Wait can be used until none of the handlers are active and this indicates that this is a good time to be
// connected to public. In other words, this middleware performes warmup for your services
func DetectInactivity(timeout time.Duration) (func(), func(chu.Handler) chu.Handler) {
	wait, tick := HeartBeat(timeout)

	middleware := func(h chu.Handler) chu.Handler {
		return chu.HandlerFunc(func(msg chu.Message) error {
			tick()
			return h.ServeMessage(msg)
		})
	}

	return wait, middleware
}
