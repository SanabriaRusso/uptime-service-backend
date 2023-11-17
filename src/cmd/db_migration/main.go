package main

import (
	dg "block_producers_uptime/delegation_backend"
	"context"
	"os"

	logging "github.com/ipfs/go-log/v2"
)

const DATABASE_MIGRATION_DIR = "../../../database/migrations"

func main() {
	// Setup logging
	logging.SetupLogging(logging.Config{
		Format: logging.JSONOutput,
		Stderr: true,
		Stdout: false,
		Level:  logging.LevelDebug,
		File:   "",
	})
	log := logging.Logger("delegation backend db migration")

	config := dg.LoadEnv(log)
	ctx := context.Background()

	if len(os.Args) < 2 {
		log.Fatal("Missing required command: 'up' or 'down'")
	}

	if config.AwsKeyspaces != nil {
		log.Infof("storage backend: Aws Keyspaces")
		switch os.Args[1] {
		case "up":
			err := dg.MigrationUp(ctx, config.AwsKeyspaces, DATABASE_MIGRATION_DIR)
			if err != nil {
				log.Fatalf("Migration up failed: %v", err)
			}
		case "down":
			err := dg.MigrationDown(ctx, config.AwsKeyspaces, DATABASE_MIGRATION_DIR)
			if err != nil {
				log.Fatalf("Migration down failed: %v", err)
			}
		default:
			log.Fatal("Invalid command. Use 'up' or 'down'")
		}
	} else {
		log.Fatalf("No Aws Keyspaces backend configured! Make sure you have loaded CONFIG_FILE environment variable with the path to the config file including aws_keyspaces configuration!")
	}
}
