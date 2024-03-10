package dbrepo

import "database/sql"

func UserExists(db *sql.DB, userId string) (bool, error) {
	sql := `select exists(select 1 from keystore where userId = ?)`
	var exists bool
	err := db.QueryRow(sql, userId).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}
