package service

import (
	"database/sql"
	"fmt"
	"net/http"
	"service-travego/database"
	"service-travego/helper"
	"strings"
	"time"
)

type NotificationPayload struct {
	Title   string `json:"title"`
	Message string `json:"message"`
	URL     string `json:"url"`
}

type NotificationItem struct {
	NotificationID string    `json:"notification_id"`
	ReferenceURL   string    `json:"reference_url"`
	Title          string    `json:"title"`
	Message        string    `json:"message"`
	CreatedAt      time.Time `json:"created_at"`
	IsRead         bool      `json:"is_read"`
}

type NotificationService struct {
	db     *sql.DB
	driver string
}

func NewNotificationService(db *sql.DB, driver string) *NotificationService {
	return &NotificationService{
		db:     db,
		driver: driver,
	}
}

func (s *NotificationService) CreateNotification(orgID string, payload NotificationPayload) (string, error) {
	orgID = strings.TrimSpace(orgID)
	payload.Title = strings.TrimSpace(payload.Title)
	payload.Message = strings.TrimSpace(payload.Message)
	payload.URL = strings.TrimSpace(payload.URL)

	if orgID == "" {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "missing organization context")
	}
	if payload.Title == "" || payload.Message == "" {
		return "", NewServiceError(ErrInvalidInput, http.StatusBadRequest, "title and message are required")
	}

	notificationID := helper.GenerateUUID()
	query := fmt.Sprintf(
		`INSERT INTO notifications
			(notification_id, organization_id, reference_url, title, message, created_at, is_read)
		 VALUES
			(%s, %s, %s, %s, %s, %s, %s)`,
		s.getPlaceholder(1),
		s.getPlaceholder(2),
		s.getPlaceholder(3),
		s.getPlaceholder(4),
		s.getPlaceholder(5),
		s.getPlaceholder(6),
		s.getPlaceholder(7),
	)

	if _, err := database.Exec(
		s.db,
		query,
		notificationID,
		orgID,
		payload.URL,
		payload.Title,
		payload.Message,
		time.Now(),
		false,
	); err != nil {
		return "", err
	}

	return notificationID, nil
}

func (s *NotificationService) GetNotifications(orgID string) ([]NotificationItem, error) {
	orgID = strings.TrimSpace(orgID)
	if orgID == "" {
		return nil, NewServiceError(ErrInvalidInput, http.StatusBadRequest, "missing organization context")
	}

	query := fmt.Sprintf(
		`SELECT notification_id, reference_url, title, message, created_at, is_read
		 FROM notifications
		 WHERE organization_id = %s
		 ORDER BY created_at DESC
		 LIMIT 50`,
		s.getPlaceholder(1),
	)

	rows, err := database.Query(s.db, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]NotificationItem, 0)
	for rows.Next() {
		var item NotificationItem
		if err := rows.Scan(
			&item.NotificationID,
			&item.ReferenceURL,
			&item.Title,
			&item.Message,
			&item.CreatedAt,
			&item.IsRead,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (s *NotificationService) MarkAsRead(orgID, notificationID string) error {
	orgID = strings.TrimSpace(orgID)
	notificationID = strings.TrimSpace(notificationID)

	if orgID == "" {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "missing organization context")
	}
	if notificationID == "" {
		return NewServiceError(ErrInvalidInput, http.StatusBadRequest, "notification_id is required")
	}

	query := fmt.Sprintf(
		`UPDATE notifications
		 SET is_read = %s
		 WHERE notification_id = %s AND organization_id = %s`,
		s.getPlaceholder(1),
		s.getPlaceholder(2),
		s.getPlaceholder(3),
	)

	result, err := database.Exec(s.db, query, true, notificationID, orgID)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return NewServiceError(ErrNotFound, http.StatusNotFound, "notification not found")
	}

	return nil
}

func (s *NotificationService) getPlaceholder(pos int) string {
	if s.driver == "postgres" || s.driver == "pgx" {
		return fmt.Sprintf("$%d", pos)
	}
	return "?"
}
