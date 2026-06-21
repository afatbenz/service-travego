package waai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// SessionManager handles conversation history storage in Redis
type SessionManager struct {
	client *redis.Client
}

// NewSessionManager creates a new session manager
func NewSessionManager(rdb *redis.Client) *SessionManager {
	return &SessionManager{
		client: rdb,
	}
}

// GetSessionKey returns the Redis key for a phone session (legacy, Skenario 1)
func GetSessionKey(phone string) string {
	return fmt.Sprintf("waai:session:%s", phone)
}

// GetSessionKeyFor returns the Redis key berbasis device/org + customer (Skenario 2)
func GetSessionKeyFor(deviceID, customerPhone string) string {
	return fmt.Sprintf("waai:session:%s:%s", deviceID, customerPhone)
}

// LoadSession retrieves conversation history for a phone number (legacy, Skenario 1)
func (sm *SessionManager) LoadSession(ctx context.Context, phone string) ([]ConversationMessage, error) {
	key := GetSessionKey(phone)

	val, err := sm.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return []ConversationMessage{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis error: %w", err)
	}

	var messages []ConversationMessage
	if err := json.Unmarshal([]byte(val), &messages); err != nil {
		return nil, fmt.Errorf("failed to unmarshal messages: %w", err)
	}

	return messages, nil
}

// SaveSession stores conversation history for a phone number (legacy, Skenario 1)
func (sm *SessionManager) SaveSession(ctx context.Context, phone string, messages []ConversationMessage) error {
	key := GetSessionKey(phone)

	data, err := json.Marshal(messages)
	if err != nil {
		return fmt.Errorf("failed to marshal messages: %w", err)
	}

	if err := sm.client.Set(ctx, key, data, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("redis error: %w", err)
	}

	return nil
}

// LoadSessionFor loads session keyed by (deviceID, customerPhone) untuk Skenario 2
func (sm *SessionManager) LoadSessionFor(ctx context.Context, deviceID, customerPhone string) ([]ConversationMessage, error) {
	key := GetSessionKeyFor(deviceID, customerPhone)

	val, err := sm.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return []ConversationMessage{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis error: %w", err)
	}

	var messages []ConversationMessage
	if err := json.Unmarshal([]byte(val), &messages); err != nil {
		return nil, fmt.Errorf("failed to unmarshal messages: %w", err)
	}

	return messages, nil
}

// SaveSessionFor saves session keyed by (deviceID, customerPhone) untuk Skenario 2
func (sm *SessionManager) SaveSessionFor(ctx context.Context, deviceID, customerPhone string, messages []ConversationMessage) error {
	key := GetSessionKeyFor(deviceID, customerPhone)

	data, err := json.Marshal(messages)
	if err != nil {
		return fmt.Errorf("failed to marshal messages: %w", err)
	}

	if err := sm.client.Set(ctx, key, data, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("redis error: %w", err)
	}

	return nil
}

// ClearSessionFor clears session for a specific device + customer pair (Skenario 2)
func (sm *SessionManager) ClearSessionFor(ctx context.Context, deviceID, customerPhone string) error {
	key := GetSessionKeyFor(deviceID, customerPhone)
	if err := sm.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis error: %w", err)
	}
	return nil
}

// ClearSession removes conversation history for a phone number
func (sm *SessionManager) ClearSession(ctx context.Context, phone string) error {
	key := GetSessionKey(phone)
	err := sm.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("redis error: %w", err)
	}
	return nil
}

// AppendMessage adds a message to the conversation history and saves it
func (sm *SessionManager) AppendMessage(ctx context.Context, phone string, message ConversationMessage) error {
	messages, err := sm.LoadSession(ctx, phone)
	if err != nil {
		return err
	}

	messages = append(messages, message)
	return sm.SaveSession(ctx, phone, messages)
}
