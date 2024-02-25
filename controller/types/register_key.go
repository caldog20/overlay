package types

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RegisterKey struct {
	gorm.Model
	Key  string `gorm:"uniqueIndex,not null"`
	Used bool
	User string
}

func NewRegisterKey(user string) *RegisterKey {
	key := uuid.New()
	return &RegisterKey{
		Key:  key.String(),
		Used: true,
		User: user,
	}
}

func (rk *RegisterKey) Compare(key string) bool {
	if rk.Key == key {
		return true
	}
	return false
}
