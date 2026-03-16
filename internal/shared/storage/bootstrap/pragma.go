package bootstrap

import (
	"log"

	"codeswitch/internal/shared/storage"

	"github.com/daodao97/xgo/xdb"
)

func (i *Initializer) configurePragmas() {
	configureSQLitePragmas(storage.CoreDBName)
	configureSQLitePragmas(storage.RequestLogDBName)
	configureSQLitePragmas(storage.SessionDBName)
}

func configureSQLitePragmas(dbName string) {
	db, err := xdb.DB(dbName)
	if err != nil {
		return
	}
	for _, stmt := range []string{
		"PRAGMA synchronous = NORMAL",
		"PRAGMA journal_size_limit = 67108864",
		"PRAGMA wal_autocheckpoint = 1000",
	} {
		if _, err := db.Exec(stmt); err != nil {
			log.Printf("设置 SQLite PRAGMA 失败 (%s): %v\n", stmt, err)
		}
	}
}
