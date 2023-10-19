package main

import (
	dg "block_producers_uptime/delegation_backend"
	itn "block_producers_uptime/itn_uptime_analyzer"
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/aws"
	logging "github.com/ipfs/go-log/v2"
)

func main() {

	// Get the time of execution
	currentTime := itn.GetCurrentTime()

	// Set up sync period of type int representing minutes
	syncPeriod := 15

	// Set up execution interval type int representing hours
	executionInterval := 12

	// Setting up logging for application
	logging.SetupLogging(logging.Config{
		Format: logging.JSONOutput,
		Stderr: true,
		Stdout: false,
		Level:  logging.LevelDebug,
		File:   "",
	})
	log := logging.Logger("itn availability script")
	log.Infof("itn availability script has the following logging subsystems active: %v\n", logging.GetSubsystems())

	// Empty context object and initializing memory for application
	ctx := context.Background()

	// Load environment variables
	appCfg := itn.LoadEnv(log)

	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(appCfg.Aws.Region))
	if err != nil {
		log.Fatalf("Error loading AWS configuration: %v\n", err)
	}

	app := new(dg.App)
	app.Log = log
	client := s3.NewFromConfig(awsCfg)

	awsctx := dg.AwsContext{Client: client, BucketName: aws.String(itn.GetBucketName(appCfg)), Prefix: appCfg.NetworkName, Context: ctx, Log: log}

	// Create Google Cloud client

	identities := itn.CreateIdentities(appCfg, awsctx, log, currentTime, executionInterval)

	// Go over identities and calculate uptime
	for _, identity := range identities {
		identity.GetUptime(appCfg, awsctx, log, currentTime, syncPeriod, executionInterval)
		fmt.Print(identity)
	}
}
