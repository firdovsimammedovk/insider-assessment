package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"

	"messaging/internal/api"
	"messaging/internal/repository"
	"messaging/internal/service"

	"github.com/redis/go-redis/v9"
)

func main() {
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "postgres")
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	webhookURL := getEnv("WEBHOOK_URL", "https://webhook.site/c3f13233-1ed4-429e-9649-8133b3b9c9cd")
	webhookAuthKey := getEnv("WEBHOOK_AUTH_KEY", "INS.me1x9uMcyYGlhKKQVPoc.bO3j9aZwRTOcA2Ywo")

	connStr := "host=" + dbHost +
		" port=" + dbPort +
		" user=" + dbUser +
		" password=" + dbPassword +
		" dbname=" + dbName +
		" sslmode=disable"
	repo, err := repository.NewPostgresRepo(connStr)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	log.Println("Connected to PostgreSQL")

	redisAddr := redisHost + ":" + redisPort
	cacheClient := initRedis(redisAddr, redisPassword)
	redisCache := &RedisCache{client: cacheClient}
	log.Println("Connected to Redis")

	sender := &WebhookSender{
		client:  &http.Client{Timeout: 10 * time.Second},
		url:     webhookURL,
		authKey: webhookAuthKey,
	}
	serv := service.NewMessageService(repo, sender, redisCache)
	scheduler := service.NewScheduler(serv, 2*time.Minute)
	if err := scheduler.Start(); err != nil {
		log.Fatalf("Failed to start scheduler: %v", err)
	}

	r := gin.Default()
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	handler := api.NewAPIHandler(scheduler, serv)
	v1 := r.Group("/api/v1")
	{
		v1.POST("/scheduler/start", handler.StartAuto)
		v1.POST("/scheduler/stop", handler.StopAuto)
		v1.GET("/messages/sent", handler.ListSentMessages)
	}

	port := getEnv("PORT", "8080")
	log.Printf("Server starting on port %s...", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}

func getEnv(key string, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}

type RedisCache struct {
	client *redis.Client
}

func (rc *RedisCache) StoreSentMessage(externalID string, sentAt time.Time) error {
	ctx := context.Background()
	err := rc.client.Set(ctx, "msgid:"+externalID, sentAt.Format(time.RFC3339), 0).Err()
	return err
}

func initRedis(addr string, password string) *redis.Client {
	opts := &redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	}
	client := redis.NewClient(opts)
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	return client
}

type WebhookSender struct {
	client  *http.Client
	url     string
	authKey string
}

func (w *WebhookSender) SendMessage(to string, content string) (string, error) {
	payload := map[string]string{"to": to, "content": content}
	bodyBytes, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", w.url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if w.authKey != "" {
		req.Header.Set("x-ins-auth-key", w.authKey)
	}
	resp, err := w.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("received status %d from webhook", resp.StatusCode)
	}
	var respData struct {
		Message   string `json:"message"`
		MessageID string `json:"messageId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return "", fmt.Errorf("failed to parse webhook response: %w", err)
	}
	if respData.MessageID == "" {
		return "", fmt.Errorf("no messageId in response")
	}
	return respData.MessageID, nil
}
