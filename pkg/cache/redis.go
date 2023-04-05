/*
 * @Author: cloudyi.li
 * @Date: 2023-04-04 20:30:36
 * @LastEditTime: 2023-04-05 15:54:11
 * @LastEditors: cloudyi.li
 * @FilePath: /chatserver-api/pkg/cache/redis.go
 */
package cache

import (
	"chatserver-api/pkg/config"
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

var redisClient *redis.Client

// InitRedis 初始化redisClient
func InitRedis(redisCfg config.RedisConfig) {
	redisClient = redis.NewClient(&redis.Options{
		DB:           redisCfg.Db,
		Addr:         redisCfg.Addr,
		Password:     redisCfg.Password,
		PoolSize:     redisCfg.PoolSize,
		MinIdleConns: redisCfg.MinIdleConns,
		IdleTimeout:  time.Duration(redisCfg.IdleTimeout) * time.Second,
	})
	_, err := redisClient.Ping(context.TODO()).Result()
	if err != nil {
		panic(err)
	}
}

// GetRedisClient 获取redis client
func GetRedisClient() *redis.Client {
	if nil == redisClient {
		panic("Please initialize the Redis client first!")
	}
	return redisClient
}

// CloseRedis 关闭redis client
func CloseRedis() {
	if nil != redisClient {
		_ = redisClient.Close()
	}
}
