package service

import (
	"database/sql"
	"net/http"
	"service-travego/configs"
	"service-travego/helper"
	"service-travego/model"
	"service-travego/repository"
	"strings"
)

type MessagesService struct {
	repo *repository.MessagesRepository
}

func NewMessagesService(repo *repository.MessagesRepository) *MessagesService {
	return &MessagesService{repo: repo}
}

func (s *MessagesService) SubmitMessage(orgID string, req *model.MessageSubmitRequest) (string, error) {
	if orgID == "" {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "missing organization context")
	}
	if req == nil {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "invalid payload")
	}

	req.CustomerEmail = strings.TrimSpace(req.CustomerEmail)
	req.CustomerName = strings.TrimSpace(req.CustomerName)
	req.CustomerPhone = strings.TrimSpace(req.CustomerPhone)
	req.Message = strings.TrimSpace(req.Message)
	req.MessageType = strings.TrimSpace(req.MessageType)

	if req.CustomerEmail == "" || req.CustomerName == "" || req.Message == "" || req.MessageType == "" {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "customer_email, customer_name, message, message_type is required")
	}

	messageID := helper.GenerateUUID()
	if err := s.repo.CreateMessage(orgID, messageID, req); err != nil {
		return "", err
	}
	return messageID, nil
}

func (s *MessagesService) ListMessages(orgID string) ([]model.MessageListItem, error) {
	if orgID == "" {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "missing organization context")
	}
	items, err := s.repo.ListMessages(orgID)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].MessageTypeLabel = configs.MessageTypeLabel[items[i].MessageType]
	}
	return items, nil
}

func (s *MessagesService) ReadMessage(orgID, messageID string) error {
	if orgID == "" {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "missing organization context")
	}
	if strings.TrimSpace(messageID) == "" {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "message_id is required")
	}
	if err := s.repo.MarkMessageRead(orgID, messageID); err != nil {
		if err == sql.ErrNoRows {
			return NewServiceError(ErrNotFound, http.StatusNotFound, "message not found")
		}
		return err
	}
	return nil
}
