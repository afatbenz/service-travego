package waai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ConversationMessage represents a message in the conversation history
type ConversationMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

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

// GetSessionKey returns the Redis key for a phone session
func GetSessionKey(phone string) string {
	return fmt.Sprintf("waai:session:%s", phone)
}

// LoadSession retrieves conversation history for a phone number
func (sm *SessionManager) LoadSession(ctx context.Context, phone string) ([]ConversationMessage, error) {
	key := GetSessionKey(phone)

	val, err := sm.client.Get(ctx, key).Result()
	if err == redis.Nil {
		// Session doesn't exist yet
		return []ConversationMessage{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis error: %w", err)
	}

	var messages []ConversationMessage
	err = json.Unmarshal([]byte(val), &messages)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal messages: %w", err)
	}

	return messages, nil
}

// SaveSession stores conversation history for a phone number
// TTL is set to 24 hours and refreshed on each save
func (sm *SessionManager) SaveSession(ctx context.Context, phone string, messages []ConversationMessage) error {
	key := GetSessionKey(phone)

	data, err := json.Marshal(messages)
	if err != nil {
		return fmt.Errorf("failed to marshal messages: %w", err)
	}

	err = sm.client.Set(ctx, key, data, 24*time.Hour).Err()
	if err != nil {
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
