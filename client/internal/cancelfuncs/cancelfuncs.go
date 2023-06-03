package cancelfuncs

import (
	"context"
	"sync"
)

type CancelFuncs struct {
	mu   sync.Mutex
	data map[int64]context.CancelFunc
}

func NewCancelFuncs() *CancelFuncs {
	return &CancelFuncs{
		data: make(map[int64]context.CancelFunc),
	}
}

func (c *CancelFuncs) Add(id int64, cancel context.CancelFunc) {
	c.mu.Lock()
	c.data[id] = cancel
	c.mu.Unlock()
}

func (c *CancelFuncs) Remove(id int64) {
	c.mu.Lock()
	if cancel, exists := c.data[id]; exists {
		cancel()
		delete(c.data, id)
	}
	c.mu.Unlock()
}
