package cache

import (
	"context"
	"encoding/json"
	"time"
	"sync"

	"github.com/go-redis/redis/v8"
)

// RedisCache Redis缓存实现
type RedisCache struct {
	client *redis.Client
	ctx    context.Context
	// 添加互斥锁，防止缓存击穿
	mu     sync.RWMutex
	// 缓存空值的过期时间，防止缓存穿透
	emptyExpiration time.Duration
	// 正常缓存的默认过期时间
	defaultExpiration time.Duration
}

// NewRedisCache 创建Redis缓存实例
func NewRedisCache(client *redis.Client) *RedisCache {
	// 检查客户端是否为空
	if client == nil {
		return nil
	}
	
	return &RedisCache{
		client: client,
		ctx:    context.Background(),
		emptyExpiration: time.Minute, // 空值缓存1分钟
		defaultExpiration: time.Hour, // 默认缓存1小时
	}
}

// Get 从缓存获取数据（增加缓存穿透、击穿防护）
func (r *RedisCache) Get(key string, dest interface{}) error {
	// 检查客户端是否为空
	if r.client == nil {
		return redis.Nil
	}
	
	// 使用读锁防止缓存击穿
	r.mu.RLock()
	defer r.mu.RUnlock()

	val, err := r.client.Get(r.ctx, key).Result()
	if err == redis.Nil {
		// 缓存未命中，返回特定错误
		return ErrKeyNotFound
	} else if err != nil {
		// Redis连接或其他错误
		return err
	}

	// 检查是否是空值标记（防止缓存穿透）
	if val == "<empty>" {
		return ErrKeyNotFound // 视为空值，让调用方从数据源获取
	}

	err = json.Unmarshal([]byte(val), dest)
	if err != nil {
		return err
	}
	return nil
}

// Set 设置缓存数据
func (r *RedisCache) Set(key string, value interface{}, expiration time.Duration) error {
	// 检查客户端是否为空
	if r.client == nil {
		return nil
	}
	
	// 使用写锁防止缓存击穿
	r.mu.Lock()
	defer r.mu.Unlock()

	var data []byte
	var err error
	
	// 检查值是否为空，防止缓存穿透
	if value == nil {
		data = []byte("<empty>")
		expiration = r.emptyExpiration // 空值使用较短的过期时间
	} else {
		data, err = json.Marshal(value)
		if err != nil {
			return err
		}
		// 如果未指定过期时间，使用默认过期时间
		if expiration <= 0 {
			expiration = r.defaultExpiration
		}
	}

	return r.client.Set(r.ctx, key, data, expiration).Err()
}

// Delete 删除缓存数据
func (r *RedisCache) Delete(key string) error {
	// 检查客户端是否为空
	if r.client == nil {
		return nil
	}
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	return r.client.Del(r.ctx, key).Err()
}

// Exists 检查键是否存在
func (r *RedisCache) Exists(key string) (bool, error) {
	// 检查客户端是否为空
	if r.client == nil {
		return false, nil
	}
	
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	exists, err := r.client.Exists(r.ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}