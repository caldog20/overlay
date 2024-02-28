package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrTokenNotValid = errors.New("jwt token is no longer valid")
	ErrParsingToken  = errors.New("error parsing jwt token")
)

var jwtKey = []byte("SUPERSECRETKEY")

type Claims struct {
	User string
	jwt.RegisteredClaims
}

func GenerateToken(username string) (string, error) {
	claims := &Claims{
		username,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	t, err := token.SignedString(jwtKey)
	if err != nil {
		return "", err
	}
	return t, nil
}

func ValidateToken(token string) (string, error) {
	claims := &Claims{}
	t, err := jwt.ParseWithClaims(token, claims, keyFunc)
	if err != nil {
		return "", ErrParsingToken
	}

	if !t.Valid {
		return "", ErrTokenNotValid
	}

	return claims.User, nil

}

func keyFunc(token *jwt.Token) (interface{}, error) {
	return jwtKey, nil
}
