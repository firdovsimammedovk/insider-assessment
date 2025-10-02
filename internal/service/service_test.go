package service

import (
	"fmt"
	"testing"
	"time"

	"messaging/internal/models"
)

type stubRepo struct {
	unsentMessages []models.Message
	markedSent     []struct {
		ID     int
		ExtID  string
		SentAt time.Time
	}
}

func (r *stubRepo) GetUnsentMessages(limit int) ([]models.Message, error) {
	if len(r.unsentMessages) == 0 {
		return []models.Message{}, nil
	}
	count := limit
	if len(r.unsentMessages) < limit {
		count = len(r.unsentMessages)
	}
	return r.unsentMessages[:count], nil
}
func (r *stubRepo) MarkMessageSent(id int, externalID string, sentAt time.Time) error {
	r.markedSent = append(r.markedSent, struct {
		ID     int
		ExtID  string
		SentAt time.Time
	}{ID: id, ExtID: externalID, SentAt: sentAt})
	for i, m := range r.unsentMessages {
		if m.ID == id {
			r.unsentMessages = append(r.unsentMessages[:i], r.unsentMessages[i+1:]...)
			break
		}
	}
	return nil
}
func (r *stubRepo) ListSentMessages() ([]models.Message, error) {
	return nil, nil
}

type stubSender struct {
	sentCalls []struct {
		To      string
		Content string
	}
	externalID string
	failNext   bool
}

func (s *stubSender) SendMessage(to string, content string) (string, error) {
	s.sentCalls = append(s.sentCalls, struct {
		To      string
		Content string
	}{To: to, Content: content})
	if s.failNext {
		s.failNext = false
		return "", fmt.Errorf("simulated send failure")
	}
	if s.externalID == "" {
		s.externalID = "dummy-ext-id"
	}
	return s.externalID, nil
}

type stubCache struct {
	stored map[string]time.Time
}

func (c *stubCache) StoreSentMessage(externalID string, sentAt time.Time) error {
	if c.stored == nil {
		c.stored = make(map[string]time.Time)
	}
	c.stored[externalID] = sentAt
	return nil
}

func TestProcessPendingMessages(t *testing.T) {
	now := time.Now().UTC()
	msg1 := models.Message{ID: 1, To: "+1111111", Content: "Hello", CreatedAt: now, UpdatedAt: now}
	msg2 := models.Message{ID: 2, To: "+2222222", Content: "World", CreatedAt: now, UpdatedAt: now}
	repo := &stubRepo{unsentMessages: []models.Message{msg1, msg2}}
	sender := &stubSender{externalID: "ext-12345"}
	cache := &stubCache{}
	service := NewMessageService(repo, sender, cache)

	err := service.ProcessPendingMessages(2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sender.sentCalls) != 2 {
		t.Errorf("expected 2 send calls, got %d", len(sender.sentCalls))
	}
	if len(repo.markedSent) != 2 {
		t.Errorf("expected 2 messages marked as sent, got %d", len(repo.markedSent))
	}
	if len(cache.stored) != 1 || cache.stored["ext-12345"].IsZero() {
		t.Errorf("expected externalID ext-12345 stored with timestamp")
	}

	repo = &stubRepo{unsentMessages: []models.Message{msg1}}
	sender = &stubSender{externalID: "ext-999", failNext: true}
	cache = &stubCache{}
	service = NewMessageService(repo, sender, cache)
	err = service.ProcessPendingMessages(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sender.sentCalls) != 1 {
		t.Errorf("expected 1 send attempt, got %d", len(sender.sentCalls))
	}
	if len(repo.markedSent) != 0 {
		t.Errorf("expected 0 messages marked as sent on failure, got %d", len(repo.markedSent))
	}
	if len(cache.stored) != 0 {
		t.Errorf("expected 0 cache entries on failure, got %d", len(cache.stored))
	}
}
