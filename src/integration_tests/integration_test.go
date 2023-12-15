package integration_tests

import (
	dg "block_producers_uptime/delegation_backend"
	"log"
	"testing"
)

func init() {
	log.Println("Integration tests setup")
	// err := encodeUptimeServiceConf()
	err := decodeUptimeServiceConf()
	if err != nil {
		log.Fatalf("Failed to decode uptime service configuration: %v", err)
	}

	// Setup AWS Keyspaces database
	setAppConfig("all")
	config := getAppConfig()
	config.AwsKeyspaces.SSLCertificatePath = AWS_SSL_CERTIFICATE_PATH
	err = dg.MigrationUp(config.AwsKeyspaces, DATABASE_MIGRATION_DIR)
	if err != nil {
		log.Fatalf("Failed to migrate up: %v", err)
	}
	tables := []string{"schema_migrations", "submissions", "blocks"}
	err = WaitForTablesActive(config.AwsKeyspaces, tables)
	if err != nil {
		log.Fatalf("Failed to wait for tables to be active: %v", err)
	}

	// Setup AWS S3
	awsConfig := config.Aws
	folderPrefix := getAWSIntegrationTestFolder(config)
	err = emptyS3IntegrationTestFolder(*awsConfig, folderPrefix)
	if err != nil {
		log.Fatalf("Failed to empty the integration_test folder: %v", err)
	}
}

func TestIntegration_BP_Uptime_Storage(t *testing.T) {

	log.Printf(" >>> Test Local File System, AWS S3 and AWS Keyspace Storage <<<")
	setAppConfig("all")
	config := getAppConfig()
	networkName := config.NetworkName

	// create network
	miniminaNetworkCreate(networkName)

	// local file system
	localNetworkDir := miniminaGetNetworkDir(networkName)
	uptimeStorageDir := localNetworkDir + "/uptime-storage"

	// AWS S3
	awsConfig := config.Aws
	folderPrefix := getAWSIntegrationTestFolder(config)

	// AWS Keyspaces
	awsKeyspacesConfig := config.AwsKeyspaces
	awsKeyspacesConfig.SSLCertificatePath = AWS_SSL_CERTIFICATE_PATH
	defer dg.DropAllTables(awsKeyspacesConfig)

	// start network
	miniminaNetworkStart(networkName)
	defer miniminaNetworkStop(networkName)

	err := waitUntilLocalStorageHasBlocksAndSubmissions(uptimeStorageDir)
	defer emptyLocalFilesystemStorage(uptimeStorageDir)
	if err != nil {
		t.Fatalf("Failed to wait until %s is not empty: %v", uptimeStorageDir, err)
	}

	err = waitUntilS3BucketHasBlocksAndSubmissions(*awsConfig, folderPrefix)
	defer emptyS3IntegrationTestFolder(*awsConfig, folderPrefix)
	if err != nil {
		t.Fatalf("Failed to wait until S3 bucket is not empty: %v", err)
	}

	err = waitUntilKeyspacesHasBlocksAndSubmissions(config)
	if err != nil {
		t.Fatalf("Failed to wait until Keyspaces has blocks and submissions: %v", err)
	}

}
