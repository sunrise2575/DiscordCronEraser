package main

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func dbConnect() *sql.DB {
	db, e := sql.Open("sqlite3", "file::memory:?mode=memory&cache=shared")
	if e != nil {
		log.Fatal(e)
	}
	return db
}

func dbExec(db *sql.DB, query string, parameters ...interface{}) int {
	result, e := db.Exec(query, parameters...)
	if e != nil {
		log.Printf("dbExec error: %v, query: %v", e, query)
		return -1
	}

	affected, e := result.RowsAffected()
	if e != nil {
		log.Printf("dbExec error: %v, query: %v", e, query)
		return -1
	}

	return int(affected)
}

func dbQuery(db *sql.DB, query string, parameters ...interface{}) [][]string {
	result := [][]string{}

	rows, e := db.Query(query, parameters...)
	if e != nil {
		log.Printf("dbQuery error: %v, query: %v", e, query)
		return nil
	}
	defer rows.Close()

	columns, e := rows.Columns()
	if e != nil {
		log.Printf("dbQuery error: %v, query: %v", e, query)
		return nil
	}

	tempBytePtr := make([]interface{}, len(columns))
	tempByte := make([][]byte, len(columns))
	tempString := make([]string, len(columns))
	for i := range tempByte {
		tempBytePtr[i] = &tempByte[i]
	}

	for rows.Next() {
		if e := rows.Scan(tempBytePtr...); e != nil {
			log.Printf("dbQuery error: %v, query: %v", e, query)
			return nil
		}

		for i, rawByte := range tempByte {
			if rawByte == nil {
				tempString[i] = "\\N"
			} else {
				tempString[i] = string(rawByte)
			}
		}

		result = append(result, make([]string, len(columns)))
		copy(result[len(result)-1], tempString)
	}

	return result
}

func dbTx(db *sql.DB, procedure func(*sql.Tx) bool) bool {
	tx, e := db.Begin()
	if e != nil {
		log.Printf("dbTx Begin error: %v", e)
		return false
	}

	if procedure(tx) {
		e := tx.Commit()
		if e != nil {
			log.Printf("dbTx Commit error: %v", e)
			return false
		}
		return true
	} else {
		e := tx.Rollback()
		if e != nil {
			log.Printf("dbTx Rollback error: %v", e)
			return false
		}
		return false
	}
}

func dbTxExec(tx *sql.Tx, query string, parameters ...interface{}) int {
	result, e := tx.Exec(query, parameters...)
	if e != nil {
		log.Printf("dbTxExec error: %v, query: %v", e, query)
		return -1
	}

	affected, e := result.RowsAffected()
	if e != nil {
		log.Printf("dbTxExec error: %v, query: %v", e, query)
		return -1
	}

	return int(affected)
}

func dbTxQuery(tx *sql.Tx, query string, parameters ...interface{}) [][]string {
	result := [][]string{}

	rows, e := tx.Query(query, parameters...)
	if e != nil {
		log.Printf("dbTxQuery error: %v, query: %v", e, query)
		return nil
	}
	defer rows.Close()

	columns, e := rows.Columns()
	if e != nil {
		log.Printf("dbTxQuery error: %v, query: %v", e, query)
		return nil
	}

	tempBytePtr := make([]interface{}, len(columns))
	tempByte := make([][]byte, len(columns))
	tempString := make([]string, len(columns))
	for i := range tempByte {
		tempBytePtr[i] = &tempByte[i]
	}

	for rows.Next() {
		if e := rows.Scan(tempBytePtr...); e != nil {
			log.Printf("dbTxQuery error: %v, query: %v", e, query)
			return nil
		}

		for i, rawByte := range tempByte {
			if rawByte == nil {
				tempString[i] = "\\N"
			} else {
				tempString[i] = string(rawByte)
			}
		}

		result = append(result, make([]string, len(columns)))
		copy(result[len(result)-1], tempString)
	}

	return result
}
