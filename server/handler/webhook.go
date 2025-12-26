package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/accnotify/server/crypto"
	"github.com/accnotify/server/model"
	"github.com/accnotify/server/storage"
)

// WebhookHandler handles webhook requests from various services
type WebhookHandler struct {
	storage *storage.SQLiteStorage
	hub     *Hub
	crypto  *crypto.Crypto
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(storage *storage.SQLiteStorage, hub *Hub) *WebhookHandler {
	return &WebhookHandler{
		storage: storage,
		hub:     hub,
		crypto:  crypto.NewCrypto(),
	}
}

// HandleGenericWebhook handles POST /webhook/:device_key (generic webhook)
func (h *WebhookHandler) HandleGenericWebhook(c *gin.Context) {
	deviceKey := c.Param("device_key")
	if deviceKey == "" {
		c.JSON(http.StatusBadRequest, model.PushResponse{
			Success: false,
			Error:   "Missing device key",
		})
		return
	}

	// Verify device exists
	device, err := h.storage.GetDeviceByKey(deviceKey)
	if err != nil || device == nil {
		c.JSON(http.StatusNotFound, model.PushResponse{
			Success: false,
			Error:   "Device not found",
		})
		return
	}

	// Read raw body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.PushResponse{
			Success: false,
			Error:   "Failed to read request body",
		})
		return
	}

	// Try to parse as JSON
	var jsonData map[string]interface{}
	if err := json.Unmarshal(body, &jsonData); err == nil {
		// Format JSON for display
		formatted, _ := json.MarshalIndent(jsonData, "", "  ")
		h.sendWebhookMessage(deviceKey, device, "Webhook", string(formatted), c)
		return
	}

	// If not JSON, send as plain text
	h.sendWebhookMessage(deviceKey, device, "Webhook", string(body), c)
}

// HandleGitHubWebhook handles POST /webhook/:device_key/github
func (h *WebhookHandler) HandleGitHubWebhook(c *gin.Context) {
	deviceKey := c.Param("device_key")
	if deviceKey == "" {
		c.JSON(http.StatusBadRequest, model.PushResponse{
			Success: false,
			Error:   "Missing device key",
		})
		return
	}

	device, err := h.storage.GetDeviceByKey(deviceKey)
	if err != nil || device == nil {
		c.JSON(http.StatusNotFound, model.PushResponse{
			Success: false,
			Error:   "Device not found",
		})
		return
	}

	var webhook model.GitHubWebhook
	if err := c.ShouldBindJSON(&webhook); err != nil {
		h.sendWebhookMessage(deviceKey, device, "GitHub", "Invalid GitHub webhook format", c)
		return
	}

	// Parse GitHub event type
	eventType := c.GetHeader("X-GitHub-Event")

	title := "GitHub"
	body := h.formatGitHubWebhook(&webhook, eventType)

	h.sendWebhookMessage(deviceKey, device, title, body, c)
}

// HandleGitLabWebhook handles POST /webhook/:device_key/gitlab
func (h *WebhookHandler) HandleGitLabWebhook(c *gin.Context) {
	deviceKey := c.Param("device_key")
	if deviceKey == "" {
		c.JSON(http.StatusBadRequest, model.PushResponse{
			Success: false,
			Error:   "Missing device key",
		})
		return
	}

	device, err := h.storage.GetDeviceByKey(deviceKey)
	if err != nil || device == nil {
		c.JSON(http.StatusNotFound, model.PushResponse{
			Success: false,
			Error:   "Device not found",
		})
		return
	}

	var webhook model.GitLabWebhook
	if err := c.ShouldBindJSON(&webhook); err != nil {
		h.sendWebhookMessage(deviceKey, device, "GitLab", "Invalid GitLab webhook format", c)
		return
	}

	title := "GitLab"
	body := h.formatGitLabWebhook(&webhook)

	h.sendWebhookMessage(deviceKey, device, title, body, c)
}

// HandleDockerHubWebhook handles POST /webhook/:device_key/docker
func (h *WebhookHandler) HandleDockerHubWebhook(c *gin.Context) {
	deviceKey := c.Param("device_key")
	if deviceKey == "" {
		c.JSON(http.StatusBadRequest, model.PushResponse{
			Success: false,
			Error:   "Missing device key",
		})
		return
	}

	device, err := h.storage.GetDeviceByKey(deviceKey)
	if err != nil || device == nil {
		c.JSON(http.StatusNotFound, model.PushResponse{
			Success: false,
			Error:   "Device not found",
		})
		return
	}

	var webhook model.DockerHubWebhook
	if err := c.ShouldBindJSON(&webhook); err != nil {
		h.sendWebhookMessage(deviceKey, device, "Docker Hub", "Invalid Docker Hub webhook format", c)
		return
	}

	title := "Docker Hub"
	body := h.formatDockerHubWebhook(&webhook)

	h.sendWebhookMessage(deviceKey, device, title, body, c)
}

// HandleGiteaWebhook handles POST /webhook/:device_key/gitea
func (h *WebhookHandler) HandleGiteaWebhook(c *gin.Context) {
	deviceKey := c.Param("device_key")
	if deviceKey == "" {
		c.JSON(http.StatusBadRequest, model.PushResponse{
			Success: false,
			Error:   "Missing device key",
		})
		return
	}

	device, err := h.storage.GetDeviceByKey(deviceKey)
	if err != nil || device == nil {
		c.JSON(http.StatusNotFound, model.PushResponse{
			Success: false,
			Error:   "Device not found",
		})
		return
	}

	var webhook model.GiteaWebhook
	if err := c.ShouldBindJSON(&webhook); err != nil {
		h.sendWebhookMessage(deviceKey, device, "Gitea", "Invalid Gitea webhook format", c)
		return
	}

	eventType := c.GetHeader("X-Gitea-Event")
	title := "Gitea"
	body := h.formatGiteaWebhook(&webhook, eventType)

	h.sendWebhookMessage(deviceKey, device, title, body, c)
}

