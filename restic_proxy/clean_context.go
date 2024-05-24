package resticProxy

import "sync"

type CleanCtx struct {
	sync.Mutex
	list []func()
}

func NewCleanCtx() *CleanCtx {
	return &CleanCtx{
		list: make([]func(), 0),
	}
}

func (c *CleanCtx) AddCleanCtx(f func()) {
	c.Lock()
	defer c.Unlock()
	c.list = append(c.list, f)
}

func (c *CleanCtx) Cleanup() {
	c.Lock()
	defer c.Unlock()
	for _, l := range c.list {
		l()
	}
	c.list = nil
}
