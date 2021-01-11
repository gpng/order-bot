package postgres

import (
	"database/sql"
	"fmt"
	"log"

	// postgres drivers
	_ "github.com/lib/pq"

	_ "github.com/jinzhu/gorm/dialects/postgres" // required for postgres dbs
)

// New db connection and trigger migrations
func New(dbHost string, dbUser string, dbName string, dbPassword string) (*sql.DB, error) {
	// connection string
	dbURI := fmt.Sprintf("host=%s user=%s dbname=%s sslmode=disable password=%s", dbHost, dbUser, dbName, dbPassword)
	log.Printf("db connection string: %s", dbURI)

	db, err := sql.Open("postgres", dbURI)
	if err != nil {
		log.Printf("failed to connect to db: %v", err)
		return nil, err
	}

	if err = db.Ping(); err != nil {
		log.Printf("failed to ping db: %v", err)
		return nil, err
	}

	log.Println("db connection successful")

	return db, nil
}
