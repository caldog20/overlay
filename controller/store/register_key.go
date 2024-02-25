package store

import (
	"errors"

	"github.com/caldog20/overlay/controller/types"
	"gorm.io/gorm"
)

// CreateRegisterKey implements Store.
func (s *Store) CreateRegisterKey(key *types.RegisterKey) error {
	return s.db.Create(key).Error
}

// GetRegisterKey implements Store.
func (s *Store) GetRegisterKey(key string) (*types.RegisterKey, error) {
	var rkey types.RegisterKey

	if err := s.db.Where("key = ?", key).First(&rkey).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, types.ErrRegisterKeyNotFound
		}
		return nil, err
	}

	if rkey.Used {
		return nil, types.ErrRegisterKeyAlreadyUsed
	}

	return &rkey, nil
}

// GetRegisterKeys implements Store.
func (s *Store) GetRegisterKeys() ([]types.RegisterKey, error) {
	var keys []types.RegisterKey
	result := s.db.Find(&keys)
	if result.Error != nil {
		return nil, result.Error
	}
	return keys, nil
}

// UpdateRegisterKey implements Store.
func (s *Store) UpdateRegisterKey(key *types.RegisterKey) error {
	if err := s.db.Model(key).Updates(key).Error; err != nil {
		return err
	}
	return nil
}

func (s *Store) SetRegisterKeyUsed(key *types.RegisterKey) error {

	return nil
}
