package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/abnotify/server/model"
	"github.com/abnotify/server/storage"
)

// PushHandler handles push notification requests
type PushHandler struct {
	storage *storage.SQLiteStorage
	hub     *Hub
}

// NewPushHandler creates a new push handler
func NewPushHandler(storage *storage.SQLiteStorage, hub *Hub) *PushHandler {
	return &PushHandler{
		storage: storage,
		hub:     hub,
	}
}

// HandlePush handles POST /push/:device_key
func (h *PushHandler) HandlePush(c *gin.Context) {
	deviceKey := c.Param("device_key")
	if deviceKey == "" {
		c.JSON(http.StatusBadRequest, model.PushResponse{
			Success: false,
			Error:   "Missing device key",
		})
		return
	}

	// Get device
	device, err := h.storage.GetDeviceByKey(deviceKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.PushResponse{
			Success: false,
			Error:   "Database error",
		})
		return
	}
	if device == nil {
		c.JSON(http.StatusNotFound, model.PushResponse{
			Success: false,
			Error:   "Device not found",
		})
		return
	}

	// Parse request - support both JSON and form-data (for SMS Forwarder compatibility)
	var req model.PushRequest
	// Try JSON first
	if err := c.ShouldBindJSON(&req); err != nil {
		// Fall back to form-data binding
		c.Bind(&req)
	}
	// Log received request for debugging
	log.Printf("HandlePush received: deviceKey=%s, Title='%s', Body='%s', Content='%s', Msg='%s', Message='%s', Text='%s'", 
		deviceKey, req.Title, req.Body, req.Content, req.Msg, req.Message, req.Text)
	
	// SMS Forwarder compatibility: check alternative field names for body
	// Common field names: body, content, msg, message, text, desp, description
	if req.Body == "" {
		if req.Content != "" {
			req.Body = req.Content
		} else if req.Msg != "" {
			req.Body = req.Msg
		} else if req.Message != "" {
			req.Body = req.Message
		} else if req.Text != "" {
			req.Body = req.Text
		} else if req.Desp != "" {
			req.Body = req.Desp
		} else if req.Description != "" {
			req.Body = req.Description
		}
	}
	// Default title if empty
	if req.Title == "" {
		req.Title = "Abnotify"
	}
	log.Printf("HandlePush processed: Title='%s', Body='%s'", req.Title, req.Body)

	// Generate message ID
	messageID := uuid.New().String()

	// Create message
	msg := &model.Message{
		DeviceID:  device.ID,
		MessageID: messageID,
		Title:     req.Title,
		Body:      req.Body,
		Group:     req.Group,
		Icon:      req.Icon,
		URL:       req.URL,
		Sound:     req.Sound,
		Badge:     req.Badge,
	}

	// Store message
	if err := h.storage.CreateMessage(msg); err != nil {
		c.JSON(http.StatusInternalServerError, model.PushResponse{
			Success: false,
			Error:   "Failed to store message",
		})
		return
	}

	// Prepare WebSocket message
	wsMsg := &model.WSMessage{
		Type:      model.WSTypeMessage,
		ID:        messageID,
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"title": req.Title,
			"body":  req.Body,
			"group": req.Group,
			"icon":  req.Icon,
			"url":   req.URL,
			"sound": req.Sound,
			"badge": req.Badge,
		},
	}

	// Send via WebSocket
	delivered := h.hub.SendToDevice(deviceKey, wsMsg)

	if delivered {
		h.storage.MarkMessageDelivered(messageID)
	}

	c.JSON(http.StatusOK, model.PushResponse{
		Success:   true,
		MessageID: messageID,
	})
}

// HandleSimplePush handles GET /push/:device_key/:title/:body (Bark-compatible)
func (h *PushHandler) HandleSimplePush(c *gin.Context) {
	deviceKey := c.Param("device_key")
	title := c.Param("title")
	body := c.Param("body")

	// If only one param, treat it as body
	if body == "" {
		body = title
		title = "Abnotify"
	}

	log.Printf("PushHandler.HandleSimplePush: deviceKey=%s, title=%s, body=%s", deviceKey, title, body)

	// Convert to POST request
	c.Set("device_key", deviceKey)

	// Get device
	device, err := h.storage.GetDeviceByKey(deviceKey)
	if err != nil || device == nil {
		c.JSON(http.StatusNotFound, model.PushResponse{
			Success: false,
			Error:   "Device not found",
		})
		return
	}

	messageID := uuid.New().String()

	msg := &model.Message{
		DeviceID:  device.ID,
		MessageID: messageID,
		Title:     title,
		Body:      body,
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
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"title": title,
			"body":  body,
		},
	}

	delivered := h.hub.SendToDevice(deviceKey, wsMsg)
	log.Printf("PushHandler.SendToDevice: deviceKey=%s, delivered=%v, title=%s, body=%s", deviceKey, delivered, title, body)
	if delivered {
		h.storage.MarkMessageDelivered(messageID)
	}

	c.JSON(http.StatusOK, model.PushResponse{
		Success:   true,
		MessageID: messageID,
	})
}

// HandleRegister handles POST /register
func (h *PushHandler) HandleRegister(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
		})
		return
	}

	// Check if device exists
	device, err := h.storage.GetDeviceByKey(req.DeviceKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database error",
		})
		return
	}

	if device != nil {
		// Update public key
		if req.PublicKey != "" {
			if err := h.storage.UpdateDevicePublicKey(req.DeviceKey, req.PublicKey); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error":   "Failed to update public key",
				})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{
			"success":    true,
			"device_key": req.DeviceKey,
			"message":    "Device updated",
		})
		return
	}

	// Create new device
	newDevice := &model.Device{
		DeviceKey: req.DeviceKey,
		PublicKey: req.PublicKey,
		Name:      req.Name,
	}

	if err := h.storage.CreateDevice(newDevice); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create device",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"device_key": req.DeviceKey,
		"message":    "Device registered",
	})
}

// HandleHealth handles GET /health
func (h *PushHandler) HandleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
	})
}
