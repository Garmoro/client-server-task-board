package redisrepo

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "strconv"
    "time"

    "github.com/Garmoro/conversation-summarizer/internal/model"
    "github.com/redis/go-redis/v9"
)

var ErrCacheMiss = errors.New("redis cache miss")

type Cache struct { client *redis.Client; prefix string; ttl time.Duration }

func New(addr, password string, database int, prefix string, ttl time.Duration) *Cache {
    return &Cache{client: redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: database}), prefix: prefix, ttl: ttl}
}
func (c *Cache) key(id int64) string { return c.prefix + strconv.FormatInt(id, 10) }
func (c *Cache) Ping(ctx context.Context) error { return c.client.Ping(ctx).Err() }

func (c *Cache) Get(ctx context.Context, id int64) (model.Summary, error) {
    value, err := c.client.Get(ctx, c.key(id)).Result()
    if errors.Is(err, redis.Nil) { return model.Summary{}, ErrCacheMiss }
    if err != nil { return model.Summary{}, fmt.Errorf("redis get: %w", err) }
    var summary model.Summary
    if err := json.Unmarshal([]byte(value), &summary); err != nil { return model.Summary{}, fmt.Errorf("decode cached summary: %w", err) }
    return summary, nil
}

func (c *Cache) Set(ctx context.Context, summary model.Summary) error {
    value, err := json.Marshal(summary)
    if err != nil { return fmt.Errorf("encode cached summary: %w", err) }
    if err := c.client.Set(ctx, c.key(summary.ConversationID), value, c.ttl).Err(); err != nil { return fmt.Errorf("redis set: %w", err) }
    return nil
}
func (c *Cache) Close() error { return c.client.Close() }
