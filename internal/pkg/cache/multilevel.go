package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
)

// MultiLevelCache 多级缓存实现
type MultiLevelCache struct {
	l1Cache     *LocalCache     // L1缓存（本地内存）
	l2Cache     *RedisCache     // L2缓存（Redis）
	ctx         context.Context
}

// NewMultiLevelCache 创建多级缓存实例
func NewMultiLevelCache(redisClient *redis.Client) *MultiLevelCache {
	// 检查Redis客户端是否为空
	if redisClient == nil {
		return nil
	}
	
	redisCache := NewRedisCache(redisClient)
	if redisCache == nil {
		return nil
	}
	
	return &MultiLevelCache{
		l1Cache: NewLocalCache(1000, 5*time.Minute), // L1缓存：最多1000个条目，5分钟过期
		l2Cache: redisCache,
		ctx:     context.Background(),
	}
}

// Get 从缓存获取数据（先查L1，再查L2）
func (m *MultiLevelCache) Get(key string, dest interface{}) error {
	// 检查缓存实例是否为空
	if m.l1Cache == nil || m.l2Cache == nil {
		return errors.New("cache instance is nil")
	}
	
	// 先从L1缓存获取
	if err := m.l1Cache.Get(key, dest); err == nil {
		return nil // L1缓存命中
	}

	// L1未命中，从L2缓存获取
	val, err := m.l2Cache.client.Get(m.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return ErrKeyNotFound
		}
		return err // L2也未命中或出错
	}

	// 检查是否是空值标记（防止缓存穿透）
	if val == "<empty>" {
		// 同时在L1缓存中存储空值标记
		m.l1Cache.Set(key, "<empty>", time.Minute)
		return ErrKeyNotFound
	}

	// 反序列化数据
	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return err
	}

	// L2命中，将数据写入L1缓存
	m.l1Cache.Set(key, val, 1*time.Minute) // L1缓存时间短一些

	return nil
}

// Set 设置缓存数据（同时写入L1和L2）
func (m *MultiLevelCache) Set(key string, value interface{}, expiration time.Duration) error {
	// 检查缓存实例是否为空
	if m.l1Cache == nil || m.l2Cache == nil {
		return errors.New("cache instance is nil")
	}
	
	// 同时写入L1和L2缓存
	// 注意：这里我们不直接返回错误，因为即使一个缓存层失败，另一个可能成功
	err1 := m.l1Cache.Set(key, value, expiration)
	err2 := m.l2Cache.Set(key, value, expiration)
	
	// 如果两个缓存层都失败了，则返回错误
	if err1 != nil && err2 != nil {
		return errors.New("failed to set cache in both L1 and L2")
	}
	
	// 如果其中一个失败，记录警告但不返回错误
	if err1 != nil {
		// 可以考虑添加日志记录
		return nil
	}
	
	if err2 != nil {
		// 可以考虑添加日志记录
		return nil
	}
	
	return nil
}

// Delete 删除缓存数据（同时删除L1和L2）
func (m *MultiLevelCache) Delete(key string) error {
	// 检查缓存实例是否为空
	if m.l1Cache == nil || m.l2Cache == nil {
		return errors.New("cache instance is nil")
	}
	
	// 同时删除L1和L2缓存
	err1 := m.l1Cache.Delete(key)
	err2 := m.l2Cache.Delete(key)
	
	// 如果两个缓存层都失败了，则返回错误
	if err1 != nil && err2 != nil {
		return errors.New("failed to delete cache in both L1 and L2")
	}
	
	// 如果其中一个失败，记录警告但不返回错误
	if err1 != nil {
		// 可以考虑添加日志记录
		return nil
	}
	
	if err2 != nil {
		// 可以考虑添加日志记录
		return nil
	}
	
	return nil
}

// Exists 检查键是否存在（先查L1，再查L2）
func (m *MultiLevelCache) Exists(key string) (bool, error) {
	// 检查缓存实例是否为空
	if m.l1Cache == nil || m.l2Cache == nil {
		return false, errors.New("cache instance is nil")
	}
	
	// 先检查L1缓存
	if exists, err := m.l1Cache.Exists(key); err == nil && exists {
		return true, nil
	}

	// 再检查L2缓存
	return m.l2Cache.Exists(key)
}