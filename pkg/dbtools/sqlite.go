package dbtools

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

type SqliteConn struct {
	err error
	DB  *sql.DB
}

type SqliteRow struct {
	Id      string
	Content string
}

func NewSqliteConn(dbPath string) *SqliteConn {
	//os.Remove("./foo.db")
	db, err := sql.Open("sqlite3", dbPath)
	//defer db.Close()
	return &SqliteConn{err, db}
}

func (sc *SqliteConn) CreateTables() error {
	statement := `
	CREATE TABLE IF NOT EXISTS "train" (
		"id"	TEXT UNIQUE,
		"content"	TEXT,
		PRIMARY KEY("id")
	);
	`
	_, err := sc.DB.Exec(statement)
	if err != nil {
		log.Printf("%q: %s\n", err, statement)
		return err
	} else {
		return nil
	}
}

func (sc *SqliteConn) InsertData(chnl chan SqliteRow) error {
	tx, err := sc.DB.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("INSERT OR IGNORE INTO train (id, content) VALUES (?,?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	for v := range chnl {
		fmt.Println(v.Id)
		_, err = stmt.Exec(v.Id, v.Content)
		if err != nil {
			return err
		}
	}
	tx.Commit()
	return nil
}

func (sc *SqliteConn) GetTrainData(chnl chan string) error {
	query := `
		select content from train
	`

	rows, err := sc.DB.Query(query)
	if err != nil {
		close(chnl)
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var content string
		err = rows.Scan(&content)
		if err != nil {
			close(chnl)
			return err
		}
		chnl <- content
	}

	err = rows.Err()
	if err != nil {
		close(chnl)
		return err
	} else {
		close(chnl)
		return nil
	}
}
