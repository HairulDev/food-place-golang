// db/db.go

package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

const (
	host      = "localhost"
	port      = 55000
	user      = "postgres"
	password  = "postgrespw"
	dbname    = "postgres"
	uploadDir = "./uploads"
)

func GetDBConnection() *sql.DB {
	// Buat string koneksi untuk database PostgreSQL
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	// Membuka koneksi ke database
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}

	// Menguji koneksi ke database
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	return db
}
