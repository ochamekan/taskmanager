package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sony/gobreaker"
	"go.uber.org/zap"

	"github.com/example/taskmanager/internal/usecase"
)

const redisTimeout = 500 * time.Millisecond

func Connect(addr, password string, db int) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  time.Second,
		ReadTimeout:  redisTimeout,
		WriteTimeout: redisTimeout,
	})
}

type Cache struct {
	client  *redis.Client
	ttl     time.Duration
	breaker *gobreaker.CircuitBreaker
	log     *zap.Logger
}

func New(client *redis.Client, ttl time.Duration, log *zap.Logger) *Cache {
	breaker := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:    "redis",
		Timeout: 30 * time.Second,
		OnStateChange: func(_ string, from, to gobreaker.State) {
			log.Warn("redis circuit breaker state changed", zap.Stringer("from", from), zap.Stringer("to", to))
		},
	})
	return &Cache{client: client, ttl: ttl, breaker: breaker, log: log}
}

func (c *Cache) TeamVersion(ctx context.Context, teamID int64) (string, bool) {
	version, err := c.breaker.Execute(func() (any, error) {
		version, err := c.client.Get(ctx, teamVersionKey(teamID)).Result()
		if errors.Is(err, redis.Nil) {
			return "0", nil
		}
		return version, err
	})
	if err != nil {
		c.log.Warn("cache: get team version failed", zap.Error(err))
		return "", false
	}
	return version.(string), true
}

func (c *Cache) Get(ctx context.Context, version string, filter usecase.TaskFilter) (*usecase.TaskPage, bool) {
	data, err := c.breaker.Execute(func() (any, error) {
		data, err := c.client.Get(ctx, pageKey(version, filter)).Bytes()
		if errors.Is(err, redis.Nil) {
			return []byte(nil), nil
		}
		return data, err
	})
	if err != nil {
		c.log.Warn("cache: get page failed", zap.Error(err))
		return nil, false
	}
	if data.([]byte) == nil {
		return nil, false
	}

	var page usecase.TaskPage
	if err := json.Unmarshal(data.([]byte), &page); err != nil {
		return nil, false
	}
	return &page, true
}

func (c *Cache) Set(ctx context.Context, version string, filter usecase.TaskFilter, page *usecase.TaskPage) {
	data, err := json.Marshal(page)
	if err != nil {
		return
	}
	_, err = c.breaker.Execute(func() (any, error) {
		return nil, c.client.Set(ctx, pageKey(version, filter), data, c.ttl).Err()
	})
	if err != nil {
		c.log.Warn("cache: set page failed", zap.Error(err))
	}
}

func (c *Cache) InvalidateTeam(ctx context.Context, teamID int64) {
	if err := c.client.Incr(ctx, teamVersionKey(teamID)).Err(); err != nil {
		c.log.Warn("cache: invalidate team failed", zap.Int64("team_id", teamID), zap.Error(err))
	}
}

func teamVersionKey(teamID int64) string {
	return fmt.Sprintf("tm:tasks:team:%d:version", teamID)
}

func pageKey(version string, f usecase.TaskFilter) string {
	status := ""
	if f.Status != nil {
		status = string(*f.Status)
	}
	assignee := int64(0)
	if f.AssigneeID != nil {
		assignee = *f.AssigneeID
	}
	return fmt.Sprintf("tm:tasks:team:%d:v%s:status=%s:assignee=%d:limit=%d:offset=%d:before=%d",
		f.TeamID, version, status, assignee, f.Limit, f.Offset, f.BeforeID)
}
