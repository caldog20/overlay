package types

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RegisterKey struct {
	gorm.Model
	Key      string `gorm:"uniqueIndex,not null"`
	Valid    bool
	Reusable bool
}

func NewRegisterKey() *RegisterKey {
	key := uuid.New()
	return &RegisterKey{
		Key: key.String(),
	}
}

func (rk *RegisterKey) Compare(key string) bool {
	if rk.Key == key {
		return true
	}
	return false
}
