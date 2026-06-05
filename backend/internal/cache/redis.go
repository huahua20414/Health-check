package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

func Open(addr string) *redis.Client {
	return redis.NewClient(&redis.Options{Addr: addr})
}

func Ping(ctx context.Context, client *redis.Client) error {
	var err error
	for i := 0; i < 30; i++ {
		err = client.Ping(ctx).Err()
		if err == nil {
			return nil
		}
		time.Sleep(time.Second)
	}
	return err
}
