package nie

import (
	"context"
	"github.com/redis/go-redis/v9"
	"time"
)

var DefaultCache *Cache

var (
	CaptchaPrefix      = "captcha:"      // 验证码前缀
	AccessTokenPrefix  = "accessToken:"  // accessToken前缀
	RefreshTokenPrefix = "refreshToken:" // refreshToken前缀
)

// Cache 用于管理 Redis 缓存
type Cache struct {
	redis redis.Cmdable
}

// NewCache 使用已有的 Redis 客户端初始化 Cache
func NewCache(redis redis.Cmdable) *Cache {
	return &Cache{redis: redis}
}

// InitCache 初始化全局 DefaultCache
func InitCache(client redis.Cmdable) {
	DefaultCache = NewCache(client)
}

// SetRedis 设置缓存
func (c *Cache) SetRedis(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.redis.Set(ctx, key, value, expiration).Err()
}

// GetRedis 获取缓存
func (c *Cache) GetRedis(ctx context.Context, key string) (string, error) {
	return c.redis.Get(ctx, key).Result()
}

// DelRedis 删除缓存
func (c *Cache) DelRedis(ctx context.Context, key string) error {
	return c.redis.Del(ctx, key).Err()
}
