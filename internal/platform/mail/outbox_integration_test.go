package mail

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type testSender struct {
	err       error
	delivered int
}

func (sender *testSender) Send(string, string, string) error {
	sender.delivered++
	return sender.err
}

func TestOutboxDeliveryAndRetry(t *testing.T) {
	databaseURL := os.Getenv("MAIL_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("MAIL_TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatal(err)
	}
	sentID := insertTestEmail(t, ctx, pool)
	retryID := insertTestEmail(t, ctx, pool)
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM mail_outbox WHERE id=ANY($1::uuid[])`, []string{sentID, retryID}); err != nil {
			t.Errorf("delete outbox integration messages: %v", err)
		}
		pool.Close()
	})
	sender := &testSender{}
	if err := deliverOutboxBatch(ctx, pool, sender, testLogger()); err != nil {
		t.Fatalf("deliver outbox batch: %v", err)
	}
	if sender.delivered < 2 {
		t.Fatalf("delivered messages = %d, want at least 2", sender.delivered)
	}
	var sent bool
	if err := pool.QueryRow(ctx, `SELECT sent_at IS NOT NULL FROM mail_outbox WHERE id=$1`, sentID).Scan(&sent); err != nil {
		t.Fatal(err)
	}
	if !sent {
		t.Fatal("successfully delivered email was not marked sent")
	}

	failedID := insertTestEmail(t, ctx, pool)
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM mail_outbox WHERE id=$1`, failedID); err != nil {
			t.Errorf("delete failed outbox integration message: %v", err)
		}
	})
	failedSender := &testSender{err: errors.New("temporary SMTP failure")}
	if err := deliverOutboxBatch(ctx, pool, failedSender, testLogger()); err != nil {
		t.Fatalf("process failed email: %v", err)
	}
	var attempts int
	var lastError string
	if err := pool.QueryRow(ctx, `SELECT attempts,last_error FROM mail_outbox WHERE id=$1`, failedID).Scan(&attempts, &lastError); err != nil {
		t.Fatal(err)
	}
	if attempts != 1 || lastError != "temporary SMTP failure" {
		t.Fatalf("failed delivery state = attempts %d, error %q", attempts, lastError)
	}
}

func insertTestEmail(t *testing.T, ctx context.Context, pool *pgxpool.Pool) string {
	t.Helper()
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		t.Fatal(err)
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(raw[:])
	id := encoded[:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:]
	if _, err := pool.Exec(ctx, `
		INSERT INTO mail_outbox (id,recipient,subject,body) VALUES ($1,'test@example.invalid','test','body')`, id); err != nil {
		t.Fatal(err)
	}
	return id
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
