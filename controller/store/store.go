package store

import (
	"fmt"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/caldog20/overlay/controller/types"
)

type Store struct {
	db *gorm.DB
}

func New(path string) (*Store, error) {
	db, err := gorm.Open(
		sqlite.Open(fmt.Sprintf("file:%s?cache=shared&_journal_mode=WAL", path)),
		&gorm.Config{
			PrepareStmt: true, Logger: logger.Default.LogMode(logger.Error),
		},
	)
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(3)

	err = sqlDB.Ping()
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&types.Peer{}, &types.User{})

	return &Store{db: db}, nil
}
