package service

import (
	"log"
	"time"

	"messaging/internal/models"
)

type MessageRepository interface {
	GetUnsentMessages(limit int) ([]models.Message, error)
	MarkMessageSent(id int, externalID string, sentAt time.Time) error
	ListSentMessages() ([]models.Message, error)
}

type MessageSender interface {
	SendMessage(to string, content string) (string, error)
}

type MessageCache interface {
	StoreSentMessage(externalID string, sentAt time.Time) error
}

type MessageService struct {
	repo   MessageRepository
	sender MessageSender
	cache  MessageCache
}

func NewMessageService(repo MessageRepository, sender MessageSender, cache MessageCache) *MessageService {
	return &MessageService{repo: repo, sender: sender, cache: cache}
}

func (s *MessageService) ProcessPendingMessages(limit int) error {
	messages, err := s.repo.GetUnsentMessages(limit)
	if err != nil {
		return err
	}
	if len(messages) == 0 {
		return nil
	}
	for _, msg := range messages {
		const MaxContentLength = 160
		if len(msg.Content) > MaxContentLength {
			log.Printf("Content for message %d exceeds %d chars, truncating", msg.ID, MaxContentLength)
			msg.Content = msg.Content[:MaxContentLength]
		}
		externalID, err := s.sender.SendMessage(msg.To, msg.Content)
		if err != nil {
			log.Printf("Failed to send message %d: %v", msg.ID, err)
			continue
		}
		sentTime := time.Now().UTC()
		err = s.repo.MarkMessageSent(msg.ID, externalID, sentTime)
		if err != nil {
			log.Printf("Error marking message %d as sent in DB: %v", msg.ID, err)
		}
		err = s.cache.StoreSentMessage(externalID, sentTime)
		if err != nil {
			log.Printf("Error caching sent message %d (external ID %s): %v", msg.ID, externalID, err)
		}
		log.Printf("Message %d sent (externalId=%s)", msg.ID, externalID)
	}
	return nil
}

func (s *MessageService) ListSentMessages() ([]models.Message, error) {
	return s.repo.ListSentMessages()
}
