package delegation_backend

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"

	"github.com/aws/aws-sigv4-auth-cassandra-gocql-driver-plugin/sigv4"
	"github.com/gocql/gocql"
	"github.com/golang-migrate/migrate/v4"
	cassandra "github.com/golang-migrate/migrate/v4/database/cassandra"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	logging "github.com/ipfs/go-log/v2"
)

// InitializeKeyspaceSession creates a new gocql session for Amazon Keyspaces using the provided configuration.
func InitializeKeyspaceSession(ctx context.Context, awsKeyspaceConf *AwsKeyspacesConfig) (*gocql.Session, error) {

	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(awsKeyspaceConf.Region))
	if err != nil {
		log.Fatalf("Error loading AWS configuration: %v", err)
	}
	creds, err := awsCfg.Credentials.Retrieve(ctx)
	if err != nil {
		log.Fatalf("Error retrieving AWS credentials: %v", err)
	}

	ksConfig := &AwsKeyspacesConfig{
		AccessKeyId:        creds.AccessKeyID,
		SecretAccessKey:    creds.SecretAccessKey,
		Region:             awsCfg.Region,
		Keyspace:           awsKeyspaceConf.Keyspace,
		SSLCertificatePath: awsKeyspaceConf.SSLCertificatePath,
	}
	auth := sigv4.NewAwsAuthenticator()
	auth.AccessKeyId = ksConfig.AccessKeyId
	auth.SecretAccessKey = ksConfig.SecretAccessKey
	auth.Region = ksConfig.Region

	// Create a SigV4 gocql cluster config
	endpoint := "cassandra." + ksConfig.Region + ".amazonaws.com"
	cluster := gocql.NewCluster(endpoint)
	cluster.Keyspace = ksConfig.Keyspace
	cluster.Port = 9142
	cluster.Authenticator = auth
	cluster.SslOpts = &gocql.SslOptions{
		CaPath:                 ksConfig.SSLCertificatePath,
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

type Submission struct {
	BlockHash          string    `json:"block_hash"`
	SubmittedAtDate    string    // Extracted from filePath
	SubmittedAt        time.Time // Extracted from filePath and parsed
	CreatedAt          time.Time `json:"created_at"`
	RemoteAddr         string    `json:"remote_addr"`
	PeerId             string    `json:"peer_id"`
	Submitter          string    `json:"submitter"` // is base58check-encoded submitter's public key
	SnarkWork          []byte    `json:"snark_work,omitempty"`
	GraphqlControlPort int       `json:"graphql_control_port,omitempty"`
	BuiltWithCommitSha string    `json:"built_with_commit_sha,omitempty"`
}

type Block struct {
	BlockHash string
	RawBlock  []byte
}

func (kc *KeyspaceContext) parseSubmissionBytes(data []byte, filePath string) (*Submission, error) {
	// Extract information from filePath
	// kc.Log.Debugf("filePath: %s\n", filePath)
	filePathParts := strings.Split(filePath, "/")
	if len(filePathParts) < 3 {
		return nil, fmt.Errorf("invalid file path: %s", filePath)
	}
	submittedAtDate := filePathParts[1]
	submittedAtWithSubmitter := strings.TrimSuffix(filePathParts[2], ".json")
	lastHyphenIndex := strings.LastIndex(submittedAtWithSubmitter, "-")
	submittedAtStr := submittedAtWithSubmitter[:lastHyphenIndex]

	// Parse submittedAtStr string into time.Time
	submittedAt, err := time.Parse("2006-01-02T15:04:05Z", submittedAtStr)
	if err != nil {
		return nil, fmt.Errorf("error parsing submitted_at string: %w", err)
	}

	// Parse JSON contents
	var submission Submission
	err = json.Unmarshal(data, &submission)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling submission JSON: %w", err)
	}

	// Populate additional fields from filePath
	submission.SubmittedAtDate = submittedAtDate
	submission.SubmittedAt = submittedAt
	// kc.Log.Debugf("submission: %v\n", submission)

	return &submission, nil
}

