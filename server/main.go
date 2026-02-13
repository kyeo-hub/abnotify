package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/abnotify/server/apns"
	"github.com/abnotify/server/config"
	"github.com/abnotify/server/handler"
	"github.com/abnotify/server/storage"
)

func main() {
	// Load configuration
	cfg := config.LoadFromEnv()

	// Initialize storage
	store, err := storage.NewSQLiteStorage(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// Initialize WebSocket hub
	hub := handler.NewHub(store)
	go hub.Run()

	// Initialize APNs client (if configured)
	var apnsClient *apns.Client
	if cfg.APNSKeyID != "" && cfg.APNSTeamID != "" && cfg.APNSPrivateKey != "" {
		apnsClient, err = apns.NewClient(cfg.APNSKeyID, cfg.APNSTeamID, cfg.APNSPrivateKey, cfg.APNSProduction)
		if err != nil {
			log.Printf("Warning: Failed to initialize APNs client: %v", err)
		} else {
			log.Println("APNs client initialized successfully")
		}
	} else {
		log.Println("APNs not configured, iOS push disabled")
	}

	// Initialize handlers
	pushHandler := handler.NewPushHandler(store, hub)
	barkHandler := handler.NewBarkHandler(store, hub, apnsClient)
	wsHandler := handler.NewWSHandler(hub, store)
	webhookHandler := handler.NewWebhookHandler(store, hub)

	// Setup Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Health check
	router.GET("/health", barkHandler.HandleHealth)
	router.GET("/healthz", func(c *gin.Context) { c.String(200, "ok") })
	router.GET("/ping", func(c *gin.Context) { c.JSON(200, gin.H{"code": 200, "message": "pong", "timestamp": time.Now().Unix()}) })

	// Register
	router.POST("/register", barkHandler.HandleRegister)
	router.GET("/register", barkHandler.HandleRegister)

	// Info
	router.GET("/info", barkHandler.HandleInfo)

	// Push routes (Abnotify style)
	router.POST("/push/:device_key", pushHandler.HandlePush)
	router.GET("/push/:device_key/*params", handleSimplePushParams(pushHandler))

	// Bark-compatible routes
	router.POST("/:device_key", barkHandler.HandlePush)
	router.GET("/:device_key", barkHandler.HandlePush)
	// Use wildcard to handle variable path segments: /:device_key/:body, /:device_key/:title/:body, etc.
	router.GET("/:device_key/*params", handleBarkParams(barkHandler))
	router.POST("/:device_key/*params", handleBarkParams(barkHandler))

	// WebSocket
	router.GET("/ws", func(c *gin.Context) {
		wsHandler.HandleConnect(c.Writer, c.Request)
	})

	// Webhook routes
	webhookGroup := router.Group("/webhook/:device_key")
	{
		webhookGroup.POST("", webhookHandler.HandleGenericWebhook)
		webhookGroup.POST("/github", webhookHandler.HandleGitHubWebhook)
		webhookGroup.POST("/gitlab", webhookHandler.HandleGitLabWebhook)
		webhookGroup.POST("/docker", webhookHandler.HandleDockerHubWebhook)
		webhookGroup.POST("/gitea", webhookHandler.HandleGiteaWebhook)
	}

	// Root
	router.GET("/", func(c *gin.Context) { c.String(200, "ok") })

	// Create server
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Abnotify server starting on %s", addr)
		log.Printf("Supporting both iOS (APNs) and Android (WebSocket) devices")
		var err error
		if cfg.EnableHTTPS {
			err = srv.ListenAndServeTLS(cfg.CertFile, cfg.KeyFile)
		} else {
			err = srv.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// handleSimplePushParams handles /push/:device_key/*params
// Supports: /push/key/body OR /push/key/title/body
func handleSimplePushParams(h *handler.PushHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		params := c.Param("params")
		// Remove leading slash
		if len(params) > 0 && params[0] == '/' {
			params = params[1:]
		}
		
		// Split by /
		parts := strings.SplitN(params, "/", 2)
		
		var title, body string
		if len(parts) == 1 {
			// Only body provided
			title = "Abnotify"
			body = parts[0]
		} else {
			// Title and body provided
			title = parts[0]
			body = parts[1]
		}
		
		c.Params = append(c.Params, gin.Param{Key: "title", Value: title})
		c.Params = append(c.Params, gin.Param{Key: "body", Value: body})
		h.HandleSimplePush(c)
	}
}

// handleBarkParams handles Bark-compatible routes with variable path segments
// Supports: /:device_key/:body, /:device_key/:title/:body, /:device_key/:title/:subtitle/:body
func handleBarkParams(h *handler.BarkHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		params := c.Param("params")
		// Remove leading slash
		if len(params) > 0 && params[0] == '/' {
			params = params[1:]
		}
		
		if params == "" {
			// No additional params, just /:device_key
			h.HandlePush(c)
			return
		}
		
		// Split by /
		parts := strings.Split(params, "/")
		
		// Set params based on number of segments
		switch len(parts) {
		case 1:
			// /:device_key/:body
			c.Params = append(c.Params, gin.Param{Key: "title", Value: ""})
			c.Params = append(c.Params, gin.Param{Key: "body", Value: parts[0]})
		case 2:
			// /:device_key/:title/:body
			c.Params = append(c.Params, gin.Param{Key: "title", Value: parts[0]})
			c.Params = append(c.Params, gin.Param{Key: "body", Value: parts[1]})
		case 3:
			// /:device_key/:title/:subtitle/:body
			c.Params = append(c.Params, gin.Param{Key: "title", Value: parts[0]})
			c.Params = append(c.Params, gin.Param{Key: "subtitle", Value: parts[1]})
			c.Params = append(c.Params, gin.Param{Key: "body", Value: parts[2]})
		default:
			// More than 3 parts, join remaining as body
			c.Params = append(c.Params, gin.Param{Key: "title", Value: parts[0]})
			if len(parts) > 2 {
				c.Params = append(c.Params, gin.Param{Key: "subtitle", Value: parts[1]})
				c.Params = append(c.Params, gin.Param{Key: "body", Value: strings.Join(parts[2:], "/")})
			} else {
				c.Params = append(c.Params, gin.Param{Key: "body", Value: strings.Join(parts[1:], "/")})
			}
		}
		
		h.HandleSimplePush(c)
	}
}
