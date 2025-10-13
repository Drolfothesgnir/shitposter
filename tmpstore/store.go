package tmpstore

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Drolfothesgnir/shitposter/util"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/redis/go-redis/v9"
)

// Different key prefixes for different use cases
const (
	PendingRegistrationPrefix   = "pending_reg:"
	PendingAuthenticationPrefix = "pending_auth:"
	CachePrefix                 = "cache:"
	SessionPrefix               = "session:"
)

// Data stored in memory during registration
type PendingRegistration struct {
	Email              string                `json:"email"`
	Username           string                `json:"username"`
	WebauthnUserHandle []byte                `json:"webauthn_user_handle"`
	SessionData        *webauthn.SessionData `json:"session_data"`
	ExpiresAt          time.Time             `json:"expires_at"`
}

type PendingAuthentication struct {
	UserID      int64                 `json:"user_id"`
	Username    string                `json:"username"`
	SessionData *webauthn.SessionData `json:"session_data"`
	ExpiresAt   time.Time             `json:"expires_at"`
}

type Store interface {
	SaveUserRegSession(ctx context.Context, sessionID string, data PendingRegistration, ttl time.Duration) error
	GetUserRegSession(ctx context.Context, sessionID string) (*PendingRegistration, error)
	DeleteUserRegSession(ctx context.Context, sessionID string) error
	SaveUserAuthSession(ctx context.Context, sessionID string, data PendingAuthentication, ttl time.Duration) error
	GetUserAuthSession(ctx context.Context, sessionID string) (*PendingAuthentication, error)
	DeleteUserAuthSession(ctx context.Context, sessionID string) error
}

type RedisStore struct {
	client *redis.Client
}

func NewStore(config *util.Config) Store {
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddress, //  default "localhost:6379"
		Password: "",                  // "" for no password, ok for now
		DB:       0,                   // 0 for default database
	})

	return &RedisStore{client: rdb}
}

// Function to save user data between requests
// while his device solves the challenge
// during registration
func (store *RedisStore) SaveUserRegSession(
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

// Function to retrieve user data pending registration session.
// Returns error if not found or expired.
func (store *RedisStore) GetUserRegSession(ctx context.Context, sessionID string) (*PendingRegistration, error) {
	key := PendingRegistrationPrefix + sessionID

	jsonData, err := store.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("registration session not found or expired")
		}
		return nil, fmt.Errorf("failed to get registration session: %w", err)
	}

	var session PendingRegistration
	if err := json.Unmarshal([]byte(jsonData), &session); err != nil {
		return nil, fmt.Errorf("failed to parse registration session json: %w", err)
	}

	return &session, nil
}

// Helper function to clean temporary user data from Redis.
func (store *RedisStore) DeleteUserRegSession(ctx context.Context, sessionID string) error {
	key := PendingRegistrationPrefix + sessionID
	return store.client.Del(ctx, key).Err()
}

// When user tries to authenticate their session must be stored between requests.
// This function does this. Same thing as with registration.
func (store *RedisStore) SaveUserAuthSession(
	ctx context.Context,
	sessionID string,
	data PendingAuthentication,
	ttl time.Duration,
) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to serialize authentication data: %w", err)
	}

	key := PendingAuthenticationPrefix + sessionID
	return store.client.Set(ctx, key, jsonData, ttl).Err()
}

// Function to retrieve user data pending authentication session.
// Returns error if not found or expired.
func (store *RedisStore) GetUserAuthSession(ctx context.Context, sessionID string) (*PendingAuthentication, error) {
	key := PendingAuthenticationPrefix + sessionID

	jsonData, err := store.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("authentication session not found or expired")
		}
		return nil, fmt.Errorf("failed to get authentication session: %w", err)
	}

	var session PendingAuthentication
	if err := json.Unmarshal([]byte(jsonData), &session); err != nil {
		return nil, fmt.Errorf("failed to parse authentication session json: %w", err)
	}

	return &session, nil
}

// Helper function to clean temporary user data from Redis.
func (store *RedisStore) DeleteUserAuthSession(ctx context.Context, sessionID string) error {
	key := PendingAuthenticationPrefix + sessionID
	return store.client.Del(ctx, key).Err()
}
