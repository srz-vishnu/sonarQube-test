package jwt

import (
	"errors"
	"time"

	"github.com/dgrijalva/jwt-go"
)

var (
	ErrExpiredToken = errors.New("token is expired")
	jwtKey          = []byte("wA2I7VqLMbKP5RtUoD7M1jsJYWD9edxBS6cOgFXElwo=")
)

type Claims struct {
	UserID   int64  `json:"id"` //here we including id and name here so that will be there on the token, so we can use it in the other layers
	Username string `json:"username"`
	IsAdmin  bool   `json:"isadmin"`
	jwt.StandardClaims
}

// JWTService defines the interface for JWT operations
// (the go:generate line is optional, for mocking tools)
//
//go:generate mockgen -destination=mock_jwtservice.go -package=jwt . JWTService
type JWTService interface {
	GenerateToken(userID int64, username string, isadmin bool) (string, error)
	ValidateToken(tokenStr string) (*Claims, error)
}

// JWTServiceImpl is the concrete implementation of JWTService
// It holds the signing key
type JWTServiceImpl struct {
	key []byte
}

// NewJWTService creates a new JWTService with the default key
func NewJWTService() JWTService {
	return &JWTServiceImpl{key: jwtKey}
}

// GenerateToken generates a new JWT token
func (j *JWTServiceImpl) GenerateToken(userID int64, username string, isadmin bool) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour) // Token valid for 24 hours
	claims := &Claims{
		UserID:   userID,
		Username: username,
		IsAdmin:  isadmin,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(j.key)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

// ValidateToken validates the JWT token and checks for expiration
func (j *JWTServiceImpl) ValidateToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return j.key, nil
	})

	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			return nil, err
		}
		return nil, err
	}

	if !token.Valid {
		return nil, err
	}

	// Check if the token has expired
	if claims.ExpiresAt < time.Now().Unix() {
		return nil, ErrExpiredToken
	}

	return claims, nil
}