func (kc *KeyspaceContext) parseBlockBytes(data []byte, filePath string) (*Block, error) {
	// Extract the filename without the extension to use as the BlockHash
	// kc.Log.Debugf("filePath: %s\n", filePath)
	filename := filepath.Base(filePath)
	blockHash := strings.TrimSuffix(filename, filepath.Ext(filename))

	block := &Block{
		BlockHash: blockHash,
		RawBlock:  data,
	}
	// kc.Log.Debugf("block: %v\n", block)
	return block, nil
}

type KeyspaceContext struct {
	Session  *gocql.Session
	Keyspace string
	Context  context.Context
	Log      *logging.ZapEventLogger
}

// Insert a submission into the Keyspaces database
func (kc *KeyspaceContext) insertSubmission(submission *Submission) error {
	return ExponentialBackoff(func() error {
		return kc.Session.Query(
			"INSERT INTO "+kc.Keyspace+".submissions (submitted_at_date, submitted_at, submitter, remote_addr, peer_id, snark_work, block_hash, created_at, graphql_control_port, built_with_commit_sha) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			submission.SubmittedAtDate,
			submission.SubmittedAt,
			submission.Submitter,
			submission.RemoteAddr,
			submission.PeerId,
			submission.SnarkWork,
			submission.BlockHash,
			submission.CreatedAt,
			submission.GraphqlControlPort,
			submission.BuiltWithCommitSha,
		).Exec()
	}, maxRetries, initialBackoff)
}

// Insert a block into the Keyspaces database
func (kc *KeyspaceContext) insertBlock(block *Block) error {
	return ExponentialBackoff(func() error {
		return kc.Session.Query(
			"INSERT INTO "+kc.Keyspace+".blocks (block_hash, raw_block) VALUES (?, ?)",
			block.BlockHash,
			block.RawBlock,
		).Exec()
	}, maxRetries, initialBackoff)
}

// KeyspaceSave saves the provided objects into Amazon Keyspaces.
func (kc *KeyspaceContext) KeyspaceSave(objs ObjectsToSave) {
	for path, bs := range objs {
		if strings.HasPrefix(path, "submissions/") {
			submission, err := kc.parseSubmissionBytes(bs, path)
			kc.Log.Debugf("Saving submission for block: %v, submitter: %v", submission.BlockHash, submission.Submitter)
			if err != nil {
				kc.Log.Warnf("Error parsing submission JSON: %v", err)
				continue
			}
			if err := kc.insertSubmission(submission); err != nil {
				kc.Log.Warnf("Error saving submission to Keyspaces: %v", err)
			}
		} else if strings.HasPrefix(path, "blocks/") {
			block, err := kc.parseBlockBytes(bs, path)
			kc.Log.Debugf("Saving block: %v", block.BlockHash)
			if err != nil {
				kc.Log.Warnf("Error parsing block file: %v", err)
				continue
			}
			if err := kc.insertBlock(block); err != nil {
				kc.Log.Warnf("Error saving block to Keyspaces: %v", err)
			}
		} else {
			kc.Log.Warnf("Unknown path format: %s", path)
		}
	}
}

func createSchemaMigrationsTableIfNotExists(session *gocql.Session, keyspace string) error {
	query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.schema_migrations (version bigint PRIMARY KEY, dirty boolean);`, keyspace)
	operation := func() error {
		return session.Query(query).Exec()
	}

	return ExponentialBackoff(operation, maxRetries, initialBackoff)
}

func DropAllTables(ctx context.Context, config *AwsKeyspacesConfig) error {
	log.Print("Dropping all tables...")
	operation := func() error {
		session, err := InitializeKeyspaceSession(ctx, config)
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
func MigrationUp(ctx context.Context, config *AwsKeyspacesConfig, migrationPath string) error {
	log.Print("Running database migration Up...")
	session, err := InitializeKeyspaceSession(ctx, config)
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
func MigrationDown(ctx context.Context, config *AwsKeyspacesConfig, migrationPath string) error {
	log.Print("Running database migration Down...")
	session, err := InitializeKeyspaceSession(ctx, config)
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
