package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"sugary/internal/config"
	httpresponse "sugary/internal/delivery/http/response"
)

type AuthHandler struct {
	auth config.AuthConfig
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func NewAuthHandler(auth config.AuthConfig) AuthHandler {
	return AuthHandler{
		auth: auth,
	}
}

func (h AuthHandler) Login(ctx *gin.Context) {
	var request loginRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "bad_request", err.Error()))
		return
	}

	request.Username = strings.TrimSpace(request.Username)
	request.Password = strings.TrimSpace(request.Password)
	if request.Username == "" || request.Password == "" {
		ctx.JSON(http.StatusBadRequest, httpresponse.Fail(ctx, "missing_credentials", "username and password are required"))
		return
	}

	if request.Username != h.auth.LoginUser || request.Password != h.auth.LoginPassword {
		ctx.JSON(http.StatusUnauthorized, httpresponse.Fail(ctx, "invalid_credentials", "invalid credentials"))
		return
	}

	expiresIn, err := time.ParseDuration(h.auth.JWTExpiresIn)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, httpresponse.Fail(ctx, "invalid_config", "invalid jwt expiry configuration"))
		return
	}

	now := time.Now().UTC()
	claims := jwt.RegisteredClaims{
		Subject:   request.Username,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(expiresIn)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(h.auth.JWTSecret))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, httpresponse.Fail(ctx, "token_sign_failed", "failed to sign token"))
		return
	}

	ctx.JSON(http.StatusOK, httpresponse.OK(ctx, gin.H{
		"access_token": signed,
		"token_type":   "Bearer",
		"expires_in":   h.auth.JWTExpiresIn,
	}))
}
