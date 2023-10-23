package integration_tests

import (
	"fmt"
	"os"
	"os/exec"
)

const (
	TEST_DATA_FOLDER = "../../test/data/integration"

	GENESIS_FILE  = TEST_DATA_FOLDER + "/topology/genesis_ledger.json"
	TOPOLOGY_FILE = TEST_DATA_FOLDER + "/topology/topology.json"

	APP_CONFIG_FILE = TEST_DATA_FOLDER + "/topology/uptime_service_config/app_config.json"
	AWS_CREDS_FILE  = TEST_DATA_FOLDER + "/topology/uptime_service_config/aws_creds.json"
	MINASHEETS_FILE = TEST_DATA_FOLDER + "/topology/uptime_service_config/minasheets.json"

	BUCKET_NAME_SUFFIX = "block-producers-uptime"
	TIMEOUT_IN_S       = 600
)

var uptime_service_config_files = []string{
	APP_CONFIG_FILE,
	AWS_CREDS_FILE,
	MINASHEETS_FILE,
}

func encodeSecrets() error {
	fixturesSecret := os.Getenv("UPTIME_SERVICE_SECRET")
	if fixturesSecret == "" {
		return fmt.Errorf("Error: UPTIME_SERVICE_SECRET environment variable not set")
	}

	for _, file := range uptime_service_config_files {
		fmt.Printf(">> Encoding %s...\n", file)

		// Construct the gpg command
		cmd := exec.Command(
			"gpg",
			"--pinentry-mode", "loopback",
			"--passphrase", fixturesSecret,
			"--symmetric",
			"--output", fmt.Sprintf("%s.gpg", file),
			file,
		)

		// Execute and get output
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("Error encoding %s: %s\n", file, err)
		}

		fmt.Println(string(out))
	}

	return nil
}

func decodeSecrets() error {
	fixturesSecret := os.Getenv("UPTIME_SERVICE_SECRET")
	if fixturesSecret == "" {
		return fmt.Errorf("Error: UPTIME_SERVICE_SECRET environment variable not set")
	}

	for _, file := range uptime_service_config_files {
		gpg_file := fmt.Sprintf("%s.gpg", file)

		// skip if file exists
		if _, err := os.Stat(file); err == nil {
			fmt.Printf(">> Skipping decoding %s... JSON file already exists.\n", gpg_file)
			continue
		}

		fmt.Printf(">> Decoding %s...\n", gpg_file)

		// Construct the gpg command
		cmd := exec.Command(
			"gpg",
			"--pinentry-mode", "loopback",
			"--yes",
			"--passphrase", fixturesSecret,
			"--output", file,
			"--decrypt", gpg_file,
		)

		// Execute and get output
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("Error decoding %s: %s\n", gpg_file, err)
		}

		fmt.Println(string(out))
	}

	return nil
}
