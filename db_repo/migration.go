package dbrepo

import (
	"database/sql"
	"log"
	"os"
)

func CreateInitialSchema(db *sql.DB) {
	createTableSql := `
  create table keystore (id integer not null primary key, userId text not null unique, key text not null, iv text not null)`
	createIndexSql := `create unique index idx_keystore_userId ON keystore(userId)`
	_, err := db.Exec(createTableSql)
	if err != nil {
		log.Printf("%q: %s\n", err, createTableSql)
		os.Exit(1)
	}
	_, err = db.Exec(createIndexSql)
	if err != nil {
		log.Printf("%q: %s\n", err, createIndexSql)
		os.Exit(1)
	}
}
