package integration_tests

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type AWSCreds struct {
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
}

type AppConfig struct {
	NetworkName string `json:"network_name"`
	AWS         struct {
		AccountID string `json:"account_id"`
		Region    string `json:"region"`
	} `json:"aws"`
}

func getAWSCreds() AWSCreds {
	credsBytes, err := ioutil.ReadFile(AWS_CREDS_FILE)
	if err != nil {
		log.Fatalf("Failed to read aws_creds.json: %v", err)
	}

	var creds AWSCreds
	if err := json.Unmarshal(credsBytes, &creds); err != nil {
		log.Fatalf("Failed to parse aws_creds.json: %v", err)
	}

	return creds
}

func getAppConfig() AppConfig {
	appConfigBytes, err := ioutil.ReadFile(APP_CONFIG_FILE)
	if err != nil {
		log.Fatalf("Failed to read app_config.json: %v", err)
	}

	var config AppConfig
	if err := json.Unmarshal(appConfigBytes, &config); err != nil {
		log.Fatalf("Failed to parse app_config.json: %v", err)
	}

	return config
}

func getAWSBucketName(config AppConfig) string {
	return config.AWS.AccountID + "-" + BUCKET_NAME_SUFFIX
}

func getAWSIntegrationTestFolder(config AppConfig) string {
	return strings.Trim(config.NetworkName, "/") + "/"
}

func getS3Service(creds AWSCreds, config AppConfig) *s3.S3 {
	accessKeyID := creds.AccessKeyID
	secretAccessKey := creds.SecretAccessKey
	region := config.AWS.Region

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
	})
	if err != nil {
		log.Fatalf("Failed to create AWS session: %v", err)
	}

	return s3.New(sess)
}

func emptyIntegrationTestFolder(creds AWSCreds, config AppConfig) error {
	log.Printf("Emptying AWS S3 integration_test folder")

	bucketName := getAWSBucketName(config)
	folderPrefix := getAWSIntegrationTestFolder(config)
	svc := getS3Service(creds, config)

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

func waitUntilS3BucketHasBlocksAndSubmissions(creds AWSCreds, config AppConfig) error {
	log.Printf("Waiting for blocks and submissions to appear in the S3 bucket")

	bucketName := getAWSBucketName(config)
	folderPrefix := getAWSIntegrationTestFolder(config)
	svc := getS3Service(creds, config)

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

			// If both blocks and submissions for today are found, return
			if hasBlocks && hasSubmissionsForToday {
				return nil
			}
		}
	}
}
