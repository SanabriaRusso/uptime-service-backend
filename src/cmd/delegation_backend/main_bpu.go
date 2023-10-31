package main

import (
	. "block_producers_uptime/delegation_backend"
	"context"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	logging "github.com/ipfs/go-log/v2"
	"google.golang.org/api/option"
	sheets "google.golang.org/api/sheets/v4"
)

func main() {
	// Setup logging
	logging.SetupLogging(logging.Config{
		Format: logging.JSONOutput,
		Stderr: true,
		Stdout: false,
		Level:  logging.LevelDebug,
		File:   "",
	})
	log := logging.Logger("delegation backend")
	log.Infof("delegation backend has the following logging subsystems active: %v", logging.GetSubsystems())

	// Context and app initialization
	ctx := context.Background()
	appCfg := LoadEnv(log)
	app := new(App)
	app.Log = log

	// Storage backend setup
	if appCfg.Aws != nil {
		log.Infof("storage backend: AWS S3")
		awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(appCfg.Aws.Region))
		if err != nil {
			log.Fatalf("Error loading AWS configuration: %v", err)
		}
		client := s3.NewFromConfig(awsCfg)
		awsctx := AwsContext{Client: client, BucketName: aws.String(GetAWSBucketName(appCfg)), Prefix: appCfg.NetworkName, Context: ctx, Log: log}
		app.Save = func(objs ObjectsToSave) {
			awsctx.S3Save(objs)
		}
	} else if appCfg.LocalFileSystem != nil {
		log.Infof("storage backend: Local File System")
		app.Save = func(objs ObjectsToSave) {
			LocalFileSystemSave(objs, appCfg.LocalFileSystem.Path, log)
		}
	} else if appCfg.Database != nil {
		log.Infof("storage backend: Database")
		// future implementation of database storage
	} else {
		log.Fatal("No storage backend configured!")
	}

	// App other configurations
	app.Now = func() time.Time { return time.Now() }
	app.SubmitCounter = NewAttemptCounter(REQUESTS_PER_PK_HOURLY)

	// HTTP handlers setup
	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		_, _ = rw.Write([]byte("delegation backend service"))
	})
	http.Handle("/v1/submit", app.NewSubmitH())

	// Sheets service and whitelist loop
	sheetsService, err2 := sheets.NewService(ctx, option.WithScopes(sheets.SpreadsheetsReadonlyScope))
	if err2 != nil {
		log.Fatalf("Error creating Sheets service: %v", err2)
	}
	initWl := RetrieveWhitelist(sheetsService, log, appCfg)
	wlMvar := new(WhitelistMVar)
	wlMvar.Replace(&initWl)
	app.Whitelist = wlMvar
	go func() {
		for {
			wl := RetrieveWhitelist(sheetsService, log, appCfg)
			wlMvar.Replace(&wl)
			time.Sleep(WHITELIST_REFRESH_INTERVAL)
		}
	}()

	// Start server
	log.Fatal(http.ListenAndServe(DELEGATION_BACKEND_LISTEN_TO, nil))
}
