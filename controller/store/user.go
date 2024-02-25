package store

import (
	"errors"

	"github.com/caldog20/overlay/controller/types"
	"gorm.io/gorm"
)

func (s *Store) CreateUser(user *types.User) error {
	return s.db.Create(user).Error
}

func (s *Store) GetUser(username string) (*types.User, error) {
	var u types.User
	if err := s.db.Where("username = ?", username).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, types.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}
