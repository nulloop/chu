package middleware

import (
	"sync"

	"github.com/nulloop/chu"
)

// IdempotentQuery is an interface which pass to Idempotent middleware
type IdempotentQuery interface {
	Exists(chu.Message) bool
}

// Idempotent is a middle ware which drop duplicate messages
// based on same id
func Idempotent(query IdempotentQuery) func(chu.Handler) chu.Handler {
	return func(h chu.Handler) chu.Handler {
		return chu.HandlerFunc(func(msg chu.Message) error {
			if query.Exists(msg) {
				return nil
			}

			return h.ServeMessage(msg)
		})
	}
}

// RotatingIdempotentQuery is simple in-memory which cached nth recent ids
// it does rotate and reuse the same memory. It also thread safe
type RotatingIdempotentQuery struct {
	ids  []string
	idx  int
	size int
	lock sync.Mutex
}

// Exists checks whether id already seen recently
func (r *RotatingIdempotentQuery) Exists(msg chu.Message) bool {
	r.lock.Lock()
	defer r.lock.Unlock()

	id := msg.ID()

	// check and see if id exists
	for i := 0; i < r.size; i++ {
		if r.ids[i] == id {
			return true
		}
	}

	// rotate/reuse memory
	r.idx++
	if r.idx == r.size {
		r.idx = 0
	}
	r.ids[r.idx] = id

	return false
}

// NewRotatingIdempotentQuery creates RotatingIdempotentQuery with fixed size
func NewRotatingIdempotentQuery(size int) *RotatingIdempotentQuery {
	return &RotatingIdempotentQuery{
		ids:  make([]string, size),
		idx:  0,
		size: size,
	}
}
