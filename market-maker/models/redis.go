package model

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var rdClient *redis.Client

func init() {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		log.Errorf(log.Global, "error when connect to redis: %+v", err)
	}

	rdClient = client
}
