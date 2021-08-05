package main

import (
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

type connInfoType struct {
	id, pw        string
	address, port string
	db, table     string
}

func san(in string) string {
	return "`" + in + "`"
}

func connectMySQL(connInfo connInfoType) *sql.DB {
	connStr := connInfo.id + ":" +
		connInfo.pw + "@tcp(" +
		connInfo.address + ":" +
		connInfo.port + ")/" + connInfo.db
	db, e := sql.Open("mysql", connStr)
	if e != nil {
		panic(e)
	}

	return db
}

func initDB(connInfo connInfoType) {
	db := connectMySQL(connInfo)
	defer db.Close()

	exist := false

	// Check if table exists
	{

		result, e := db.Query(`SHOW TABLES`)
		if e != nil {
			panic(e)
		}

		tableName := ""
		for result.Next() {
			if e := result.Scan(&tableName); e != nil {
				panic(e)
			}
			if tableName == connInfo.table {
				log.Println("exist", tableName, connInfo.table)
				exist = true
				break
			}
		}
	}

	// If table exists, remove old table
	if exist {
		_, e := db.Exec("DROP TABLE " + san(connInfo.table))
		if e != nil {
			panic(e)
		}
	}

	// create new table anyway
	{
		_, e := db.Exec(`
			CREATE TABLE ` + san(connInfo.table) + ` (
				channel_id bigint not null,
				author_id bigint not null,
				timestamp datetime(6) not null,
				message_id bigint not null,
				unique key (message_id),
				INDEX chan_idx (channel_id),
				INDEX chan_auth_idx (channel_id, author_id)
			)`)
		if e != nil {
			panic(e)
		}
	}
}
