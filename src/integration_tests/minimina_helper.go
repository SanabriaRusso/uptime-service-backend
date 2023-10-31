package integration_tests

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"os/exec"
)

func miniminaNetworkCreate(network string) {
	log.Printf("Creating network %s", network)
	cmd := exec.Command("minimina", "network", "create", "-n", network, "-g", GENESIS_FILE, "-t", TOPOLOGY_FILE)

	// cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		log.Fatalf("failed to execute command: %v", err)
	}
}

func miniminaNetworkStart(network string) {
	log.Printf("Starting network %s", network)
	cmd := exec.Command("minimina", "network", "start", "-n", network)

	// cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		log.Fatalf("failed to execute command: %v", err)
	}
}

func miniminaNetworkStop(network string) {
	log.Printf("Stopping network %s", network)
	cmd := exec.Command("minimina", "network", "stop", "-n", network)

	// cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		log.Fatalf("failed to execute command: %v", err)
	}
}

func miniminaNetworkDelete(network string) {
	log.Printf("Deleting network %s", network)
	cmd := exec.Command("minimina", "network", "delete", "-n", network)

	// cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		log.Fatalf("failed to execute command: %v", err)
	}
}

func miniminaNodeStop(network string, node string) {
	log.Printf("Stopping node %s of network %s", node, network)
	cmd := exec.Command("minimina", "node", "stop", "-n", network, "-i", node)

	// cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		log.Fatalf("failed to execute command: %v", err)
	}
}

func miniminaNodeStart(network string, node string) {
	log.Printf("Starting node %s of network %s", node, network)
	cmd := exec.Command("minimina", "node", "start", "-n", network, "-i", node)

	// cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		log.Fatalf("failed to execute command: %v", err)
	}
}

type NetworkStatus struct {
	NetworkDir string `json:"network_dir"`
}

func getNetworkDir(network string) string {
	var out bytes.Buffer
	cmd := exec.Command("minimina", "network", "status", "-n", network)
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		log.Fatalf("failed to execute command: %v", err)
	}

	var netStatus NetworkStatus
	err = json.Unmarshal(out.Bytes(), &netStatus)
	if err != nil {
		log.Fatalf("failed to unmarshal JSON: %v", err)
	}

	return netStatus.NetworkDir
}
