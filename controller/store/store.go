package store

import (
	"fmt"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"github.com/caldog20/overlay/controller/types"
)

type Store struct {
	db *gorm.DB
}

func NewSqlStore(path string) (*Store, error) {
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

	schema.RegisterSerializer("addr", AddrSerializer{})
	schema.RegisterSerializer("addrport", AddrPortSerializer{})

	err = db.AutoMigrate(&types.Peer{}, &types.RegisterKey{}, &types.User{})
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}
