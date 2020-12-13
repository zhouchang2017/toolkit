package utils

import "database/sql"

func Transaction(db *sql.DB, handle func(tx *sql.Tx) error) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()
	if err = handle(tx); err != nil {
		return err
	}
	return tx.Commit()
}
