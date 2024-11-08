package internal

import (
	"database/sql"

	"github.com/lopezator/migrator"
	_ "github.com/mattn/go-sqlite3"
)

const (
	MAX_CONNS = 10
)

func ConnectToDB(url string, opts ...migrator.Option) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", url)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(MAX_CONNS)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	m, err := migrator.New(opts...)
	if err != nil {
		return nil, err
	}
	if err := m.Migrate(db); err != nil {
		return nil, err
	}

	return db, nil
}
