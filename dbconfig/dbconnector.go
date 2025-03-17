package dbconfig

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ConnectToDB establishes a connection to PostgreSQL using credentials from dbreader.go
func ConnectToDB(credentials *DatabaseCredentials) (*pgx.Conn, error) {
	// Format connection string
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		credentials.User,
		credentials.Password,
		credentials.Host,
		credentials.Port,
		credentials.Name,
	)

	// Connect to PostgreSQL
	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %v", err)
	}

	fmt.Println("✅ Successfully connected to PostgreSQL!")
	return conn, nil
}

// CheckDBRole checks if the PostgreSQL database is in recovery mode (standby) or primary
func CheckDBRole(conn *pgx.Conn) (string, error) {
	var isInRecovery bool

	// Run the pg_is_in_recovery() query
	err := conn.QueryRow(context.Background(), "SELECT pg_is_in_recovery();").Scan(&isInRecovery)
	if err != nil {
		return "", fmt.Errorf("error checking database role: %v", err)
	}

	// Determine the role
	if isInRecovery {
		return "Standby", nil
	}
	return "Primary", nil
}
