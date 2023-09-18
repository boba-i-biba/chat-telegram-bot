package main

import (
	"context"
	"github.com/redis/go-redis/v9"
)

type PubSub interface {
	Publish(ctx context.Context, channel string, message string) error
	Subscribe(ctx context.Context, channels ...string) <-chan string
}

type RedisPubSub struct {
	instance *redis.Client
}

func NewRedisPubSub(url string) (*RedisPubSub, error) {
	redisOptions, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(redisOptions)

	return &RedisPubSub{
		instance: client,
	}, nil
}

func (ps *RedisPubSub) Publish(ctx context.Context, channel string, message string) error {
	err := ps.instance.Publish(ctx,
		channel,
		message,
	).Err()
	if err != nil {
		return err
	}
	return nil
}

func (ps *RedisPubSub) Subscribe(ctx context.Context, channels ...string) <-chan string {
	stringCh := make(chan string)

	go func() {
		originalCh := ps.instance.Subscribe(ctx, channels...).Channel()
		for msg := range originalCh {
			str := msg.Payload
			stringCh <- str
		}
		close(stringCh)
	}()

	return stringCh
}
