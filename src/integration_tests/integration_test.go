package integration_tests

import (
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
}

func TestIntegration_BP_Uptime_Storage(t *testing.T) {
	setAppConfig("filesystem")
	config := getAppConfig()
	networkName := config.NetworkName
	miniminaNetworkCreate(networkName)
	networkDir := getNetworkDir(networkName)
	uptimeStorageDir := networkDir + "/uptime-storage"

	// 1. Test Local File System Storage
	log.Printf(" >>> 1. Test Local File System Storage")
	err := emptyLocalFilesystemStorage(uptimeStorageDir)
	if err != nil {
		t.Fatalf("Failed to empty the %s folder: %v", uptimeStorageDir, err)
	}

	miniminaNetworkStart(networkName)

	err = waitUntilLocalStorageHasBlocksAndSubmissions(uptimeStorageDir)
	if err != nil {
		t.Fatalf("Failed to wait until %s is not empty: %v", uptimeStorageDir, err)
	}

	// 2. Test AWS S3 Storage
	log.Printf(" >>> 2. Test AWS S3 Storage")
	setAppConfig("aws")
	config = getAppConfig()
	miniminaNodeStop(networkName, "uptime-service-backend")
	copyFile(APP_CONFIG_FILE, networkDir+"/uptime_service_config/app_config.json")
	miniminaNodeStart(networkName, "uptime-service-backend")

	err = emptyS3IntegrationTestFolder(config)
	if err != nil {
		t.Fatalf("Failed to empty the integration_test folder: %v", err)
	}

	err = waitUntilS3BucketHasBlocksAndSubmissions(config)
	if err != nil {
		t.Fatalf("Failed to wait until S3 bucket is not empty: %v", err)
	}

	defer miniminaNetworkStop(networkName)
	defer emptyLocalFilesystemStorage(uptimeStorageDir)
	defer emptyS3IntegrationTestFolder(config)

}
