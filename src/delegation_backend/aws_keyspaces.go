package delegation_backend

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sigv4-auth-cassandra-gocql-driver-plugin/sigv4"
	"github.com/gocql/gocql"
	"github.com/golang-migrate/migrate/v4"
	cassandra "github.com/golang-migrate/migrate/v4/database/cassandra"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Operation is a function type that represents an operation that might fail and need a retry.
type Operation func() error

const (
	maxRetries     = 20
	initialBackoff = 1 * time.Second
)

// ExponentialBackoff retries the provided operation with an exponential backoff strategy.
func ExponentialBackoff(operation Operation, maxRetries int, initialBackoff time.Duration) error {
	backoff := initialBackoff
	var err error
	for i := 0; i < maxRetries; i++ {
		err = operation()
		if err == nil {
			return nil // Success
		}

		if i < maxRetries-1 {
			// If not the last retry, wait for a bit
			time.Sleep(backoff)
			backoff *= 2 // Exponential increase
		}
	}

	return fmt.Errorf("operation failed after %d retries, returned error: %s", maxRetries, err)
}

// InitializeKeyspaceSession creates a new gocql session for Amazon Keyspaces using the provided configuration.
func InitializeKeyspaceSession(config *AwsKeyspacesConfig) (*gocql.Session, error) {
	auth := sigv4.NewAwsAuthenticator()
	auth.AccessKeyId = config.AccessKeyId
	auth.SecretAccessKey = config.SecretAccessKey
	auth.Region = config.Region

	// Create a SigV4 gocql cluster config
	endpoint := "cassandra." + config.Region + ".amazonaws.com"
	cluster := gocql.NewCluster(endpoint)
	cluster.Keyspace = config.Keyspace
	cluster.Port = 9142
	cluster.Authenticator = auth
	cluster.SslOpts = &gocql.SslOptions{
		CaPath:                 config.SSLCertificatePath,
		EnableHostVerification: false,
	}

	cluster.Consistency = gocql.LocalQuorum
	cluster.DisableInitialHostLookup = false

	// Create a SigV4 gocql session
	session, err := cluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("could not create Cassandra session: %w", err)
	}

	return session, nil
}

func WaitForTablesActive(config *AwsKeyspacesConfig, tables []string) error {
	log.Printf("Waiting for tables %v to be active...", tables)
	operation := func() error {
		for _, tableName := range tables {
			session, err := InitializeKeyspaceSession(config)
			if err != nil {
				return err
			}
			defer session.Close()

			// Check if the table exists
			query := fmt.Sprintf("SELECT * FROM %s.%s LIMIT 1", config.Keyspace, tableName)
			if err := session.Query(query).Consistency(gocql.One).Exec(); err != nil {
				return err
			}

		}
		return nil
	}

	return ExponentialBackoff(operation, maxRetries, initialBackoff)
}

func createSchemaMigrationsTableIfNotExists(session *gocql.Session, keyspace string) error {
	query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.schema_migrations (version bigint PRIMARY KEY, dirty boolean);`, keyspace)
	operation := func() error {
		return session.Query(query).Exec()
	}

	return ExponentialBackoff(operation, maxRetries, initialBackoff)
}

func DropAllTables(config *AwsKeyspacesConfig) error {
	log.Print("Dropping all tables...")
	operation := func() error {
		session, err := InitializeKeyspaceSession(config)
		if err != nil {
			return fmt.Errorf("could not initialize Cassandra session: %w", err)
		}
		query := fmt.Sprintf(`SELECT table_name FROM system_schema.tables WHERE keyspace_name = '%s';`, config.Keyspace)
		iter := session.Query(query).Iter()
		var tableName string
		for iter.Scan(&tableName) {
			query = fmt.Sprintf(`DROP TABLE %s.%s;`, config.Keyspace, tableName)
			err := session.Query(query).Exec()
			if err != nil {
				return fmt.Errorf("could not drop table %s: %w", tableName, err)
			}
		}
		if err := iter.Close(); err != nil {
			return fmt.Errorf("could not close iterator: %w", err)
		}

		return nil
	}

	return ExponentialBackoff(operation, maxRetries, initialBackoff)

}

// MigrationUp applies all up migrations.
func MigrationUp(config *AwsKeyspacesConfig, migrationPath string) error {
	log.Print("Running database migration Up...")
	session, err := InitializeKeyspaceSession(config)
	if err != nil {
		return fmt.Errorf("could not initialize Cassandra session: %w", err)
	}
	defer session.Close()

	// Check if the schema_migrations table exists, create if not
	err = createSchemaMigrationsTableIfNotExists(session, config.Keyspace)
	if err != nil {
		return fmt.Errorf("could not create schema_migrations table: %w", err)
	}

	//run migrations
	operation := func() error {
		driver, err := cassandra.WithInstance(session, &cassandra.Config{
			KeyspaceName: config.Keyspace,
		})
		if err != nil {
			return fmt.Errorf("could not create Cassandra migration driver: %w", err)
		}

		m, err := migrate.NewWithDatabaseInstance(
			fmt.Sprintf("file://%s", migrationPath),
			config.Keyspace, driver)
		if err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}

		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			return fmt.Errorf("an error occurred while applying migrations: %w", err)
		}
		return nil
	}

	return ExponentialBackoff(operation, maxRetries, initialBackoff)

}

// MigrationDown rolls back all migrations.
func MigrationDown(config *AwsKeyspacesConfig, migrationPath string) error {
	log.Print("Running database migration Down...")
	session, err := InitializeKeyspaceSession(config)
	if err != nil {
		return err
	}
	defer session.Close()

	operation := func() error {
		driver, err := cassandra.WithInstance(session, &cassandra.Config{
			KeyspaceName: config.Keyspace,
		})
		if err != nil {
			return fmt.Errorf("could not create Cassandra migration driver: %w", err)
		}

		m, err := migrate.NewWithDatabaseInstance(
			fmt.Sprintf("file://%s", migrationPath),
			config.Keyspace, driver)
		if err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}

		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			return fmt.Errorf("an error occurred while rolling back migrations: %w", err)
		}

		return nil
	}

	return ExponentialBackoff(operation, maxRetries, initialBackoff)
}
