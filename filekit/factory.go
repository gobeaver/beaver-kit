package filekit

import (
	"fmt"
	"sync"
)

// DriverFactory is a function that creates a FileSystem from a config
type DriverFactory func(cfg Config) (FileSystem, error)

var (
	driverFactories = make(map[string]DriverFactory)
	factoryMutex    sync.RWMutex
)

// RegisterDriver registers a driver factory function
func RegisterDriver(name string, factory DriverFactory) {
	factoryMutex.Lock()
	defer factoryMutex.Unlock()
	driverFactories[name] = factory
}

// CreateDriver creates a driver instance from config
func CreateDriver(cfg Config) (FileSystem, error) {
	factoryMutex.RLock()
	factory, exists := driverFactories[cfg.Driver]
	factoryMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("driver %s not registered", cfg.Driver)
	}

	return factory(cfg)
}
