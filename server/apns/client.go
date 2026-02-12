package apns

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	// APNsProduction URL
	APNsProduction = "https://api.push.apple.com"
	// APNsDevelopment URL
	APNsDevelopment = "https://api.development.push.apple.com"
	// Topic for Bark app
	Topic = "me.fin.bark"
)

// Client represents an APNs client
type Client struct {
	httpClient *http.Client
	keyID      string
	teamID     string
	privateKey *ecdsa.PrivateKey
	topic      string
	endpoint   string

	// Token caching
	token      string
	tokenMu    sync.RWMutex
	tokenExp   time.Time
}

// Payload represents APNs payload
type Payload struct {
	Aps Aps `json:"aps"`
	// Bark extension fields
	Group     string `json:"group,omitempty"`
	Call      bool   `json:"call,omitempty"`
	IsArchive bool   `json:"isarchive,omitempty"`
	Icon      string `json:"icon,omitempty"`
	Cipher    string `json:"ciphertext,omitempty"`
	Level     string `json:"level,omitempty"`
	Volume    int    `json:"volume,omitempty"`
	URL       string `json:"url,omitempty"`
	Copy      bool   `json:"copy,omitempty"`
	Badge     int    `json:"badge,omitempty"`
	AutoCopy  bool   `json:"autocopy,omitempty"`
	Action    string `json:"action,omitempty"`
	IV        string `json:"iv,omitempty"`
	Image     string `json:"image,omitempty"`
	ID        string `json:"id,omitempty"`
	Delete    bool   `json:"delete,omitempty"`
	Markdown  string `json:"markdown,omitempty"`
}

// Aps represents the aps dictionary
type Aps struct {
	Alert            interface{} `json:"alert,omitempty"`
	Badge            *int        `json:"badge,omitempty"`
	Sound            string      `json:"sound,omitempty"`
	ThreadID         string      `json:"thread-id,omitempty"`
	Category         string      `json:"category,omitempty"`
	ContentAvailable int         `json:"content-available,omitempty"`
	MutableContent   int         `json:"mutable-content,omitempty"`
}

// Alert represents alert dictionary
type Alert struct {
	Title        string `json:"title,omitempty"`
	Subtitle     string `json:"subtitle,omitempty"`
	Body         string `json:"body,omitempty"`
	LaunchImage  string `json:"launch-image,omitempty"`
	TitleLocKey  string `json:"title-loc-key,omitempty"`
	TitleLocArgs string `json:"title-loc-args,omitempty"`
	LocKey       string `json:"loc-key,omitempty"`
	LocArgs      string `json:"loc-args,omitempty"`
}

// Response represents APNs response
type Response struct {
	StatusCode int
	Reason     string
	ApnsID     string
}

// NewClient creates a new APNs client
func NewClient(keyID, teamID, privateKeyPEM string, production bool) (*Client, error) {
	privateKey, err := parsePrivateKey(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	endpoint := APNsProduction
	if !production {
		endpoint = APNsDevelopment
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				ForceAttemptHTTP2: true,
			},
		},
		keyID:      keyID,
		teamID:     teamID,
		privateKey: privateKey,
		topic:      Topic,
		endpoint:   endpoint,
	}, nil
}

// parsePrivateKey parses PEM-encoded ECDSA private key
func parsePrivateKey(pemKey string) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemKey))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		key, err = x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
	}

	privateKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not an ECDSA private key")
	}

	return privateKey, nil
}

// Push sends a push notification
func (c *Client) Push(deviceToken string, payload *Payload, headers map[string]string) (*Response, error) {
	url := fmt.Sprintf("%s/3/device/%s", c.endpoint, deviceToken)

	// Marshal payload
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apns-topic", c.topic)
	req.Header.Set("apns-push-type", "alert")
	if payload.Delete {
		req.Header.Set("apns-push-type", "background")
	}

	// Set custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Generate JWT token
	token, err := c.generateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}
	req.Header.Set("authorization", fmt.Sprintf("bearer %s", token))

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	apnsResp := &Response{
		StatusCode: resp.StatusCode,
		ApnsID:     resp.Header.Get("apns-id"),
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return apnsResp, nil
	}

	var errResp struct {
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal(respBody, &errResp); err == nil {
		apnsResp.Reason = errResp.Reason
	}

	return apnsResp, nil
}

// generateToken generates a JWT token for APNs authentication with caching
func (c *Client) generateToken() (string, error) {
	// Check if cached token is still valid (reuse for 50 minutes)
	c.tokenMu.RLock()
	if c.token != "" && time.Now().Before(c.tokenExp) {
		token := c.token
		c.tokenMu.RUnlock()
		return token, nil
	}
	c.tokenMu.RUnlock()

	// Generate new token
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	// Double check after acquiring write lock
	if c.token != "" && time.Now().Before(c.tokenExp) {
		return c.token, nil
	}

	// Simple JWT generation
	// Header
	header := map[string]interface{}{
		"alg": "ES256",
		"kid": c.keyID,
	}
	headerJSON, _ := json.Marshal(header)
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)

	// Claims
	claims := map[string]interface{}{
		"iss": c.teamID,
		"iat": time.Now().Unix(),
	}
	claimsJSON, _ := json.Marshal(claims)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	// Sign
	signingInput := fmt.Sprintf("%s.%s", headerB64, claimsB64)
	signature, err := c.sign([]byte(signingInput))
	if err != nil {
		return "", err
	}
	signatureB64 := base64.RawURLEncoding.EncodeToString(signature)

	c.token = fmt.Sprintf("%s.%s.%s", headerB64, claimsB64, signatureB64)
	c.tokenExp = time.Now().Add(50 * time.Minute) // Cache for 50 minutes

	return c.token, nil
}

// sign signs the data using ECDSA with SHA256
func (c *Client) sign(data []byte) ([]byte, error) {
	// SHA256 hash
	hash := sha256.Sum256(data)

	// ECDSA sign with crypto/rand
	r, s, err := ecdsa.Sign(rand.Reader, c.privateKey, hash[:])
	if err != nil {
		return nil, err
	}

	// Convert r and s to fixed-size 32-byte arrays (P-256 curve)
	// The signature for ES256 is the concatenation of r and s, each 32 bytes
	sig := make([]byte, 64)
	rBytes := r.Bytes()
	sBytes := s.Bytes()

	// Right-pad with zeros if needed (big-endian)
	copy(sig[32-len(rBytes):32], rBytes)
	copy(sig[64-len(sBytes):64], sBytes)

	return sig, nil
}
