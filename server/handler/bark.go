package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/accnotify/server/apns"
	"github.com/accnotify/server/crypto"
	"github.com/accnotify/server/model"
	"github.com/accnotify/server/storage"
)

// BarkHandler handles Bark-compatible push requests
type BarkHandler struct {
	storage    *storage.SQLiteStorage
	hub        *Hub
	crypto     *crypto.Crypto
	apnsClient *apns.Client
}

// NewBarkHandler creates a new Bark handler
func NewBarkHandler(storage *storage.SQLiteStorage, hub *Hub, apnsClient *apns.Client) *BarkHandler {
	return &BarkHandler{
		storage:    storage,
		hub:        hub,
		crypto:     crypto.NewCrypto(),
		apnsClient: apnsClient,
	}
}

// HandleRegister handles device registration (Bark-compatible)
func (h *BarkHandler) HandleRegister(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// JSON binding failed, try query parameters
		req.DeviceKey = c.Query("device_key")
		if req.DeviceKey == "" {
			req.DeviceKey = c.Query("key")
		}
		req.DeviceToken = c.Query("device_token")
		if req.DeviceToken == "" {
			req.DeviceToken = c.Query("devicetoken")
		}
		req.PublicKey = c.Query("public_key")
	}
	
	// Also try query parameters if not set from JSON
	if req.DeviceKey == "" {
		req.DeviceKey = c.Query("device_key")
		if req.DeviceKey == "" {
			req.DeviceKey = c.Query("key")
		}
	}
	if req.DeviceToken == "" {
		req.DeviceToken = c.Query("device_token")
		if req.DeviceToken == "" {
			req.DeviceToken = c.Query("devicetoken")
		}
	}
	if req.PublicKey == "" {
		req.PublicKey = c.Query("public_key")
	}

	// Auto-detect device type
	if req.DeviceType == "" {
		if req.DeviceToken != "" {
			req.DeviceType = model.DeviceTypeIOS
		} else if req.PublicKey != "" {
			req.DeviceType = model.DeviceTypeAndroid
		} else {
			c.JSON(http.StatusBadRequest, model.NewBarkError(400, "device_token or public_key is required"))
			return
		}
	}

	// Check if device exists
	device, err := h.storage.GetDeviceByKey(req.DeviceKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewBarkError(500, "database error"))
		return
	}

	if device != nil {
		// Update existing device
		device.DeviceType = req.DeviceType
		if req.DeviceToken != "" {
			device.DeviceToken = req.DeviceToken
		}
		if req.PublicKey != "" {
			device.PublicKey = req.PublicKey
		}
		if err := h.storage.UpdateDevice(device); err != nil {
			c.JSON(http.StatusInternalServerError, model.NewBarkError(500, "failed to update device"))
			return
		}

		c.JSON(http.StatusOK, model.NewBarkResponse(gin.H{
			"key":         device.DeviceKey,
			"device_key":  device.DeviceKey,
			"device_type": device.DeviceType,
		}))
		return
	}

	// Create new device
	if req.DeviceKey == "" {
		req.DeviceKey = uuid.New().String()[:22]
	}

	newDevice := &model.Device{
		DeviceKey:  req.DeviceKey,
		DeviceType: req.DeviceType,
		DeviceToken: req.DeviceToken,
		PublicKey:  req.PublicKey,
		Name:       req.Name,
	}

	if err := h.storage.CreateDevice(newDevice); err != nil {
		c.JSON(http.StatusInternalServerError, model.NewBarkError(500, "failed to create device"))
		return
	}

	c.JSON(http.StatusOK, model.NewBarkResponse(gin.H{
		"key":         newDevice.DeviceKey,
		"device_key":  newDevice.DeviceKey,
		"device_type": newDevice.DeviceType,
	}))
}

