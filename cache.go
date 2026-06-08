package main

import (
	"fmt"
	"sync"
	"time"
)

type CacheEntry struct {
	Response  []byte
	ExpiresAt time.Time
}

type DNSCache struct {
	mu    sync.RWMutex
	items map[string]CacheEntry
}

func (c *DNSCache) Set(name string, qtype uint16, response []byte, answers []DNSResourceRecord) {
	if len(answers) == 0 {
		return
	}

	minTTL := answers[0].TTL

	for _, answer := range answers {
		if answer.TTL < minTTL {
			minTTL = answer.TTL
		}
	}

	cacheEntry := CacheEntry{
		Response:  response,
		ExpiresAt: time.Now().Add(time.Duration(minTTL) * time.Second),
	}

	key := cacheKey(name, qtype)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = cacheEntry
}

func (c *DNSCache) Get(name string, qtype uint16) ([]byte, bool) {
	key := cacheKey(name, qtype)

	c.mu.RLock()
	entry, ok := c.items[key]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()

		return nil, false
	}

	return entry.Response, true
}

func (c *DNSCache) StartCleanup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			<-ticker.C

			now := time.Now()

			c.mu.Lock()

			for key, entry := range c.items {
				if now.After(entry.ExpiresAt) {
					delete(c.items, key)
				}
			}

			c.mu.Unlock()
		}
	}()
}

func NewDNSCache() *DNSCache {
	return &DNSCache{
		items: make(map[string]CacheEntry),
	}
}

func cacheKey(name string, qtype uint16) string {
	return fmt.Sprintf("%s:%d", name, qtype)
}
