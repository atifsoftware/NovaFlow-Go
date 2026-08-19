package core

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Config holds key/value pairs loaded from .env plus a thread-safe accessor API.
type Config struct {
	mu     sync.RWMutex
	values map[string]string
}

var globalConfig *Config
var once sync.Once

// LoadEnv reads a .env-style file (KEY=VALUE per line, '#' comments, blank
// lines ignored) and merges it with real OS environment variables. OS env
// vars always win, so production deployments can override .env values
// without editing the file.
func LoadEnv(path string) (*Config, error) {
	c := &Config{values: make(map[string]string)}

	file, err := os.Open(path)
	if err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, `"'`)
			c.values[key] = val
		}
	}
	// OS environment variables always win — load ALL of them unconditionally.
	// This means production can inject entirely new keys (e.g. DATABASE_URL)
	// without requiring them to be listed in .env first.
	for _, kv := range os.Environ() {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) == 2 {
			c.values[parts[0]] = parts[1]
		}
	}

	globalConfig = c
	return c, nil
}

// Env returns the global config instance, initializing an empty one if
// LoadEnv was never called (useful in tests).
func Env() *Config {
	once.Do(func() {
		if globalConfig == nil {
			globalConfig = &Config{values: make(map[string]string)}
		}
	})
	return globalConfig
}

func (c *Config) Get(key, fallback string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if v, ok := c.values[key]; ok && v != "" {
		return v
	}
	return fallback
}

func (c *Config) GetInt(key string, fallback int) int {
	v := c.Get(key, "")
	if v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
}

func (c *Config) GetBool(key string, fallback bool) bool {
	v := c.Get(key, "")
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func (c *Config) Set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[key] = value
}