// sendWebhookMessage sends a webhook message to device
func (h *WebhookHandler) sendWebhookMessage(deviceKey string, device *model.Device, title, body string, c *gin.Context) {
	messageID := uuid.New().String()

	msg := &model.Message{
		DeviceID:  device.ID,
		MessageID: messageID,
		Title:     title,
		Body:      body,
		Group:     "webhook",
	}

	// Encrypt if device has public key
	var encryptedContent string
	if device.PublicKey != "" {
		publicKey, err := h.crypto.ParsePublicKey(device.PublicKey)
		if err == nil {
			payload := map[string]interface{}{
				"title": title,
				"body":  body,
				"group": "webhook",
			}
			payloadBytes, _ := json.Marshal(payload)
			encryptedContent, _ = h.crypto.EncryptMessage(publicKey, payloadBytes)
			msg.EncryptedPayload = []byte(encryptedContent)
		}
	}

	if err := h.storage.CreateMessage(msg); err != nil {
		c.JSON(http.StatusInternalServerError, model.PushResponse{
			Success: false,
			Error:   "Failed to store message",
		})
		return
	}

	wsMsg := &model.WSMessage{
		Type:      model.WSTypeMessage,
		ID:        messageID,
		Timestamp: 0, // Will be set in hub
		Data: map[string]interface{}{
			"title":             title,
			"body":              body,
			"group":             "webhook",
			"encrypted_content": encryptedContent,
		},
	}

	delivered := h.hub.SendToDevice(deviceKey, wsMsg)
	if delivered {
		h.storage.MarkMessageDelivered(messageID)
	}

	c.JSON(http.StatusOK, model.PushResponse{
		Success:   true,
		MessageID: messageID,
	})
}

// formatGitHubWebhook formats GitHub webhook into readable message
func (h *WebhookHandler) formatGitHubWebhook(w *model.GitHubWebhook, eventType string) string {
	var sb strings.Builder

	switch eventType {
	case "push":
		branch := strings.TrimPrefix(w.Ref, "refs/heads/")
		sb.WriteString(fmt.Sprintf("【Push】%s\n", w.Repository.FullName))
		sb.WriteString(fmt.Sprintf("分支: %s\n", branch))
		if !w.Forced {
			sb.WriteString(fmt.Sprintf("提交: %s\n", w.HeadCommit.Message))
		} else {
			sb.WriteString("强制推送\n")
		}
		sb.WriteString(fmt.Sprintf("推送者: %s", w.Pusher.Name))
	case "ping":
		sb.WriteString(fmt.Sprintf("【Ping】%s\n", w.Repository.FullName))
	default:
		sb.WriteString(fmt.Sprintf("【%s】%s\n", eventType, w.Repository.FullName))
	}

	return sb.String()
}

// formatGitLabWebhook formats GitLab webhook into readable message
func (h *WebhookHandler) formatGitLabWebhook(w *model.GitLabWebhook) string {
	var sb strings.Builder

	switch w.ObjectKind {
	case "push":
		branch := strings.TrimPrefix(w.Ref, "refs/heads/")
		sb.WriteString(fmt.Sprintf("【Push】%s\n", w.Project.Name))
		sb.WriteString(fmt.Sprintf("分支: %s\n", branch))
		sb.WriteString(fmt.Sprintf("提交: %s\n", w.Commit.Message))
		sb.WriteString(fmt.Sprintf("推送者: %s", w.UserName))
	case "merge_request":
		sb.WriteString(fmt.Sprintf("【Merge Request】%s\n", w.Project.Name))
	default:
		sb.WriteString(fmt.Sprintf("【%s】%s\n", w.ObjectKind, w.Project.Name))
	}

	return sb.String()
}

// formatDockerHubWebhook formats Docker Hub webhook into readable message
func (h *WebhookHandler) formatDockerHubWebhook(w *model.DockerHubWebhook) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("【Docker Hub】%s\n", w.Repository.RepoName))
	sb.WriteString(fmt.Sprintf("标签: %s\n", w.PushData.Tag))
	sb.WriteString(fmt.Sprintf("推送者: %s", w.PushData.Pusher))

	return sb.String()
}

// formatGiteaWebhook formats Gitea webhook into readable message
func (h *WebhookHandler) formatGiteaWebhook(w *model.GiteaWebhook, eventType string) string {
	var sb strings.Builder

	switch eventType {
	case "push":
		branch := strings.TrimPrefix(w.Ref, "refs/heads/")
		sb.WriteString(fmt.Sprintf("【Push】%s\n", w.Repository.FullName))
		sb.WriteString(fmt.Sprintf("分支: %s\n", branch))
		sb.WriteString(fmt.Sprintf("提交: %s\n", w.HeadCommit.Message))
		sb.WriteString(fmt.Sprintf("推送者: %s", w.Pusher.Name))
	default:
		sb.WriteString(fmt.Sprintf("【%s】%s\n", eventType, w.Repository.FullName))
	}

	return sb.String()
}
