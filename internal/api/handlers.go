package api

import (
	"net/http"

	"messaging/internal/models"
	"messaging/internal/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Scheduler *service.Scheduler
	Service   *service.MessageService
}

func NewAPIHandler(scheduler *service.Scheduler, service *service.MessageService) *Handler {
	return &Handler{
		Scheduler: scheduler,
		Service:   service,
	}
}

type SentMessagesResponse struct {
	SentMessages []models.Message `json:"sent_messages"`
}

func (h *Handler) StartAuto(c *gin.Context) {
	if h.Scheduler.IsRunning() {
		c.JSON(http.StatusOK, gin.H{"message": "Scheduler already running"})
		return
	}
	_ = h.Scheduler.Start()
	c.JSON(http.StatusOK, gin.H{"message": "Scheduler started"})
}

func (h *Handler) StopAuto(c *gin.Context) {
	if !h.Scheduler.IsRunning() {
		c.JSON(http.StatusOK, gin.H{"message": "Scheduler already stopped"})
		return
	}
	_ = h.Scheduler.Stop()
	c.JSON(http.StatusOK, gin.H{"message": "Scheduler stopped"})
}

func (h *Handler) ListSentMessages(c *gin.Context) {
	messages, err := h.Service.ListSentMessages()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"sent_messages": messages})
}
