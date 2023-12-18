package database

import (
	"context"
	"log"
	"os"

	//_ "github.com/lib/pq"
	//_ "github.com/jackc/pgx/v5/pgxpool"

	"github.com/jackc/pgx/v5"
	//_ "github.com/jackc/pgx/v5/stdlib"
)

func OpenDBConnection(connectionString string) *pgx.Conn {

	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))

	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	err = conn.Ping(context.Background())

	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}

	return conn
}
