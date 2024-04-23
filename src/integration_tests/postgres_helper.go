package integration_tests

import (
	"block_producers_uptime/delegation_backend"
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func StartPostgresContainerAndSetupSchema(config delegation_backend.PostgreSQLConfig) (*sql.DB, error) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres",
		Name:         "postgres_integration",
		ExposedPorts: []string{fmt.Sprintf("%d/tcp", config.Port)},
		Env: map[string]string{
			"POSTGRES_DB":       config.DBName,
			"POSTGRES_USER":     config.User,
			"POSTGRES_PASSWORD": config.Password,
		},
		WaitingFor:  wait.ForListeningPort("5432/tcp"),
		NetworkMode: "integration-test_default",
	}

	postgresContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start PostgreSQL container: %v", err)
	}

	// Get the dynamic port mapped to the PostgreSQL server
	mappedPort, err := postgresContainer.MappedPort(ctx, "5432")
	if err != nil {
		return nil, fmt.Errorf("failed to get mapped port: %v", err)
	}

	// Build the connection string to connect to the dynamically started PostgreSQL
	connStr := fmt.Sprintf("host=localhost port=%s user=%s password=%s dbname=%s sslmode=disable",
		mappedPort.Port(), config.User, config.Password, config.DBName)

	// Wait for the container to be ready and establish a database connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the database: %v", err)
	}

	submissions_schema := `CREATE TABLE IF NOT EXISTS submissions (
		id SERIAL PRIMARY KEY,
		submitted_at_date DATE NOT NULL,
		submitted_at TIMESTAMP NOT NULL,
		submitter TEXT NOT NULL,
		created_at TIMESTAMP,
		block_hash TEXT,
		remote_addr TEXT,
		peer_id TEXT,
		snark_work BYTEA,
		graphql_control_port INT,
		built_with_commit_sha TEXT,
		state_hash TEXT,
		parent TEXT,
		height INTEGER,
		slot INTEGER,
		validation_error TEXT,
		verified BOOLEAN
	);`

	if _, err = db.Exec(submissions_schema); err != nil {
		return nil, fmt.Errorf("failed to execute SQL script: %v", err)
	}

	return db, nil
}

func WaitUntilPostgresHasSubmissions(db *sql.DB) error {
	log.Println("Waiting for submissions to appear in PostgreSQL")

	// Set up timeout logic
	timeout := time.After(TIMEOUT_IN_S * time.Second)
	tick := time.Tick(5 * time.Second)

	query := `SELECT COUNT(*) FROM submissions;`

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout reached while waiting for submissions in PostgreSQL")
		case <-tick:
			// Check for submissions
			var count int
			err := db.QueryRow(query).Scan(&count)
			if err != nil {
				return fmt.Errorf("error querying submissions count: %w", err)
			}
			if count > 0 {
				log.Println("Found submissions in PostgreSQL")
				return nil
			}
		}
	}
}
