package delegation_backend

import (
	"encoding/json"
	"os"

	logging "github.com/ipfs/go-log/v2"
)

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
		if config.Aws != nil {
			os.Setenv("AWS_ACCESS_KEY_ID", config.Aws.AccessKeyId)
			os.Setenv("AWS_SECRET_ACCESS_KEY", config.Aws.SecretAccessKey)
		}
	} else {
		networkName := os.Getenv("CONFIG_NETWORK_NAME")
		if networkName == "" {
			log.Fatal("missing NETWORK_NAME environment variable")
		}

		gsheetId := os.Getenv("CONFIG_GSHEET_ID")
		if gsheetId == "" {
			log.Fatal("missing GSHEET_ID environment variable")
		}

		delegationWhitelistList := os.Getenv("DELEGATION_WHITELIST_LIST")
		if delegationWhitelistList == "" {
			log.Fatal("missing DELEGATION_WHITELIST_LIST environment variable")
		}

		delegationWhitelistColumn := os.Getenv("DELEGATION_WHITELIST_COLUMN")
		if delegationWhitelistColumn == "" {
			log.Fatal("missing DELEGATION_WHITELIST_COLUMN environment variable")
		}

		// AWS configurations
		if accessKeyId := os.Getenv("AWS_ACCESS_KEY_ID"); accessKeyId != "" {
			secretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
			if secretAccessKey == "" {
				log.Fatal("missing AWS_SECRET_ACCESS_KEY environment variable")
			}
			awsRegion := os.Getenv("CONFIG_AWS_REGION")
			if awsRegion == "" {
				log.Fatal("missing AWS_REGION environment variable")
			}
			awsAccountId := os.Getenv("CONFIG_AWS_ACCOUNT_ID")
			if awsAccountId == "" {
				log.Fatal("missing CONFIG_AWS_ACCOUNT_ID environment variable")
			}
			bucketNameSuffix := os.Getenv("CONFIG_BUCKET_NAME_SUFFIX")
			if bucketNameSuffix == "" {
				log.Fatal("missing CONFIG_BUCKET_NAME_SUFFIX environment variable")
			}

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
			databaseType := os.Getenv("CONFIG_DATABASE_TYPE")
			if databaseType == "" {
				log.Fatal("missing CONFIG_DATABASE_TYPE environment variable")
			}

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
