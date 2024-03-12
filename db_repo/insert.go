package dbrepo

import "database/sql"

func InsertUserAndKey(db *sql.DB, user UserEntry) error {
	sql := `insert into keystore (userId, key, iv) values (?, ?, ?)`
	_, err := db.Exec(sql, user.UserId, user.Key, user.Iv)
	return err
}
