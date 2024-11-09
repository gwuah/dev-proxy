package internal

import (
	"database/sql"

	"github.com/lopezator/migrator"
)

//	{
//		"method": "POST",
//		"protocol": "https",
//		"host": "echo.free.beeceptor.com",
//		"path": "/",
//		"ip": "86.86.6.48:56040",
//		"headers": {
//		  "Host": "echo.free.beeceptor.com",
//		  "User-Agent": "Go-http-client/1.1",
//		  "Content-Length": "15",
//		  "Accept-Encoding": "gzip",
//		  "Content-Type": "application/json"
//		},
//		"parsedQueryParams": {},
//		"parsedBody": {
//		  "key": "value"
//		}
//	  }

var (
	Migrations = migrator.Migrations(
		execsql(
			"create_rpc",
			`create table if not exists rpcs (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				payload TEXT,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);`,
		),
	)
)

func execsql(name, raw string) *migrator.MigrationNoTx {
	return &migrator.MigrationNoTx{
		Name: name,
		Func: func(db *sql.DB) error {
			_, err := db.Exec(raw)
			return err
		},
	}
}
