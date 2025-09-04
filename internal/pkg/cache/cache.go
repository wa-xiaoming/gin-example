package cache

import "time"

// Cache 缓存接口
type Cache interface {
	// Get 从缓存获取数据
	Get(key string, dest interface{}) error
	
	// Set 设置缓存数据
	Set(key string, value interface{}, expiration time.Duration) error
	
	// Delete 删除缓存数据
	Delete(key string) error
	
	// Exists 检查键是否存在
	Exists(key string) (bool, error)
}