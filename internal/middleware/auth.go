package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type Claims struct {
	BusinessID uint   `json:"business_id"`
	Email      string `json:"email"`
	jwt.RegisteredClaims
}

var jwtSecret []byte
var db *gorm.DB

func SetJWTSecret(secret string) {
	jwtSecret = []byte(secret)
}

func SetDB(d *gorm.DB) {
	db = d
}

func GenerateToken(businessID uint, email string, expireHours int) (string, error) {
	claims := Claims{
		BusinessID: businessID,
		Email:      email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expireHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(parts[1], claims, func(t *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		// Business DB'de hâlâ mevcut mu kontrol et
		if db != nil {
			var count int64
			db.Table("businesses").Where("id = ? AND status = 'active' AND deleted_at IS NULL", claims.BusinessID).Count(&count)
			if count == 0 {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session expired, please login again"})
				return
			}
		}

		c.Set("business_id", claims.BusinessID)
		c.Set("email", claims.Email)
		c.Next()
	}
}

func GetBusinessID(c *gin.Context) uint {
	id, _ := c.Get("business_id")
	if bid, ok := id.(uint); ok {
		return bid
	}
	return 0
}
