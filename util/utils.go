package util

import "github.com/go-redis/redis/v8"

var (
	Rdc = &redis.Client{}
)

func InitRedisClient() {
	Rdc = redis.NewClient(&redis.Options{
		Addr:     "10.10.10.10:9898",
		Password: "secret",
		DB:       0,
	})
}
