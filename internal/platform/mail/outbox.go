package mail

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type sender interface {
	Send(to, subject, body string) error
}

type outboxMessage struct {
	id      string
	to      string
	subject string
	body    string
	attempt int
}

func RunOutbox(ctx context.Context, pool *pgxpool.Pool, delivery sender, logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		if err := deliverOutboxBatch(ctx, pool, delivery, logger); err != nil && ctx.Err() == nil {
			logger.ErrorContext(ctx, "process email outbox batch", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func deliverOutboxBatch(ctx context.Context, pool *pgxpool.Pool, delivery sender, logger *slog.Logger) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin outbox claim: %w", err)
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `
		SELECT id::text,recipient,subject,body,attempts
		FROM mail_outbox
		WHERE sent_at IS NULL AND failed_at IS NULL AND next_attempt_at <= now()
		ORDER BY created_at,id
		LIMIT 20
		FOR UPDATE SKIP LOCKED`)
	if err != nil {
		return fmt.Errorf("claim pending emails: %w", err)
	}
	messages := make([]outboxMessage, 0)
	for rows.Next() {
		var message outboxMessage
		if err := rows.Scan(&message.id, &message.to, &message.subject, &message.body, &message.attempt); err != nil {
			rows.Close()
			return fmt.Errorf("scan pending email: %w", err)
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate pending emails: %w", err)
	}
	rows.Close()
	if len(messages) == 0 {
		return tx.Commit(ctx)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE mail_outbox SET next_attempt_at=now()+interval '10 minutes'
		WHERE id = ANY($1::uuid[])`, messageIDs(messages)); err != nil {
		return fmt.Errorf("reserve pending emails: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit outbox claim: %w", err)
	}
	for _, message := range messages {
		if err := delivery.Send(message.to, message.subject, message.body); err != nil {
			logger.ErrorContext(ctx, "deliver queued email", "message_id", message.id, "error", err)
			if err := retryOutboxMessage(ctx, pool, message, err); err != nil {
				logger.ErrorContext(ctx, "schedule email retry", "message_id", message.id, "error", err)
			}
			continue
		}
		if _, err := pool.Exec(ctx, `
			UPDATE mail_outbox
			SET sent_at=now(),last_error=NULL,recipient='',subject='',body=''
			WHERE id=$1 AND sent_at IS NULL`, message.id); err != nil {
			logger.ErrorContext(ctx, "mark email delivered", "message_id", message.id, "error", err)
		}
	}
	return nil
}

func retryOutboxMessage(ctx context.Context, pool *pgxpool.Pool, message outboxMessage, deliveryErr error) error {
	nextAttempt := message.attempt + 1
	delay := time.Second * time.Duration(1<<min(max(nextAttempt-1, 0), 10))
	if delay > 30*time.Minute {
		delay = 30 * time.Minute
	}
	_, err := pool.Exec(ctx, `
		UPDATE mail_outbox
		SET attempts=attempts+1,next_attempt_at=now()+$2::interval,last_error=$3,
		    failed_at=CASE WHEN attempts+1>=10 THEN now() ELSE failed_at END,
		    recipient=CASE WHEN attempts+1>=10 THEN '' ELSE recipient END,
		    subject=CASE WHEN attempts+1>=10 THEN '' ELSE subject END,
		    body=CASE WHEN attempts+1>=10 THEN '' ELSE body END
		WHERE id=$1 AND sent_at IS NULL AND failed_at IS NULL`,
		message.id, delay.String(), deliveryErr.Error())
	if err != nil {
		return fmt.Errorf("update outbox retry state: %w", err)
	}
	return nil
}

func messageIDs(messages []outboxMessage) []string {
	ids := make([]string, len(messages))
	for index, message := range messages {
		ids[index] = message.id
	}
	return ids
}