// HandlePush handles push notification (Bark-compatible)
func (h *BarkHandler) HandlePush(c *gin.Context) {
	deviceKey := c.Param("device_key")
	if deviceKey == "" {
		deviceKey = c.Query("device_key")
	}

	if deviceKey == "" {
		c.JSON(http.StatusBadRequest, model.NewBarkError(400, "device_key is required"))
		return
	}

	// Get device
	device, err := h.storage.GetDeviceByKey(deviceKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewBarkError(500, "database error"))
		return
	}
	if device == nil {
		c.JSON(http.StatusNotFound, model.NewBarkError(400, "device not found"))
		return
	}

	// Parse request
	var req model.PushRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Try query parameters
		req.Title = c.Query("title")
		req.Body = c.Query("body")
		req.Subtitle = c.Query("subtitle")
		req.Group = c.Query("group")
		req.Sound = c.Query("sound")
		req.URL = c.Query("url")
		req.Icon = c.Query("icon")
		req.Image = c.Query("image")
		req.Level = c.Query("level")
	}
	// Also check query params if not set from JSON
	if req.Image == "" {
		req.Image = c.Query("image")
	}
	if req.Icon == "" {
		req.Icon = c.Query("icon")
	}
	if req.URL == "" {
		req.URL = c.Query("url")
	}

	// Generate message ID
	messageID := uuid.New().String()

	// Push based on device type
	if device.DeviceType == model.DeviceTypeIOS {
		h.pushToIOS(device, &req, messageID, c)
	} else {
		h.pushToAndroid(device, &req, messageID, c)
	}
}

// HandleSimplePush handles GET /:device_key/:title/:body (Bark-compatible)
func (h *BarkHandler) HandleSimplePush(c *gin.Context) {
	deviceKey := c.Param("device_key")
	title := c.Param("title")
	body := c.Param("body")

	// If only one param, treat it as body
	if body == "" {
		body = title
		title = ""
	}

	// Get device
	device, err := h.storage.GetDeviceByKey(deviceKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewBarkError(500, "database error"))
		return
	}
	if device == nil {
		c.JSON(http.StatusNotFound, model.NewBarkError(400, "device not found"))
		return
	}

	// Create push request
	req := &model.PushRequest{
		Title: title,
		Body:  body,
	}

	// Parse query parameters
	if q := c.Query("group"); q != "" {
		req.Group = q
	}
	if q := c.Query("sound"); q != "" {
		req.Sound = q
	}
	if q := c.Query("url"); q != "" {
		req.URL = q
	}
	if q := c.Query("icon"); q != "" {
		req.Icon = q
	}
	if q := c.Query("image"); q != "" {
		req.Image = q
	}
	if q := c.Query("subtitle"); q != "" {
		req.Subtitle = q
	}
	if q := c.Query("level"); q != "" {
		req.Level = q
	}
	if q := c.Query("call"); q != "" {
		req.Call = q == "1" || q == "true"
	}
	if q := c.Query("badge"); q != "" {
		if b, err := strconv.Atoi(q); err == nil {
			req.Badge = b
		}
	}
	if q := c.Query("isArchive"); q != "" {
		req.IsArchive = q == "1" || q == "true"
	}
	if q := c.Query("id"); q != "" {
		req.ID = q
	}

	messageID := uuid.New().String()

	// Push based on device type
	if device.DeviceType == model.DeviceTypeIOS {
		h.pushToIOS(device, req, messageID, c)
	} else {
		h.pushToAndroid(device, req, messageID, c)
	}
}

