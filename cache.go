package nie

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
)

var DefaultCache *Cache

var (
	// Deprecated: it will be removed
	CaptchaPrefix = "captcha:" // 验证码前缀
	// Deprecated: it will be removed
	AccessTokenPrefix = "accessToken:" // accessToken前缀
	// Deprecated: it will be removed
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
	// 处理字符串类型（直接存储）
	if str, ok := value.(string); ok {
		return c.redis.Set(ctx, key, str, expiration).Err()
	}
	// 处理基本类型（int, float等）
	if isBasicType(value) {
		return c.redis.Set(ctx, key, value, expiration).Err()
	}
	// 其他类型视为结构体/复杂类型，进行JSON序列化
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.redis.Set(ctx, key, data, expiration).Err()
}

// AsyncSetRedis 异步设置缓存
func (c *Cache) AsyncSetRedis(key string, data interface{}, expiration time.Duration, logHelper *log.Helper) {
	go func() {
		// 使用独立超时上下文，不影响主流程
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := c.SetRedis(ctx, key, data, expiration); err != nil && logHelper != nil {
			logHelper.Errorf("async set redis failed for key %s: %v", key, err)
		}
	}()
}

// GetRedis 获取缓存
func (c *Cache) GetRedis(ctx context.Context, key string, result ...interface{}) (string, error) {
	// 普通获取模式：不传入result则返回原始字符串
	if len(result) == 0 {
		return c.redis.Get(ctx, key).Result()
	}
	// 结构体获取模式：传入result指针则进行反序列化
	data, err := c.redis.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", err
		}
		return "", err
	}
	// 反序列化到目标结构体
	if err := json.Unmarshal(data, result[0]); err != nil {
		return "", err
	}
	return string(data), nil
}

// DelRedis 删除缓存
func (c *Cache) DelRedis(ctx context.Context, key string) error {
	return c.redis.Del(ctx, key).Err()
}

// AsyncDelRedis 异步删除缓存
func (c *Cache) AsyncDelRedis(key string, logHelper *log.Helper) {
	go func() {
		// 使用独立超时上下文，不影响主流程
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := c.DelRedis(ctx, key); err != nil && logHelper != nil {
			logHelper.Errorf("async delete redis failed for key '%s': %v", key, err)
		}
	}()
}

// TTLRefresh 刷新缓存过期时间
func (c *Cache) TTLRefresh(ctx context.Context, key string, expiration time.Duration) error {
	return c.redis.Expire(ctx, key, expiration).Err()
}

// 判断是否为基本数据类型（避免对基本类型进行JSON序列化）
func isBasicType(v interface{}) bool {
	kind := reflect.TypeOf(v).Kind()
	return kind >= reflect.Bool && kind <= reflect.Complex128 ||
		kind == reflect.String
}
