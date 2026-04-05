// ALR - Any Linux Repository
// Copyright (C) 2025 The ALR Authors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package manager

import (
	"log/slog"
	"sync"
)

// CachedManager оборачивает Manager и кэширует результаты ListAvailable
type CachedManager struct {
	Manager
	cache     map[string][]string
	cacheMu   sync.RWMutex
	hitCount  int
	missCount int
}

// NewCachedManager создаёт новый CachedManager
func NewCachedManager(m Manager) *CachedManager {
	return &CachedManager{
		Manager: m,
		cache:   make(map[string][]string),
	}
}

// ListAvailable возвращает список доступных пакетов с кэшированием
func (c *CachedManager) ListAvailable(prefix string) ([]string, error) {
	c.cacheMu.RLock()
	if result, ok := c.cache[prefix]; ok {
		c.hitCount++
		c.cacheMu.RUnlock()
		slog.Debug("ListAvailable cache hit", "prefix", prefix, "hits", c.hitCount)
		return result, nil
	}
	c.cacheMu.RUnlock()

	// Кэш промах - вызываем реальный метод
	result, err := c.Manager.ListAvailable(prefix)
	if err != nil {
		return nil, err
	}

	c.cacheMu.Lock()
	c.cache[prefix] = result
	c.missCount++
	c.cacheMu.Unlock()

	slog.Debug("ListAvailable cache miss", "prefix", prefix, "misses", c.missCount, "result_count", len(result))

	return result, nil
}

// GetCacheStats возвращает статистику кэша
func (c *CachedManager) GetCacheStats() (hits, misses int) {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()
	return c.hitCount, c.missCount
}
