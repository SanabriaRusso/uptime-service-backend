package integration_tests

import (
	"log"
	"os"
	"os/exec"
)

func miniminaNetworkCreate(network string) {
	cmd := exec.Command("minimina", "network", "create", "-n", network, "-g", GENESIS_FILE, "-t", TOPOLOGY_FILE)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("Creating network %s", network)
	err := cmd.Run()
	if err != nil {
		log.Fatalf("failed to execute command: %v", err)
	}
}

func miniminaNetworkStart(network string) {
	cmd := exec.Command("minimina", "network", "start", "-n", network)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("Starting network %s", network)
	err := cmd.Run()
	if err != nil {
		log.Fatalf("failed to execute command: %v", err)
	}
}

func miniminaNetworkStop(network string) {
	cmd := exec.Command("minimina", "network", "stop", "-n", network)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("Stopping network %s", network)
	err := cmd.Run()
	if err != nil {
		log.Fatalf("failed to execute command: %v", err)
	}
}

func miniminaNetworkDelete(network string) {
	cmd := exec.Command("minimina", "network", "delete", "-n", network)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("Deleting network %s", network)
	err := cmd.Run()
	if err != nil {
		log.Fatalf("failed to execute command: %v", err)
	}
}
