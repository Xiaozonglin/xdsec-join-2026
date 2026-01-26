package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	jwtSecret       = []byte(os.Getenv("secret-key"))
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

// Claims 自定义声明
type Claims struct {
	UserUUID string `json:"user_uuid"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken 生成JWT Token（有效期7天）
func GenerateToken(userUUID, email, role string) (string, error) {
	// 设置7天有效期
	expireTime := time.Now().Add(7 * 24 * time.Hour)

	claims := &Claims{
		UserUUID: userUUID,
		Email:    email,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "xdsec-auth",
		},
	}

	// 使用HS256算法签名
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ParseToken 解析并验证Token
func ParseToken(tokenString string) (*Claims, error) {
	// 解析Token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	// 验证Token有效性
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

// GenerateCSRFToken 生成CSRF Token
func GenerateCSRFToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// GenerateEmailCode 生成6位邮箱验证码
func GenerateEmailCode() (string, error) {
	b := make([]byte, 3)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	code := fmt.Sprintf("%06x", b)
	return code[:6], nil
}

// HashPassword 使用bcrypt加密密码
func HashPassword(password string) (string, error) {
	// bcrypt.DefaultCost = 10，这个值可以在4-31之间，越大越安全但越慢
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPassword 验证密码
func CheckPassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// ParseUUIDString 安全地解析UUID字符串
func ParseUUIDString(s string) (uuid.UUID, error) {
	if s == "" {
		return uuid.Nil, errors.New("empty uuid string")
	}
	return uuid.Parse(s)
}