// pushToIOS pushes message to iOS device via APNs
func (h *BarkHandler) pushToIOS(device *model.Device, req *model.PushRequest, messageID string, c *gin.Context) {
	if h.apnsClient == nil {
		c.JSON(http.StatusInternalServerError, model.NewBarkError(500, "APNs client not configured"))
		return
	}

	if device.DeviceToken == "" {
		c.JSON(http.StatusBadRequest, model.NewBarkError(400, "device token not found"))
		return
	}

	// Build APNs payload
	sound := req.Sound
	if sound != "" && !strings.HasSuffix(sound, ".caf") {
		sound += ".caf"
	}
	if sound == "" {
		sound = "1107.caf"
	}

	// Handle call (持续响铃)
	if req.Call {
		sound = "alarm.caf"
	}

	// Prepare badge
	var badge *int
	if req.Badge > 0 {
		badge = &req.Badge
	}

	payload := &apns.Payload{
		Aps: apns.Aps{
			Alert: apns.Alert{
				Title:    req.Title,
				Subtitle: req.Subtitle,
				Body:     req.Body,
			},
			Badge:          badge,
			Sound:          sound,
			ThreadID:       req.Group,
			Category:       "myNotificationCategory",
			MutableContent: 1,
		},
		Group:     req.Group,
		Icon:      req.Icon,
		Image:     req.Image,
		URL:       req.URL,
		Badge:     req.Badge,
		Call:      req.Call,
		IsArchive: req.IsArchive,
		Level:     req.Level,
		Delete:    req.Delete,
	}

	// Send to APNs
	headers := make(map[string]string)
	if req.ID != "" {
		headers["apns-collapse-id"] = req.ID
	}

	resp, err := h.apnsClient.Push(device.DeviceToken, payload, headers)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewBarkError(500, "APNs push failed: "+err.Error()))
		return
	}

	if resp.StatusCode == 200 {
		c.JSON(http.StatusOK, model.NewBarkResponse(nil))
		return
	}

	// Handle errors
	if resp.StatusCode == 410 || strings.Contains(resp.Reason, "BadDeviceToken") {
		// Device token invalid, clear it
		h.storage.UpdateDeviceToken(device.DeviceKey, "")
	}

	c.JSON(http.StatusBadRequest, model.NewBarkError(int64(resp.StatusCode), "APNs push failed: "+resp.Reason))
}

// pushToAndroid pushes message to Android device via WebSocket
func (h *BarkHandler) pushToAndroid(device *model.Device, req *model.PushRequest, messageID string, c *gin.Context) {
	// Build message data
	data := map[string]interface{}{
		"title":     req.Title,
		"body":      req.Body,
		"group":     req.Group,
		"icon":      req.Icon,
		"url":       req.URL,
		"sound":     req.Sound,
		"badge":     req.Badge,
		"level":     req.Level,
		"call":      req.Call,
		"isArchive": req.IsArchive,
	}

	// Encrypt if device has public key
	var encrypted string
	if device.PublicKey != "" {
		pubKey, err := h.crypto.ParsePublicKey(device.PublicKey)
		if err == nil {
			dataBytes, _ := json.Marshal(data)
			encrypted, _ = h.crypto.EncryptMessage(pubKey, dataBytes)
		}
	}

	// Create WebSocket message
	wsMsg := &model.WSMessage{
		Type:      model.WSTypeMessage,
		ID:        messageID,
		Timestamp: time.Now().Unix(),
	}

	if encrypted != "" {
		wsMsg.Data = map[string]interface{}{
			"encrypted_content": encrypted,
		}
	} else {
		wsMsg.Data = data
	}

	// Send via WebSocket
	delivered := h.hub.SendToDevice(device.DeviceKey, wsMsg)

	if delivered {
		c.JSON(http.StatusOK, model.NewBarkResponse(nil))
	} else {
		// Device offline, save message
		msg := &model.Message{
			DeviceID:         device.ID,
			MessageID:        messageID,
			Title:            req.Title,
			Body:             req.Body,
			Group:            req.Group,
			Icon:             req.Icon,
			URL:              req.URL,
			Sound:            req.Sound,
			Badge:            req.Badge,
			EncryptedPayload: []byte(encrypted),
		}
		h.storage.CreateMessage(msg)
		c.JSON(http.StatusOK, model.NewBarkResponse(nil))
	}
}

// HandleHealth handles health check
func (h *BarkHandler) HandleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
	})
}

// HandleInfo handles server info
func (h *BarkHandler) HandleInfo(c *gin.Context) {
	count, _ := h.storage.CountDevices()
	c.JSON(http.StatusOK, gin.H{
		"version": "v2.3.0",
		"build":   "2026-02-12",
		"devices": count,
		"time":    time.Now().Unix(),
	})
}
