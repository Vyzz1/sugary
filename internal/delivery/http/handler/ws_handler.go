package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"sugary/internal/platform/hub"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// CheckOrigin validates the request origin. CORS for WS is handled here
	// rather than via the HTTP CORS middleware (which doesn't apply after upgrade).
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for MVP; tighten in production.
		return true
	},
}

// WSHandler upgrades HTTP connections to WebSocket and registers clients with
// the Hub so they can receive async AI analysis push notifications.
type WSHandler struct {
	hub       *hub.Hub
	jwtSecret string
}

// NewWSHandler creates a WSHandler.
func NewWSHandler(h *hub.Hub, jwtSecret string) WSHandler {
	return WSHandler{hub: h, jwtSecret: jwtSecret}
}

// Connect godoc
//
// GET /ws?token=<jwt>
//
// Upgrades to a WebSocket connection. Clients receive push notifications when
// AI meal analysis completes or fails.
//
// Message envelope (server → client):
//
//	{
//	  "type":   "meal_analysis",
//	  "status": "completed" | "failed",
//	  "data":   { ...meal... },          // present on completed
//	  "error":  { "code", "message", "meal_id" } // present on failed
//	}
func (h WSHandler) Connect(c *gin.Context) {
	// Validate JWT from query param — standard HTTP headers are not
	// available after the WebSocket handshake from browser clients.
	tokenString := strings.TrimSpace(c.Query("token"))
	if tokenString == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   gin.H{"code": "missing_token", "message": "missing token query param"},
		})
		return
	}

	if strings.TrimSpace(h.jwtSecret) == "" {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   gin.H{"code": "missing_jwt_secret", "message": "jwt secret is not configured"},
		})
		return
	}

	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		return []byte(h.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   gin.H{"code": "invalid_token", "message": "invalid token"},
		})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		zap.L().Warn("ws_upgrade_failed", zap.Error(err))
		return
	}

	client := hub.NewClient(h.hub, conn)
	h.hub.Register(client)

	// Read pump: keep the connection alive and honour browser close frames.
	// We don't process inbound messages (server-push only), but we must drain
	// the read side so that control frames (ping/pong/close) are handled.
	go func() {
		defer h.hub.Unregister(client)
		conn.SetReadDeadline(time.Time{}) // no read deadline — long-lived connection
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err,
					websocket.CloseGoingAway,
					websocket.CloseNormalClosure,
				) {
					zap.L().Warn("ws_read_error", zap.Error(err))
				}
				return
			}
		}
	}()
}
