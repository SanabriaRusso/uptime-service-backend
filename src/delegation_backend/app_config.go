package delegation_backend

import (
	"encoding/json"
	"os"

	logging "github.com/ipfs/go-log/v2"
)

func GetAWSBucketName(config AppConfig) string {
	if config.Aws != nil {
		return config.Aws.AccountId + "-" + config.Aws.BucketNameSuffix
	}
	return "" // return empty in case AWSConfig is nil
}

func LoadEnv(log logging.EventLogger) AppConfig {
	var config AppConfig

	configFile := os.Getenv("CONFIG_FILE")
	if configFile != "" {
		file, err := os.Open(configFile)
		if err != nil {
			log.Fatalf("Error loading config file: %s", err)
		}
		defer file.Close()
		decoder := json.NewDecoder(file)
		err = decoder.Decode(&config)
		if err != nil {
			log.Fatalf("Error decoding config file: %s", err)
		}
		// Set AWS credentials from config file in case we are using AWS
		if config.Aws != nil {
			os.Setenv("AWS_ACCESS_KEY_ID", config.Aws.AccessKeyId)
			os.Setenv("AWS_SECRET_ACCESS_KEY", config.Aws.SecretAccessKey)
		}
	} else {
		networkName := getEnvChecked("CONFIG_NETWORK_NAME", log)
		gsheetId := getEnvChecked("CONFIG_GSHEET_ID", log)
		delegationWhitelistList := getEnvChecked("DELEGATION_WHITELIST_LIST", log)
		delegationWhitelistColumn := getEnvChecked("DELEGATION_WHITELIST_COLUMN", log)

		// AWS configurations
		if accessKeyId := os.Getenv("AWS_ACCESS_KEY_ID"); accessKeyId != "" {
			secretAccessKey := getEnvChecked("AWS_SECRET_ACCESS_KEY", log)
			awsRegion := getEnvChecked("CONFIG_AWS_REGION", log)
			awsAccountId := getEnvChecked("CONFIG_AWS_ACCOUNT_ID", log)
			bucketNameSuffix := getEnvChecked("CONFIG_BUCKET_NAME_SUFFIX", log)

			config.Aws = &AwsConfig{
				AccountId:        awsAccountId,
				BucketNameSuffix: bucketNameSuffix,
				Region:           awsRegion,
				AccessKeyId:      accessKeyId,
				SecretAccessKey:  secretAccessKey,
			}
		}

		// Database configurations
		if connectionString := os.Getenv("CONFIG_DATABASE_CONNECTION_STRING"); connectionString != "" {
			databaseType := getEnvChecked("CONFIG_DATABASE_TYPE", log)

			config.Database = &DatabaseConfig{
				ConnectionString: connectionString,
				DatabaseType:     databaseType,
			}
		}

		// LocalFileSystem configurations
		if path := os.Getenv("CONFIG_FILESYSTEM_PATH"); path != "" {
			config.LocalFileSystem = &LocalFileSystemConfig{
				Path: path,
			}
		}

		config.NetworkName = networkName
		config.GsheetId = gsheetId
		config.DelegationWhitelistList = delegationWhitelistList
		config.DelegationWhitelistColumn = delegationWhitelistColumn
	}

	// Check that only one of Aws, Database, or LocalFileSystem is provided
	configCount := 0
	if config.Aws != nil {
		configCount++
	}
	if config.Database != nil {
		configCount++
	}
	if config.LocalFileSystem != nil {
		configCount++
	}

	if configCount != 1 {
		log.Fatalf("Error: You can only provide one of Aws, Database, or LocalFileSystem configurations.")
	}

	return config
}

func getEnvChecked(variable string, log logging.EventLogger) string {
	value := os.Getenv(variable)
	if value == "" {
		log.Fatalf("missing %s environment variable", variable)
	}
	return value
}

type AwsConfig struct {
	AccountId        string `json:"account_id"`
	BucketNameSuffix string `json:"bucket_name_suffix"`
	Region           string `json:"region"`
	AccessKeyId      string `json:"access_key_id"`
	SecretAccessKey  string `json:"secret_access_key"`
}

type DatabaseConfig struct {
	ConnectionString string `json:"connection_string"`
	DatabaseType     string `json:"database_type"`
}

type LocalFileSystemConfig struct {
	Path string `json:"path"`
}

type AppConfig struct {
	NetworkName               string                 `json:"network_name"`
	GsheetId                  string                 `json:"gsheet_id"`
	DelegationWhitelistList   string                 `json:"delegation_whitelist_list"`
	DelegationWhitelistColumn string                 `json:"delegation_whitelist_column"`
	Aws                       *AwsConfig             `json:"aws,omitempty"`
	Database                  *DatabaseConfig        `json:"database,omitempty"`
	LocalFileSystem           *LocalFileSystemConfig `json:"filesystem,omitempty"`
}
