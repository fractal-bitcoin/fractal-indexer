package loader

import (
	"context"
	"fractal-indexer/constant"
	"strconv"

	redis "github.com/go-redis/redis/v8"
)

// GetInfoHeight reads a uint32 height from pika info hash. Returns 0 on missing key.
func GetInfoHeight(client redis.UniversalClient, field string) (uint32, error) {
	ctx := context.Background()
	val, err := client.HGet(ctx, constant.TASK_INFO_KEYNAME, field).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	h, err := strconv.ParseUint(val, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(h), nil
}
