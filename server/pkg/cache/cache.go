package cache

import (
	"os"
	"path/filepath"
	"sync"
	"time"
)

type entry struct {
	path    string
	size    int64
	addedAt time.Time
	refs    int
}

type Cache struct {
	mu      sync.RWMutex
	dir     string
	entries map[string]*entry
}

func NewCache(dir string) (*Cache, error) {
	return &Cache{dir: dir, entries: make(map[string]*entry)}, nil
}

func (c *Cache) Add(key, srcPath string) error {
	dest := filepath.Join(c.dir, filepath.FromSlash(key))
	if err := os.Rename(srcPath, dest); err != nil {
		return err
	}

	info, err := os.Stat(dest)
	if err != nil {
		return err
	}
	//Should the mutex call be earlier?
	c.mu.Lock()
	defer c.mu.Unlock()

	//Should check for existing entry?

	c.entries[key] = &entry{
		path:    dest,
		size:    info.Size(),
		addedAt: time.Now(),
	}
	return nil
}
