package dbrepo

import (
	"database/sql"
)

func HealthCheck(db *sql.DB) error {
	sql := `PRAGMA integrity_check`
	rows, err := db.Query(sql)
	if err != nil {
		return err
	}
	defer rows.Close()

	var result string
	for rows.Next() {
		err := rows.Scan(&result)
		if err != nil {
			return err
		}
		//fmt.Println(result)
	}
	return rows.Err()
}
