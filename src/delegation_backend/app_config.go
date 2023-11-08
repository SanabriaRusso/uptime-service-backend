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
		// Set AWS credentials from config file in case we are using AWS S3 or AWS Keyspaces
		if config.Aws != nil {
			os.Setenv("AWS_ACCESS_KEY_ID", config.Aws.AccessKeyId)
			os.Setenv("AWS_SECRET_ACCESS_KEY", config.Aws.SecretAccessKey)
		}
	} else {
		networkName := getEnvChecked("CONFIG_NETWORK_NAME", log)

		delegationWhitelistDisabled := boolEnvChecked("DELEGATION_WHITELIST_DISABLED", log)
		var gsheetId, delegationWhitelistList, delegationWhitelistColumn string
		if delegationWhitelistDisabled {
			// If delegation whitelist is disabled, we don't need to load related environment variables
			// just loading them from env in case they are set, but they won't be used
			gsheetId = os.Getenv("CONFIG_GSHEET_ID")
			delegationWhitelistList = os.Getenv("DELEGATION_WHITELIST_LIST")
			delegationWhitelistColumn = os.Getenv("DELEGATION_WHITELIST_COLUMN")
		} else {
			// If delegation whitelist is enabled, we need to load related environment variables
			// program will terminate if any of them is missing
			gsheetId = getEnvChecked("CONFIG_GSHEET_ID", log)
			delegationWhitelistList = getEnvChecked("DELEGATION_WHITELIST_LIST", log)
			delegationWhitelistColumn = getEnvChecked("DELEGATION_WHITELIST_COLUMN", log)
		}

		// AWS configurations
		if bucketNameSuffix := os.Getenv("AWS_BUCKET_NAME_SUFFIX"); bucketNameSuffix != "" {
			accessKeyId := getEnvChecked("AWS_ACCESS_KEY_ID", log)
			secretAccessKey := getEnvChecked("AWS_SECRET_ACCESS_KEY", log)
			awsRegion := getEnvChecked("AWS_REGION", log)
			awsAccountId := getEnvChecked("AWS_ACCOUNT_ID", log)
			bucketNameSuffix := getEnvChecked("AWS_BUCKET_NAME_SUFFIX", log)

			config.Aws = &AwsConfig{
				AccountId:        awsAccountId,
				BucketNameSuffix: bucketNameSuffix,
				Region:           awsRegion,
				AccessKeyId:      accessKeyId,
				SecretAccessKey:  secretAccessKey,
			}
		}

		// AWSKeyspace configurations
		if awsKeyspace := os.Getenv("AWS_KEYSPACE"); awsKeyspace != "" {
			accessKeyId := getEnvChecked("AWS_ACCESS_KEY_ID", log)
			secretAccessKey := getEnvChecked("AWS_SECRET_ACCESS_KEY", log)
			awsRegion := getEnvChecked("AWS_REGION", log)
			awsKeyspace := getEnvChecked("AWS_KEYSPACE", log)
			sslCertificatePath := getEnvChecked("AWS_SSL_CERTIFICATE_PATH", log)

			config.AwsKeyspaces = &AwsKeyspacesConfig{
				Keyspace:           awsKeyspace,
				Region:             awsRegion,
				AccessKeyId:        accessKeyId,
				SecretAccessKey:    secretAccessKey,
				SSLCertificatePath: sslCertificatePath,
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
		config.DelegationWhitelistDisabled = delegationWhitelistDisabled
	}

	// Check that only one of Aws, Database, or LocalFileSystem is provided
	configCount := 0
	if config.Aws != nil {
		configCount++
	}
	if config.AwsKeyspaces != nil {
		configCount++
	}
	if config.LocalFileSystem != nil {
		configCount++
	}

	if configCount != 1 {
		log.Fatalf("Error: You can only provide one of AwsS3, AwsKeyspaces, or LocalFileSystem configurations.")
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

func boolEnvChecked(variable string, log logging.EventLogger) bool {
	value := os.Getenv(variable)
	switch value {
	case "1":
		return true
	case "0":
		return false
	case "":
		return false
	default:
		log.Fatalf("%s, if set, should be either 0 or 1!", variable)
		return false
	}
}

type AwsConfig struct {
	AccountId        string `json:"account_id"`
	BucketNameSuffix string `json:"bucket_name_suffix"`
	Region           string `json:"region"`
	AccessKeyId      string `json:"access_key_id"`
	SecretAccessKey  string `json:"secret_access_key"`
}

type AwsKeyspacesConfig struct {
	Keyspace           string `json:"keyspace"`
	Region             string `json:"region"`
	AccessKeyId        string `json:"access_key_id"`
	SecretAccessKey    string `json:"secret_access_key"`
	SSLCertificatePath string `json:"ssl_certificate_path"`
}

type LocalFileSystemConfig struct {
	Path string `json:"path"`
}

type AppConfig struct {
	NetworkName                 string                 `json:"network_name"`
	GsheetId                    string                 `json:"gsheet_id"`
	DelegationWhitelistList     string                 `json:"delegation_whitelist_list"`
	DelegationWhitelistColumn   string                 `json:"delegation_whitelist_column"`
	DelegationWhitelistDisabled bool                   `json:"delegation_whitelist_disabled,omitempty"`
	Aws                         *AwsConfig             `json:"aws,omitempty"`
	AwsKeyspaces                *AwsKeyspacesConfig    `json:"aws_keyspaces,omitempty"`
	LocalFileSystem             *LocalFileSystemConfig `json:"filesystem,omitempty"`
}
