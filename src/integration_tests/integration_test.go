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

func TestIntegrationBP_Uptime_S3(t *testing.T) {
	config := getAppConfig()

	defer miniminaNetworkDelete(config.NetworkName)
	defer emptyIntegrationTestFolder(config)

	err := emptyIntegrationTestFolder(config)
	if err != nil {
		t.Fatalf("Failed to empty the integration_test folder: %v", err)
	}

	miniminaNetworkCreate(config.NetworkName)
	miniminaNetworkStart(config.NetworkName)

	err = waitUntilS3BucketHasBlocksAndSubmissions(config)
	if err != nil {
		t.Fatalf("Failed to wait until S3 bucket is not empty: %v", err)
	}

}
