package integration_tests

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	TEST_DATA_FOLDER = "../../test/integration"

	GENESIS_FILE  = TEST_DATA_FOLDER + "/topology/genesis_ledger.json"
	TOPOLOGY_FILE = TEST_DATA_FOLDER + "/topology/topology.json"

	UPTIME_SERVICE_CONFIG_DIR = TEST_DATA_FOLDER + "/topology/uptime_service_config"
	APP_CONFIG_FILE           = UPTIME_SERVICE_CONFIG_DIR + "/app_config.json"

	TIMEOUT_IN_S = 600
)

func getDirFiles(dir string, suffix string) ([]string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var gpgFiles []string
	for _, f := range files {
		if strings.HasSuffix(f.Name(), suffix) {
			absolutePath := dir + "/" + f.Name()
			gpgFiles = append(gpgFiles, absolutePath)
		}
	}

	return gpgFiles, nil
}

func getGpgFiles(dir string) ([]string, error) {
	return getDirFiles(dir, ".gpg")
}

func getJsonFiles(dir string) ([]string, error) {
	return getDirFiles(dir, ".json")
}

func encodeUptimeServiceConf() error {
	fixturesSecret := os.Getenv("UPTIME_SERVICE_SECRET")
	if fixturesSecret == "" {
		return fmt.Errorf("Error: UPTIME_SERVICE_SECRET environment variable not set")
	}

	jsonFiles, err := getJsonFiles(UPTIME_SERVICE_CONFIG_DIR)
	if err != nil {
		return err
	}
	fmt.Printf("jsonFiles: %v\n", jsonFiles)
	for _, json_file := range jsonFiles {
		fmt.Printf(">> Encoding %s...\n", json_file)

		// Construct the gpg command
		cmd := exec.Command(
			"gpg",
			"--pinentry-mode", "loopback",
			"--passphrase", fixturesSecret,
			"--symmetric",
			"--output", fmt.Sprintf("%s.gpg", json_file),
			json_file,
		)

		// Execute and get output
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("Error encoding %s: %s\n", json_file, err)
		}

		fmt.Println(string(out))
	}

	return nil
}

func decodeUptimeServiceConf() error {
	fixturesSecret := os.Getenv("UPTIME_SERVICE_SECRET")
	if fixturesSecret == "" {
		return fmt.Errorf("Error: UPTIME_SERVICE_SECRET environment variable not set")
	}

	gpgFiles, err := getGpgFiles(UPTIME_SERVICE_CONFIG_DIR)
	if err != nil {
		return err
	}

	for _, gpg_file := range gpgFiles {
		json_file := strings.TrimSuffix(gpg_file, ".gpg")
		// skip if file exists
		if _, err := os.Stat(json_file); err == nil {
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
			"--output", json_file,
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
