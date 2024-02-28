package store

import (
	"errors"
	"fmt"

	"github.com/caldog20/overlay/controller/types"
	"gorm.io/gorm"
)

func (s *Store) CreateUser(user *types.User) error {
	if err := s.db.Create(user).Error; err != nil {
		return fmt.Errorf("error creating user: %w", err)
	}
	return nil
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
