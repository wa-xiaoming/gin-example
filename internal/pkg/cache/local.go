package cache

import (
	"container/list"
	"encoding/json"
	"sync"
	"time"
)

// LocalCache 本地内存缓存实现
type LocalCache struct {
	capacity  int                      // 缓存容量
	ttl       time.Duration            // 过期时间
	mu        sync.RWMutex             // 读写锁
	cache     map[string]*list.Element // 缓存映射
	evictList *list.List               // LRU链表
}

// cacheEntry 缓存条目
type cacheEntry struct {
	key       string
	value     interface{}
	expiresAt time.Time
}

// NewLocalCache 创建本地缓存实例
type NotFoundError struct {
	message string
}

func (e *NotFoundError) Error() string {
	return e.message
}

// ErrKeyNotFound 键不存在错误
var ErrKeyNotFound = &NotFoundError{"key not found"}

// NewLocalCache 创建本地缓存实例
func NewLocalCache(capacity int, ttl time.Duration) *LocalCache {
	return &LocalCache{
		capacity:  capacity,
		ttl:       ttl,
		cache:     make(map[string]*list.Element),
		evictList: list.New(),
	}
}

// Get 从缓存获取数据
func (l *LocalCache) Get(key string, dest interface{}) error {
	l.mu.RLock()
	defer l.mu.RUnlock()

	element, exists := l.cache[key]
	if !exists {
		return ErrKeyNotFound
	}

	entry := element.Value.(*cacheEntry)
	
	// 检查是否过期
	if time.Now().After(entry.expiresAt) {
		// 过期则删除
		l.mu.RUnlock()
		l.mu.Lock()
		l.removeElement(element)
		l.mu.Unlock()
		l.mu.RLock()
		return ErrKeyNotFound
	}

	// 移动到链表前端（更新LRU顺序）
	l.evictList.MoveToFront(element)

	// 反序列化数据
	data, err := json.Marshal(entry.value)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, dest)
}

// Set 设置缓存数据
func (l *LocalCache) Set(key string, value interface{}, expiration time.Duration) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 如果键已存在，更新值
	if element, exists := l.cache[key]; exists {
		l.evictList.MoveToFront(element)
		entry := element.Value.(*cacheEntry)
		entry.value = value
		entry.expiresAt = time.Now().Add(expiration)
		return nil
	}

	// 检查是否需要驱逐
	if l.evictList.Len() >= l.capacity {
		l.evict()
	}

	// 添加新条目
	entry := &cacheEntry{
		key:       key,
		value:     value,
		expiresAt: time.Now().Add(expiration),
	}

	element := l.evictList.PushFront(entry)
	l.cache[key] = element

	return nil
}

// Delete 删除缓存数据
func (l *LocalCache) Delete(key string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if element, exists := l.cache[key]; exists {
		l.removeElement(element)
	}

	return nil
}

// Exists 检查键是否存在
func (l *LocalCache) Exists(key string) (bool, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	element, exists := l.cache[key]
	if !exists {
		return false, nil
	}

	entry := element.Value.(*cacheEntry)
	
	// 检查是否过期
	if time.Now().After(entry.expiresAt) {
		// 过期则删除
		l.mu.RUnlock()
		l.mu.Lock()
		l.removeElement(element)
		l.mu.Unlock()
		l.mu.RLock()
		return false, nil
	}

	return true, nil
}

// evict 驱逐最久未使用的条目
func (l *LocalCache) evict() {
	element := l.evictList.Back()
	if element != nil {
		l.removeElement(element)
	}
}

// removeElement 删除指定元素
func (l *LocalCache) removeElement(element *list.Element) {
	l.evictList.Remove(element)
	entry := element.Value.(*cacheEntry)
	delete(l.cache, entry.key)
}