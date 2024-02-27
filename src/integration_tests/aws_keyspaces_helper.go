package integration_tests

import (
	dg "block_producers_uptime/delegation_backend"
	"fmt"
	"log"
	"time"

	"github.com/gocql/gocql"
)

func checkForSubmissions(session *gocql.Session, keyspace, date string) (bool, error) {
	var submitter, blockHash, rawBlock string

	query := fmt.Sprintf("SELECT submitter, block_hash, raw_block FROM %s.submissions WHERE submitted_at_date='%s' LIMIT 1 ALLOW FILTERING", keyspace, date)

	if err := session.Query(query).Scan(&submitter, &blockHash, &rawBlock); err != nil {
		if err == gocql.ErrNotFound {
			return false, nil
		}
		return false, err
	}

	if submitter == "" || blockHash == "" || rawBlock == "" {
		log.Printf("Found submission for today with empty required fields\n")
		return false, nil // Found a row but required fields are empty
	}

	log.Printf("Found valid submission for today: submitter=%s, block_hash=%s\n", submitter, blockHash)
	return true, nil // Valid submission found with all required fields
}

func waitUntilKeyspacesHasBlocksAndSubmissions(config dg.AppConfig) error {
	log.Printf("Waiting for submissions to appear in Keyspaces")

	sess, err := dg.InitializeKeyspaceSession(config.AwsKeyspaces)
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

			hasSubmissionsForToday, err := checkForSubmissions(sess, config.AwsKeyspaces.Keyspace, currentDate)
			if err != nil {
				return err
			}

			if hasSubmissionsForToday {
				log.Printf("Found submissions for today in Keyspaces")
				return nil
			}
		}
	}
}

func WaitForTablesActive(config *dg.AwsKeyspacesConfig, tables []string) error {
	log.Printf("Waiting for tables %v to be active...", tables)
	session, err := dg.InitializeKeyspaceSession(config)
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
