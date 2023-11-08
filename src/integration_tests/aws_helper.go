package integration_tests

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"block_producers_uptime/delegation_backend"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func getAppConfig() delegation_backend.AppConfig {
	appConfigBytes, err := os.ReadFile(APP_CONFIG_FILE)
	if err != nil {
		log.Fatalf("Failed to read app_config.json: %v", err)
	}

	var config delegation_backend.AppConfig
	if err := json.Unmarshal(appConfigBytes, &config); err != nil {
		log.Fatalf("Failed to parse app_config.json: %v", err)
	}

	return config
}

func getAWSBucketName(aws delegation_backend.AwsConfig) string {
	return aws.AccountId + "-" + aws.BucketNameSuffix
}

func getAWSIntegrationTestFolder(config delegation_backend.AppConfig) string {
	return strings.Trim(config.NetworkName, "/") + "/"
}

func getS3Service(config delegation_backend.AwsConfig) *s3.S3 {
	accessKeyID := config.AccessKeyId
	secretAccessKey := config.SecretAccessKey
	region := config.Region

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
	})
	if err != nil {
		log.Fatalf("Failed to create AWS session: %v", err)
	}

	return s3.New(sess)
}

func emptyS3IntegrationTestFolder(config delegation_backend.AwsConfig, folderPrefix string) error {
	log.Printf("Emptying AWS S3 integration_test folder")

	bucketName := getAWSBucketName(config)
	svc := getS3Service(config)

	// List objects with the specified prefix
	listObjectsInput := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
		Prefix: aws.String(folderPrefix),
	}
	objects, err := svc.ListObjectsV2(listObjectsInput)
	if err != nil {
		return err
	}

	// Delete each object with the specified prefix
	for _, object := range objects.Contents {
		if *object.Key == folderPrefix {
			// Skip the folder itself
			continue
		}
		deleteObjectInput := &s3.DeleteObjectInput{
			Bucket: aws.String(bucketName),
			Key:    object.Key,
		}
		_, err := svc.DeleteObject(deleteObjectInput)
		if err != nil {
			return err
		}
	}

	return nil
}

func waitUntilS3BucketHasBlocksAndSubmissions(config delegation_backend.AwsConfig, folderPrefix string) error {
	log.Printf("Waiting for blocks and submissions to appear in the S3 bucket")

	bucketName := getAWSBucketName(config)
	svc := getS3Service(config)

	hasBlocks := false
	hasSubmissionsForToday := false
	currentDate := time.Now().Format("2006-01-02") // YYYY-MM-DD format

	timeout := time.After(TIMEOUT_IN_S * time.Second)
	tick := time.Tick(5 * time.Second)

	for {
		select {
		case <-timeout:
			return fmt.Errorf("Timeout reached while waiting for S3 bucket contents")
		case <-tick:
			// List objects with the specified prefix
			listObjectsInput := &s3.ListObjectsV2Input{
				Bucket: aws.String(bucketName),
				Prefix: aws.String(folderPrefix),
			}
			objects, err := svc.ListObjectsV2(listObjectsInput)
			if err != nil {
				return err
			}

			// Reset the checks
			hasBlocks = false
			hasSubmissionsForToday = false

			if len(objects.Contents) > 1 {
				log.Printf("Found objects in the S3 bucket. Checking for blocks and submissions...")
				// Check the objects
				for _, object := range objects.Contents {
					key := *object.Key
					if strings.HasPrefix(key, folderPrefix+"blocks/") && strings.HasSuffix(key, ".dat") {
						hasBlocks = true
					}
					if strings.HasPrefix(key, folderPrefix+"submissions/"+currentDate+"/") && strings.HasSuffix(key, ".json") {
						hasSubmissionsForToday = true
					}
				}
			}

			// If both blocks and submissions for today are found, return
			if hasBlocks && hasSubmissionsForToday {
				log.Printf("Found blocks and submissions for today")
				return nil
			}
		}
	}
}
