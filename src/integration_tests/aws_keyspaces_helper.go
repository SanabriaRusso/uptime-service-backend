package integration_tests

import (
	dg "block_producers_uptime/delegation_backend"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gocql/gocql"
)

func checkForBlocks(session *gocql.Session, keyspace string) (bool, error) {
	var blockHash string
	query := fmt.Sprintf("SELECT block_hash FROM %s.blocks LIMIT 1", keyspace)
	if err := session.Query(query).Scan(&blockHash); err != nil {
		if err == gocql.ErrNotFound {
			return false, nil // No blocks found
		}
		return false, err // An error occurred
	}
	log.Printf("Found block: %s\n", blockHash)
	return true, nil // At least one block found
}

func checkForSubmissions(session *gocql.Session, keyspace, date string) (bool, error) {
	var submitter string
	query := fmt.Sprintf("SELECT submitter FROM %s.submissions WHERE submitted_at_date='%s' LIMIT 1", keyspace, date)
	if err := session.Query(query).Scan(&submitter); err != nil {
		if err == gocql.ErrNotFound {
			return false, nil // No submissions found for today
		}
		return false, err // An error occurred
	}
	log.Printf("Found submission for today: %s\n", submitter)
	return true, nil // At least one submission found for today
}

func waitUntilKeyspacesHasBlocksAndSubmissions(ctx context.Context, config dg.AppConfig) error {
	log.Printf("Waiting for blocks and submissions to appear in Keyspaces")

	sess, err := dg.InitializeKeyspaceSession(ctx, config.AwsKeyspaces)
	if err != nil {
		return fmt.Errorf("error initializing Keyspace session: %w", err)
	}
	defer sess.Close()

	currentDate := time.Now().Format("2006-01-02") // YYYY-MM-DD format
	timeout := time.After(TIMEOUT_IN_S * time.Second)
	tick := time.Tick(5 * time.Second)

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout reached while waiting for Keyspaces contents")
		case <-tick:
			hasBlocks, err := checkForBlocks(sess, config.AwsKeyspaces.Keyspace)
			if err != nil {
				return err
			}

			hasSubmissionsForToday, err := checkForSubmissions(sess, config.AwsKeyspaces.Keyspace, currentDate)
			if err != nil {
				return err
			}

			// If both blocks and submissions for today are found, return
			if hasBlocks && hasSubmissionsForToday {
				log.Printf("Found blocks and submissions for today in Keyspaces")
				return nil
			}
		}
	}
}

func WaitForTablesActive(ctx context.Context, config *dg.AwsKeyspacesConfig, tables []string) error {
	log.Printf("Waiting for tables %v to be active...", tables)
	session, err := dg.InitializeKeyspaceSession(ctx, config)
	if err != nil {
		return err
	}
	defer session.Close()
	operation := func() error {
		for _, tableName := range tables {

			// Check if the table exists
			query := fmt.Sprintf("SELECT * FROM %s.%s LIMIT 1", config.Keyspace, tableName)
			if err := session.Query(query).Consistency(gocql.One).Exec(); err != nil {
				return err
			}

		}
		return nil
	}

	return dg.ExponentialBackoff(operation, 15, 1*time.Second)
}
