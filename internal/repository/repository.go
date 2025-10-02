package repository

import (
	"database/sql"
	"fmt"
	"time"

	"messaging/internal/models"
	"messaging/internal/service"

	_ "github.com/lib/pq"
)

type PostgresRepo struct {
	db *sql.DB
}

func NewPostgresRepo(connStr string) (*PostgresRepo, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS messages (
		id SERIAL PRIMARY KEY,
		recipient VARCHAR(20) NOT NULL,
		content VARCHAR(160) NOT NULL,
		external_id TEXT,
		sent_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);
	`
	if _, err = db.Exec(createTableQuery); err != nil {
		return nil, fmt.Errorf("failed to ensure messages table exists: %w", err)
	}
	return &PostgresRepo{db: db}, nil
}

var _ service.MessageRepository = (*PostgresRepo)(nil)

func (r *PostgresRepo) GetUnsentMessages(limit int) ([]models.Message, error) {
	query := `SELECT id, recipient, content FROM messages
	          WHERE sent_at IS NULL
	          ORDER BY id ASC
	          LIMIT $1;`
	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []models.Message
	for rows.Next() {
		var msg models.Message
		var recipient, content string
		if err := rows.Scan(&msg.ID, &recipient, &content); err != nil {
			return nil, err
		}
		msg.To = recipient
		msg.Content = content
		msg.SentAt = nil
		msg.ExternalID = nil
		results = append(results, msg)
	}
	return results, rows.Err()
}

func (r *PostgresRepo) MarkMessageSent(id int, externalID string, sentAt time.Time) error {
	query := `UPDATE messages
	          SET sent_at=$2, external_id=$3, updated_at=$2
	          WHERE id=$1;`
	_, err := r.db.Exec(query, id, sentAt, externalID)
	return err
}

func (r *PostgresRepo) ListSentMessages() ([]models.Message, error) {
	query := `SELECT id, recipient, content, external_id, sent_at, created_at
	          FROM messages
	          WHERE sent_at IS NOT NULL
	          ORDER BY sent_at ASC;`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var results []models.Message
	for rows.Next() {
		var msg models.Message
		var recipient, content string
		var extID sql.NullString
		var sentTime sql.NullTime
		var created time.Time

		if err := rows.Scan(&msg.ID, &recipient, &content, &extID, &sentTime, &created); err != nil {
			return nil, err
		}

		msg.To = recipient
		msg.Content = content
		if extID.Valid {
			extVal := extID.String
			extPtr := new(string)
			*extPtr = extVal
			msg.ExternalID = extPtr
		} else {
			msg.ExternalID = nil
		}
		if sentTime.Valid {
			tVal := sentTime.Time
			tPtr := new(time.Time)
			*tPtr = tVal
			msg.SentAt = tPtr
		} else {
			msg.SentAt = nil
		}
		msg.CreatedAt = created
		if sentTime.Valid {
			msg.UpdatedAt = sentTime.Time
		} else {
			msg.UpdatedAt = created
		}
		results = append(results, msg)
	}
	return results, rows.Err()
}
