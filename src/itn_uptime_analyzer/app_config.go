package itn_uptime_analyzer

import (
	"encoding/json"
	"os"
	"strconv"
	"time"

	logging "github.com/ipfs/go-log/v2"
)

func loadAwsCredentials(filename string, log logging.EventLogger) {
	file, err := os.Open(filename)
	if err != nil {
		log.Errorf("Error loading credentials file: %s", err)
		os.Exit(1)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	var credentials AwsCredentials
	err = decoder.Decode(&credentials)
	if err != nil {
		log.Errorf("Error loading credentials file: %s", err)
		os.Exit(1)
	}
	os.Setenv("AWS_ACCESS_KEY_ID", credentials.AccessKeyId)
	os.Setenv("AWS_SECRET_ACCESS_KEY", credentials.SecretAccessKey)
}

func LoadEnv(log logging.EventLogger) AppConfig {
	var config AppConfig

	configFile := os.Getenv("CONFIG_FILE")
	if configFile != "" {
		file, err := os.Open(configFile)
		if err != nil {
			log.Errorf("Error loading config file: %s", err)
			os.Exit(1)
		}
		defer file.Close()
		decoder := json.NewDecoder(file)
		err = decoder.Decode(&config)
		if err != nil {
			log.Errorf("Error loading config file: %s", err)
			os.Exit(1)
		}

        var start *time.Time
        var end *time.Time
		var interval *time.Duration

		// 1st Jan 0001 is the default value, which appears if it is absent
		// from the config file.
		if config.Period.End.Year() == 1 {
			end = nil
		} else {
			end = &config.Period.End
		}
		if config.Period.Start.Year() == 1 {
			start = nil
		} else {
			start = &config.Period.Start
		}
		if config.Period.Interval == 0 {
			interval = nil
		} else {
			interval = &config.Period.Interval
		}
		config.Period = GetPeriodConfig(start, end, interval, log)
    } else {
        var start *time.Time
        var end *time.Time
		var interval *time.Duration

		networkName := os.Getenv("CONFIG_NETWORK_NAME")
		if networkName == "" {
			log.Fatal("missing NETWORK_NAME environment variable")
		}

		awsRegion := os.Getenv("CONFIG_AWS_REGION")
		if awsRegion == "" {
			log.Fatal("missing AWS_REGION environment variable")
		}

		awsAccountId := os.Getenv("CONFIG_AWS_ACCOUNT_ID")
		if awsAccountId == "" {
			log.Fatal("missing AWS_ACCOUNT_ID environment variable")
		}

	    startRaw := os.Getenv("CONFIG_PERIOD_START")
	    if startRaw == "" {
			start = nil
	    } else {
			startParsed, err := time.Parse(time.RFC3339, startRaw)
			if err != nil {
				log.Fatalf("invalid CONFIG_PERIOD_START environment variable (%v)", err)
			}
			start = &startParsed
		}

	    endRaw := os.Getenv("CONFIG_PERIOD_END")
	    if endRaw == "" {
			end = nil
	    } else {
			endParsed, err := time.Parse(time.RFC3339, endRaw)
			if err != nil {
				log.Fatalf("invalid CONFIG_PERIOD_END environment variable (%v)", err)
			}
			end = &endParsed
		}

        intervalRaw := os.Getenv("CONFIG_PERIOD_INTERVAL")
	    if intervalRaw == "" {
			interval = nil
		} else {
			intervalInt, err := strconv.ParseInt(intervalRaw, 10, 64)
			if err != nil {
				log.Fatal("CONFIG_PERIOD_INTERVAL specified, but not an integer (%v)!", err)
			}
            intervalParsed := time.Duration(intervalInt)
			interval = &intervalParsed
		}
        period := GetPeriodConfig(start, end, interval, log)

		config = AppConfig{
			NetworkName:            networkName,
			Aws: AwsConfig{
				Region:    awsRegion,
				AccountId: awsAccountId,
			},
			Period: period,
		}
	}

	awsCredentialsFile := os.Getenv("AWS_CREDENTIALS_FILE")
	if awsCredentialsFile != "" {
		loadAwsCredentials(awsCredentialsFile, log)
	}

return config
}

type AwsConfig struct {
	Region    string `json:"region"`
	AccountId string `json:"account_id"`
}

type AppConfig struct {
	Aws                    AwsConfig      `json:"aws"`
	NetworkName            string         `json:"network_name"`
	Period                 PeriodConfig   `json:"period"`
}

type AwsCredentials struct {
	AccessKeyId     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
}
