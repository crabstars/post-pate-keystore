package dbrepo

import "database/sql"

func DelteUser(db *sql.DB, userId string) (int64, error) {
	sql := `delete from keystore where userId = ?`
	result, err := db.Exec(sql, userId)
	if err != nil {
		return 0, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return count, err
}
