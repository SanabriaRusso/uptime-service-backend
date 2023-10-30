package integration_tests

import (
	"log"
	"testing"
)

func init() {
	log.Println("Integration tests setup")
	// err := encodeUptimeServiceConf()
	// if err != nil {
	// 	log.Fatalf("Failed to encode uptime service configuration: %v", err)
	// }
	err := decodeUptimeServiceConf()
	if err != nil {
		log.Fatalf("Failed to decode uptime service configuration: %v", err)
	}
}

func TestIntegration_BP_Uptime_S3(t *testing.T) {
	setAppConfig("aws")
	config := getAppConfig()
	network_name := config.NetworkName + "-aws"

	err := emptyS3IntegrationTestFolder(config)
	if err != nil {
		t.Fatalf("Failed to empty the integration_test folder: %v", err)
	}

	miniminaNetworkCreate(network_name)
	miniminaNetworkStart(network_name)
	defer miniminaNetworkStop(network_name)
	defer emptyS3IntegrationTestFolder(config)

	err = waitUntilS3BucketHasBlocksAndSubmissions(config)
	if err != nil {
		t.Fatalf("Failed to wait until S3 bucket is not empty: %v", err)
	}

}

func TestIntegration_BP_Uptime_Filesystem(t *testing.T) {
	setAppConfig("filesystem")
	config := getAppConfig()
	network_name := config.NetworkName + "-filesystem"
	miniminaNetworkCreate(network_name)
	uptime_storage_dir := getNetworkDir(network_name) + "/uptime-storage"

	err := emptyLocalFilesystemStorage(uptime_storage_dir)
	if err != nil {
		t.Fatalf("Failed to empty the %s folder: %v", uptime_storage_dir, err)
	}

	miniminaNetworkStart(network_name)
	defer miniminaNetworkStop(network_name)
	defer emptyLocalFilesystemStorage(uptime_storage_dir)

	err = waitUntilLocalStorageHasBlocksAndSubmissions(uptime_storage_dir)
	if err != nil {
		t.Fatalf("Failed to wait until %s is not empty: %v", uptime_storage_dir, err)
	}

}
