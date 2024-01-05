package controller

import (
	"time"
)

type Peer struct {
	ID        uint32 `gorm:"primaryKey,autoIncrement"`
	IP        string `gorm:"uniqueIndex,not null"`
	PubKey    string `gorm:"uniqueIndex,not null"`
	EndPoint  string
	Connected bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Error string

func (e Error) Error() string { return string(e) }
