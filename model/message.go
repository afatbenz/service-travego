package model

import "time"

type MessageSubmitRequest struct {
	CustomerEmail string `json:"customer_email"`
	CustomerName  string `json:"customer_name"`
	CustomerPhone string `json:"customer_phone"`
	Message       string `json:"message"`
	MessageType   string `json:"message_type"`
}

type MessageReadRequest struct {
	MessageID string `json:"message_id"`
}

type MessageListItem struct {
	MessageID        string    `json:"message_id"`
	CustomerName     string    `json:"customer_name"`
	CustomerEmail    string    `json:"customer_email"`
	CustomerPhone    string    `json:"customer_phone"`
	MessageType      string    `json:"message_type"`
	MessageTypeLabel string    `json:"message_type_label"`
	Message          string    `json:"message"`
	Status           int       `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
}
