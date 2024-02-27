package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtKey []byte

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
		return "", err
	}

	if !t.Valid {
		return "", errors.New("invalid jwt token")
	}

	return claims.User, nil

}

func keyFunc(token *jwt.Token) (interface{}, error) {
	return jwtKey, nil
}
