package unique

import "sync"

type Idempotency struct {
	idx    int
	values []string
	mtx    sync.Mutex
}

func (i *Idempotency) lookup(given string) int {
	i.mtx.Lock()
	defer i.mtx.Unlock()

	for idx, val := range i.values {
		if val == given {
			return idx
		}
	}

	return -1
}

func (i *Idempotency) add(given string) {
	i.mtx.Lock()
	defer i.mtx.Unlock()

	if i.idx == (len(i.values) - 1) {
		i.idx = 0
	}

	i.values[i.idx] = given
	i.idx++
}

func (i *Idempotency) IsUnique(given string) bool {
	if i.lookup(given) != -1 {
		return false
	}

	i.add(given)
	return true
}

// New creates an Idempotency object
// NOTE: Use the optimal size for your Idempotency. very small
func New(size int) *Idempotency {
	return &Idempotency{
		values: make([]string, size),
	}
}
