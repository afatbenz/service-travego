package repository

import (
	"database/sql"
	"fmt"
	"service-travego/database"
	"service-travego/model"
	"time"
)

type MessagesRepository struct {
	db     *sql.DB
	driver string
}

func NewMessagesRepository(db *sql.DB, driver string) *MessagesRepository {
	return &MessagesRepository{db: db, driver: driver}
}

func (r *MessagesRepository) ListMessages(orgID string) ([]model.MessageListItem, error) {
	query := fmt.Sprintf(
		`SELECT message_id, customer_name, customer_email, customer_phone, message_type, message, status, created_at
		 FROM messages
		 WHERE organization_id = %s
		 ORDER BY created_at DESC`,
		r.getPlaceholder(1),
	)

	rows, err := database.Query(r.db, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.MessageListItem, 0)
	for rows.Next() {
		var it model.MessageListItem
		if err := rows.Scan(
			&it.MessageID,
			&it.CustomerName,
			&it.CustomerEmail,
			&it.CustomerPhone,
			&it.MessageType,
			&it.Message,
			&it.Status,
			&it.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, nil
}

func (r *MessagesRepository) CreateMessage(orgID, messageID string, req *model.MessageSubmitRequest) error {
	query := fmt.Sprintf(
		`INSERT INTO messages
			(message_id, organization_id, customer_name, customer_email, customer_phone, message_type, message, status, created_at)
		 VALUES
			(%s, %s, %s, %s, %s, %s, %s, %s, %s)`,
		r.getPlaceholder(1),
		r.getPlaceholder(2),
		r.getPlaceholder(3),
		r.getPlaceholder(4),
		r.getPlaceholder(5),
		r.getPlaceholder(6),
		r.getPlaceholder(7),
		r.getPlaceholder(8),
		r.getPlaceholder(9),
	)

	_, err := database.Exec(
		r.db,
		query,
		messageID,
		orgID,
		req.CustomerName,
		req.CustomerEmail,
		req.CustomerPhone,
		req.MessageType,
		req.Message,
		0,
		time.Now(),
	)
	return err
}

func (r *MessagesRepository) MarkMessageRead(orgID, messageID string) error {
	query := fmt.Sprintf(
		`UPDATE messages
		 SET status = %s, updated_at = %s
		 WHERE message_id = %s AND organization_id = %s`,
		r.getPlaceholder(1),
		r.getPlaceholder(2),
		r.getPlaceholder(3),
		r.getPlaceholder(4),
	)

	res, err := database.Exec(r.db, query, 1, time.Now(), messageID, orgID)
	if err != nil {
		return err
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if ra == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *MessagesRepository) getPlaceholder(pos int) string {
	if r.driver == "postgres" || r.driver == "pgx" {
		return fmt.Sprintf("$%d", pos)
	}
	return "?"
}

