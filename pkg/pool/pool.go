package pool

import (
	"errors"
	"sync"
)

// Possible errors
var (
	ErrClosed = errors.New("pool closed")
	ErrMax    = errors.New("reach max connection limit")
)

type request chan interface{}

type Config struct {
	MaxIdle   uint // The maximum number of idle items in a pool
	MaxActive uint // The maximum number of items that a pool can store
}

// Pool stores object for reusing, such as slava connection
type Pool struct {
	Config
	factory     func() (interface{}, error) // The factory function to create a new item
	finalizer   func(x interface{})         // The function to destroy an item
	idles       chan interface{}            // The channel to store the idle items
	waitingReqs []request                   // Requests to waiting for allocating an item
	activeCount uint                        // increases during creating item, decrease during destroying item
	mu          sync.Mutex
	closed      bool // Flag for the closed pool
}

// New initialize a pool, need the factory function, finalize item function and the configurations
func New(factory func() (interface{}, error), finalizer func(x interface{}), cfg Config) *Pool {
	return &Pool{
		factory:     factory,
		finalizer:   finalizer,
		idles:       make(chan interface{}, cfg.MaxIdle),
		waitingReqs: make([]request, 0),
		Config:      cfg,
	}
}

// getOnNoIdle try to create a new item or waiting for items being returned
// invoker should have pool.mu
func (pool *Pool) getOnNoIdle() (interface{}, error) {
	// Items reach the capacity of the pool, cannot create a new item
	if pool.activeCount >= pool.MaxActive {
		// waiting for item being returned
		req := make(chan interface{}, 1)
		pool.waitingReqs = append(pool.waitingReqs, req)
		pool.mu.Unlock()
		x, ok := <-req
		// No item can be arranged for the request (reach the item limit)
		if !ok {
			return nil, ErrMax
		}
		return x, nil
	}

	// create a new item
	pool.activeCount++ // hold a place for new item
	pool.mu.Unlock()
	x, err := pool.factory()
	if err != nil {
		// create failed return token
		pool.mu.Lock()
		pool.activeCount-- // release the holding place
		pool.mu.Unlock()
		return nil, err
	}
	return x, nil
}

// Get try to get an idle item from pool
func (pool *Pool) Get() (interface{}, error) {
	pool.mu.Lock()
	if pool.closed {
		pool.mu.Unlock()
		return nil, ErrClosed
	}

	select {
	case item := <-pool.idles:
		pool.mu.Unlock()
		return item, nil
	default:
		// no pooled item, create one
		return pool.getOnNoIdle()
	}
}

// Put try to put an idle item into the pool
func (pool *Pool) Put(x interface{}) {
	pool.mu.Lock()
	// if the pool is closed, directly destroy the item
	if pool.closed {
		pool.mu.Unlock()
		pool.finalizer(x)
		return
	}
	// If there is waiting requests directly allocate the item to the first request
	if len(pool.waitingReqs) > 0 {
		req := pool.waitingReqs[0]
		copy(pool.waitingReqs, pool.waitingReqs[1:])
		pool.waitingReqs = pool.waitingReqs[:len(pool.waitingReqs)-1]
		req <- x
		pool.mu.Unlock()
		return
	}
	// Store the item or destroy it
	select {
	case pool.idles <- x:
		pool.mu.Unlock()
		return
	default:
		// reach max idle, destroy redundant items
		pool.mu.Unlock()
		pool.activeCount--
		pool.finalizer(x)
	}
}

// Close try to close the pool
func (pool *Pool) Close() {
	pool.mu.Lock()
	if pool.closed {
		pool.mu.Unlock()
		return
	}
	// Change the flag and close the idle item channel
	pool.closed = true
	close(pool.idles)
	pool.mu.Unlock()
	// Destroy all the left idle items in the pool
	for x := range pool.idles {
		pool.finalizer(x)
	}
}
