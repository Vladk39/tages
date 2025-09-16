package cache

import (
	"Tages/internal/dto"
	"sync"

	"github.com/sirupsen/logrus"
)

type CacheInterface interface {
	Set(f dto.File)
	GetFilesFromCache() []dto.File
	Warm(files []dto.File)
}

type Cache struct {
	data          map[string]dto.File
	rm            sync.RWMutex
	logger        *logrus.Logger
	cacheDetector chan bool
	enabled       bool
}

func NewCache(logger *logrus.Logger, cacheDetector chan bool) *Cache {
	return &Cache{
		data:          make(map[string]dto.File),
		logger:        logger,
		cacheDetector: cacheDetector,
		enabled:       true,
	}
}
func (c *Cache) RunWatcher() {
	go func() {
		for state := range c.cacheDetector {
			c.rm.Lock()
			c.enabled = false
			c.rm.Unlock()
			if !state {
				c.logger.Warn("Cache disabled due to high CPU load")
				return
			}
		}
	}()
}

func (c *Cache) Set(f dto.File) {
	c.rm.RLock()
	enabled := c.enabled
	c.rm.RUnlock()

	if !enabled {
		c.logger.Debug("Skipping cache write (disabled)")
		return
	}

	c.rm.Lock()
	c.data[f.Name] = f
	c.rm.Unlock()
}

func (c *Cache) GetFilesFromCache() []dto.File {
	c.rm.RLock()
	defer c.rm.RUnlock()

	files := make([]dto.File, 0, len(c.data))

	for _, v := range c.data {
		files = append(files, v)
	}

	return files
}

func (c *Cache) Warm(files []dto.File) {
	c.rm.Lock()
	for _, v := range files {
		c.data[v.Name] = v
	}
	c.rm.Unlock()
}
