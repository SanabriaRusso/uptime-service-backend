package delegation_backend

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sigv4-auth-cassandra-gocql-driver-plugin/sigv4"
	"github.com/gocql/gocql"
	"github.com/golang-migrate/migrate/v4"
	cassandra "github.com/golang-migrate/migrate/v4/database/cassandra"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	logging "github.com/ipfs/go-log/v2"
)

// InitializeKeyspaceSession creates a new gocql session for Amazon Keyspaces using the provided configuration.
func InitializeKeyspaceSession(config *AwsKeyspacesConfig) (*gocql.Session, error) {
	auth := sigv4.NewAwsAuthenticator()
	roleSessionName := os.Getenv("UPTIME_SERVICE_AWS_ROLE_SESSION_NAME")
	roleArn := os.Getenv("UPTIME_SERVICE_AWS_ROLE_ARN")

	if roleSessionName != "" && roleArn != "" {
		// If role-related env variables are set, use temporary credentials
		awsSession, err := session.NewSession(&aws.Config{Region: aws.String(config.Region)})
		if err != nil {
			return nil, fmt.Errorf("error creating AWS session: %w", err)
		}

		stsSvc := sts.New(awsSession)
		creds, err := stsSvc.AssumeRole(&sts.AssumeRoleInput{
			RoleArn:         &roleArn,
			RoleSessionName: &roleSessionName,
		})
		if err != nil {
			return nil, fmt.Errorf("unable to assume role: %w", err)
		}

		auth.AccessKeyId = *creds.Credentials.AccessKeyId
		auth.SecretAccessKey = *creds.Credentials.SecretAccessKey
		auth.SessionToken = *creds.Credentials.SessionToken
		auth.Region = config.Region
	} else {
		// Otherwise, use credentials from the config
		auth.AccessKeyId = config.AccessKeyId
		auth.SecretAccessKey = config.SecretAccessKey
		auth.Region = config.Region
	}

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

type Submission struct {
	BlockHash          string    `json:"block_hash"`
	SubmittedAtDate    string    // Extracted from filePath
	SubmittedAt        time.Time // Extracted from filePath and parsed
	CreatedAt          time.Time `json:"created_at"`
	RemoteAddr         string    `json:"remote_addr"`
	PeerId             string    `json:"peer_id"`
	Submitter          string    `json:"submitter"` // is base58check-encoded submitter's public key
	RawBlock           []byte    `json:"raw_block,omitempty"`
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
		if err := kc.tryInsertSubmission(submission, true); err != nil {
			if isRowSizeError(err) {
				kc.Log.Warnf("KeyspaceSave: Block too large, inserting without raw_block")
				return kc.tryInsertSubmission(submission, false)
			}
			return err
		}
		return nil
	}, maxRetries, initialBackoff)
}

func (kc *KeyspaceContext) tryInsertSubmission(submission *Submission, includeRawBlock bool) error {
	query := "INSERT INTO " + kc.Keyspace + ".submissions (submitted_at_date, submitted_at, submitter, remote_addr, peer_id, snark_work, block_hash, created_at, graphql_control_port, built_with_commit_sha"
	values := []interface{}{submission.SubmittedAtDate, submission.SubmittedAt, submission.Submitter, submission.RemoteAddr, submission.PeerId, submission.SnarkWork, submission.BlockHash, submission.CreatedAt, submission.GraphqlControlPort, submission.BuiltWithCommitSha}
	if includeRawBlock {
		query += ", raw_block) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
		values = append(values, submission.RawBlock)
	} else {
		query += ") VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	}
	return kc.Session.Query(query, values...).Exec()
}

func isRowSizeError(err error) bool {
	// Replace with more robust error checking if possible
	return strings.Contains(err.Error(), "The update would cause the row to exceed the maximum allowed size")
}

// KeyspaceSave saves the provided objects into Amazon Keyspaces.
func (kc *KeyspaceContext) KeyspaceSave(objs ObjectsToSave) {
	var submissionToSave *Submission = &Submission{}
	for path, bs := range objs {
		if strings.HasPrefix(path, "submissions/") {
			submission, err := kc.parseSubmissionBytes(bs, path)
			if err != nil {
				kc.Log.Warnf("KeyspaceSave: Error parsing submission JSON: %v", err)
				continue
			}
			submissionToSave.BlockHash = submission.BlockHash
			submissionToSave.CreatedAt = submission.CreatedAt
			submissionToSave.GraphqlControlPort = submission.GraphqlControlPort
			submissionToSave.PeerId = submission.PeerId
			submissionToSave.RemoteAddr = submission.RemoteAddr
			submissionToSave.SnarkWork = submission.SnarkWork
			submissionToSave.SubmittedAt = submission.SubmittedAt
			submissionToSave.SubmittedAtDate = submission.SubmittedAtDate
			submissionToSave.Submitter = submission.Submitter
			submissionToSave.BuiltWithCommitSha = submission.BuiltWithCommitSha

		} else if strings.HasPrefix(path, "blocks/") {
			block, err := kc.parseBlockBytes(bs, path)
			if err != nil {
				kc.Log.Warnf("KeyspaceSave: Error parsing block file: %v", err)
				continue
			}
			submissionToSave.RawBlock = block.RawBlock
			submissionToSave.BlockHash = block.BlockHash
		} else {
			kc.Log.Warnf("KeyspaceSave: Unknown path format: %s", path)
		}

	}
	kc.Log.Debugf("KeyspaceSave: Saving submission for block: %v, submitter: %v, submitted_at: %v", submissionToSave.BlockHash, submissionToSave.Submitter, submissionToSave.SubmittedAt)
	if err := kc.insertSubmission(submissionToSave); err != nil {
		kc.Log.Warnf("KeyspaceSave: Error saving submission to Keyspaces: %v", err)
	}
}

func createSchemaMigrationsTableIfNotExists(session *gocql.Session, keyspace string) error {
	query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.schema_migrations (version bigint PRIMARY KEY, dirty boolean);`, keyspace)
	operation := func() error {
		return session.Query(query).Exec()
	}

	return ExponentialBackoff(operation, 5, 1*time.Second)
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

	return ExponentialBackoff(operation, 10, 1*time.Second)

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
