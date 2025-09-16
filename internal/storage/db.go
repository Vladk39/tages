package storage

import (
	"Tages/internal/dto"
	"context"
	"database/sql"
	"time"

	"github.com/pkg/errors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const ExtensionForPsg = `create extension if not exists "uuid-ossp"`

// инкапсулируем логику создания подключения
func newGorm(dsn string) (*gorm.DB, error) {
	connect, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true,
	}))
	if err != nil {
		return nil, errors.Wrap(err, "error in create db")
	}
	return connect, err
}

type Storage struct {
	conn *gorm.DB
}

func NewStorage(dsn string) (*Storage, error) {
	const (
		maxRetries = 10
		delay      = 1 * time.Second
	)

	var conn *gorm.DB
	var err error

	for i := 0; i < maxRetries; i++ {
		conn, err = newGorm(dsn)
		if err == nil {
			return &Storage{conn: conn}, nil
		}
		time.Sleep(delay)
	}

	return nil, errors.Wrap(err, "cannot connect postgresql")
}

func NewStorageWithDB(db *sql.DB) (*Storage, error) {
	conn, err := gorm.Open(postgres.New(postgres.Config{
		Conn:                 db,
		PreferSimpleProtocol: true,
	}))
	if err != nil {
		return nil, errors.Wrap(err, "error in create db")
	}

	return &Storage{
		conn: conn,
	}, nil
}

// migrator
func (s *Storage) Init(ctx context.Context) error {
	if err := s.conn.WithContext(ctx).Exec(ExtensionForPsg).Error; err != nil {
		return errors.Wrap(err, "cant create extension for postgres")
	}

	return s.conn.WithContext(ctx).AutoMigrate(&dto.File{})
}

// drop table
func (s *Storage) Drop() error {
	return s.conn.Migrator().DropTable(&dto.File{})
}

func (s *Storage) WithInTransaction(
	ctx context.Context,
	tFunc func(ctx context.Context) error,
) error {
	tx := s.conn.Begin()

	err := tFunc(injectTx(ctx, tx))
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()

	return nil
}

func injectTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, tx.Callback(), tx)
}

func (s *Storage) Close(ctx context.Context) error {
	sql, err := s.conn.DB()
	if err != nil {
		return errors.Wrap(err, "failed to get sql.DB from gorm.DB")
	}

	return sql.Close()
}
