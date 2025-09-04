package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Drolfothesgnir/shitposter/util"
	"github.com/redis/go-redis/v9"
)

// Different key prefixes for different use cases
const (
	PendingRegistrationPrefix = "pending_reg:"
	CachePrefix               = "cache:"
	SessionPrefix             = "session:"
)

type Store struct {
	client *redis.Client
}

func NewStore(config *util.Config) *Store {
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddress, //  default "localhost:6379"
		Password: "",                  // "" for no password, ok for now
		DB:       0,                   // 0 for default database
	})

	return &Store{client: rdb}
}

// Function to save user data between requests
// while his device solves the challenge
// during registration
func (store *Store) SaveUserRegSession(
	ctx context.Context,
	sessionID string,
	data PendingRegistration,
	ttl time.Duration,
) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to serialize registration data: %w", err)
	}

	key := PendingRegistrationPrefix + sessionID
	return store.client.Set(ctx, key, jsonData, ttl).Err()
}
