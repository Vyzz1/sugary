package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	httpresponse "sugary/internal/delivery/http/response"
)

func JWT(secret string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if strings.TrimSpace(secret) == "" {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, httpresponse.Fail(ctx, "missing_jwt_secret", "jwt secret is not configured"))
			return
		}

		tokenString := bearerToken(ctx.GetHeader("Authorization"))
		if tokenString == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, httpresponse.Fail(ctx, "missing_token", "missing bearer token"))
			return
		}

		claims := &jwt.RegisteredClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, httpresponse.Fail(ctx, "invalid_token", "invalid token"))
			return
		}

		ctx.Set("auth_subject", claims.Subject)
		ctx.Next()
	}
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}

	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}
