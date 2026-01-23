package lock

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// Manager gestisce l'acquisizione e il rilascio di lock distribuiti.
type Manager interface {
	Acquire(ctx context.Context, key string) (token string, ok bool, err error)
	Release(ctx context.Context, key, token string) error
}

// RedisLock implementa un lock distribuito basato su Redis.
type RedisLock struct {
	client  *redis.Client
	ttl     time.Duration
	retries int
	backoff time.Duration
}

func NewRedisLock(client *redis.Client, ttl time.Duration, retries int, backoff time.Duration) *RedisLock {
	// TTL breve evita lock orfani in caso di crash.
	return &RedisLock{
		client:  client,
		ttl:     ttl,
		retries: retries,
		backoff: backoff,
	}
}

func (l *RedisLock) Acquire(ctx context.Context, key string) (string, bool, error) {
	token := newToken()
	for attempt := 0; attempt <= l.retries; attempt++ {
		ok, err := l.client.SetNX(ctx, key, token, l.ttl).Result()
		if err != nil {
			return "", false, err
		}
		if ok {
			return token, true, nil
		}
		if attempt < l.retries {
			time.Sleep(l.backoff)
		}
	}
	return "", false, nil
}

func (l *RedisLock) Release(ctx context.Context, key, token string) error {
	if key == "" || token == "" {
		return errors.New("key e token sono richiesti")
	}
	return releaseLua.Run(ctx, l.client, []string{key}, token).Err()
}

var releaseLua = redis.NewScript(`
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("del", KEYS[1])
end
return 0
`)
