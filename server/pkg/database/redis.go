package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"devops-manager/server/pkg/config"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

// InitRedis 初始化 Redis 连接
func InitRedis(cfg *config.RedisConfig) error {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:            fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:        cfg.Password,
		DB:              cfg.DB,
		PoolSize:        10,              // 连接池大小
		MinIdleConns:    5,               // 最小空闲连接数
		MaxIdleConns:    10,              // 最大空闲连接数
		ConnMaxIdleTime: time.Minute * 5, // 连接最大空闲时间
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := RedisClient.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Printf("Connected to Redis at %s:%d", cfg.Host, cfg.Port)
	return nil
}

// GetRedis 获取 Redis 客户端
func GetRedis() *redis.Client {
	return RedisClient
}

// CloseRedis 关闭 Redis 连接
func CloseRedis() error {
	if RedisClient != nil {
		return RedisClient.Close()
	}
	return nil
}
