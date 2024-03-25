package database

import (
	"database/sql"
	"log"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/emoss08/trenova/ent"
	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver
)

var client *ent.Client

func GetClient() *ent.Client {
	return client
}

func SetClient(newClient *ent.Client) {
	client = newClient
}

func NewEntClient(dsn string) *ent.Client {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatal(err)
	}

	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(100)
	db.SetConnMaxLifetime(time.Hour)

	drv := entsql.OpenDB(dialect.Postgres, db)
	return ent.NewClient(ent.Driver(drv))
}

func Close() {
	if err := client.Close(); err != nil {
		log.Fatalf("failed closing client: %v", err)
	}
}
