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
	setAppConfig("aws_keyspaces")
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
}

func TestIntegration_BP_Uptime_Storage(t *testing.T) {

	// 1. Test Local File System Storage
	log.Printf(" >>> 1. Test Local File System Storage")
	setAppConfig("filesystem")
	config := getAppConfig()
	networkName := config.NetworkName
	miniminaNetworkCreate(networkName)
	networkDir := getNetworkDir(networkName)
	uptimeStorageDir := networkDir + "/uptime-storage"
	err := emptyLocalFilesystemStorage(uptimeStorageDir)
	if err != nil {
		t.Fatalf("Failed to empty the %s folder: %v", uptimeStorageDir, err)
	}

	miniminaNetworkStart(networkName)
	defer miniminaNetworkStop(networkName)

	err = waitUntilLocalStorageHasBlocksAndSubmissions(uptimeStorageDir)
	defer emptyLocalFilesystemStorage(uptimeStorageDir)
	if err != nil {
		t.Fatalf("Failed to wait until %s is not empty: %v", uptimeStorageDir, err)
	}

	// 2. Test AWS S3 Storage
	log.Printf(" >>> 2. Test AWS S3 Storage")
	setAppConfig("aws")
	config = getAppConfig()
	awsConfig := config.Aws
	folderPrefix := getAWSIntegrationTestFolder(config)

	miniminaNodeStop(networkName, "uptime-service-backend")
	copyFile(APP_CONFIG_FILE, networkDir+"/uptime_service_config/app_config.json")
	miniminaNodeStart(networkName, "uptime-service-backend")

	err = emptyS3IntegrationTestFolder(*awsConfig, folderPrefix)
	if err != nil {
		t.Fatalf("Failed to empty the integration_test folder: %v", err)
	}

	err = waitUntilS3BucketHasBlocksAndSubmissions(*awsConfig, folderPrefix)
	defer emptyS3IntegrationTestFolder(*awsConfig, folderPrefix)
	if err != nil {
		t.Fatalf("Failed to wait until S3 bucket is not empty: %v", err)
	}

	// 3. Test AWS Keyspaces Storage
	log.Printf(" >>> 3. Test AWS Keyspaces Storage")
	setAppConfig("aws_keyspaces")
	config = getAppConfig()
	config.AwsKeyspaces.SSLCertificatePath = AWS_SSL_CERTIFICATE_PATH

	defer dg.DropAllTables(config.AwsKeyspaces)

	miniminaNodeStop(networkName, "uptime-service-backend")
	copyFile(APP_CONFIG_FILE, networkDir+"/uptime_service_config/app_config.json")
	copyFile(AWS_SSL_CERTIFICATE_PATH, networkDir+"/uptime_service_config/sf-class2-root.crt")
	miniminaNodeStart(networkName, "uptime-service-backend")

	err = waitUntilKeyspacesHasBlocksAndSubmissions(config)
	if err != nil {
		t.Fatalf("Failed to wait until Keyspaces has blocks and submissions: %v", err)
	}

}
