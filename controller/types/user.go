package types

import (
	"errors"
	"fmt"
	"net/mail"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	HashCost = 14
)

type User struct {
	gorm.Model
	Username     string `gorm:"uniqueIndex"`
	PasswordHash []byte
	Active       bool
}

func NewUser(username, password string) (*User, error) {
	err := validateUsername(username)
	if err != nil {
		return nil, err
	}

	err = validatePassword(password)
	if err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), HashCost)
	if err != nil {
		return nil, err
	}

	return &User{
		Username:     username,
		PasswordHash: hash,
		Active:       true,
	}, nil
}

// TODO: temporary fallback until Oauth/openid implemented
func (u *User) ComparePassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	if err != nil {
		return false
	}
	return true
}

func validateUsername(username string) error {
	if username == "admin" {
		return nil
	}

	_, err := mail.ParseAddress(username)
	if err != nil {
		return fmt.Errorf("invalid username: %s - must be valid email address", username)
	}
	return nil
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be a minimum of 8 characters long")
	}
	return nil
}
