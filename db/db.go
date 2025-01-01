package db

import (
	"context"

	"github.com/redis/go-redis/v9"
)

const redisAddr string = "localhost:6379"
var initValues = map[string]interface{}{
    "confidence": 0,
    "voteCount": 0,
}

func Initialize() (*redis.Client, error) {
    ctx := context.Background()
    r := redis.NewClient(&redis.Options{
        Addr:	  redisAddr,
        Password: "",
        DB:       0,
    })

    if err := r.FlushAll(ctx).Err(); err != nil {
        return nil, err
    }

    err := r.MSet(ctx, initValues).Err(); if err != nil {
        return nil, err
    }

    return r, nil
}


