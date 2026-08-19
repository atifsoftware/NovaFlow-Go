package core

import (
	"fmt"
	"sync"
)

// Container is a minimal IoC container: services are registered under a
// string key either as an eagerly-built singleton (Bind) or a factory
// invoked once and cached on first use (Singleton).
type Container struct {
	mu        sync.RWMutex
	instances map[string]interface{}
	factories map[string]func(*Container) interface{}
}

func NewContainer() *Container {
	return &Container{
		instances: map[string]interface{}{},
		factories: map[string]func(*Container) interface{}{},
	}
}

// Bind registers an already-constructed instance under key.
func (c *Container) Bind(key string, instance interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.instances[key] = instance
}

// Singleton registers a factory that is lazily invoked on first Make and
// cached for subsequent calls.
func (c *Container) Singleton(key string, factory func(*Container) interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.factories[key] = factory
}

// Make resolves a service by key, building it from its factory if needed.
func (c *Container) Make(key string) interface{} {
	c.mu.RLock()
	if inst, ok := c.instances[key]; ok {
		c.mu.RUnlock()
		return inst
	}
	factory, ok := c.factories[key]
	c.mu.RUnlock()
	if !ok {
		panic(fmt.Sprintf("container: no binding registered for %q", key))
	}
	inst := factory(c)
	c.mu.Lock()
	c.instances[key] = inst
	c.mu.Unlock()
	return inst
}
