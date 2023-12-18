package database

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
)

type StateRecord struct {
	Key   string
	Value string
}

func GetStateRecords(ctx context.Context, the_db *pgx.Conn) ([]StateRecord, error) {
	var ret []StateRecord

	rows, err := the_db.Query(ctx, "SELECT key, value FROM state;")
	if err != nil {
		return nil, fmt.Errorf("Error: query of state: %v", err)
	}
	defer rows.Close()

	// Loop through rows, using Scan to assign column data to struct fields.
	for rows.Next() {
		var state StateRecord
		if err := rows.Scan(&state.Key, &state.Value); err != nil {
			return nil, fmt.Errorf("Error state query Scan: %v", err)
		}
		ret = append(ret, state)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Error state processing of records: %v", err)
	}

	return ret, nil
}

func Delete(ctx context.Context, the_db *pgx.Conn, key string) error {

	log.Printf("DB:Delete Key = %s\n", key)
	res, err := the_db.Exec(ctx, "DELETE FROM state WHERE key = $1;", key)
	if err != nil {
		return fmt.Errorf("`Delete failed for state record with key %s: %v", key, err)
	}

	rowsAffected := res.RowsAffected()

	if rowsAffected != 1 {
		return fmt.Errorf("Wrong number of records deleted %v \n", rowsAffected)
	}

	return err
}
