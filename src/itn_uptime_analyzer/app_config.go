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
    Options := [8]Option { NetworkName, AwsRegion, AwsAccountId, IgnoreIPs,
		StdOut, LocalOutput, S3Output, Period }
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
        for i := 0; i < len(Options); i++ {
            Options[i].updateJSON(log, &config)
        }
    } else {
        for i := 0; i < len(Options); i++ {
            Options[i].updateFromEnv(log, &config)
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

type OutputConfig struct {
    Stdout    bool    `json:"stdout"`
    Local     string  `json:"local"`
	S3Bucket  string  `json:"s3_bucket"`
    S3Key     string  `json:"s3"`
}

type AppConfig struct {
    Aws                    AwsConfig      `json:"aws"`
    NetworkName            string         `json:"network_name"`
    Period                 PeriodConfig   `json:"period"`
    IgnoreIPs              bool           `json:"ignore_ips"`
	Output                 OutputConfig   `json:"output"`
}

type AwsCredentials struct {
    AccessKeyId     string `json:"access_key_id"`
    SecretAccessKey string `json:"secret_access_key"`
}

type Option struct {
    updateJSON func (logging.EventLogger, *AppConfig)
    updateFromEnv func(logging.EventLogger, *AppConfig)
}

func noop(log logging.EventLogger, config *AppConfig) {}

func getEnvParsed[T any](log logging.EventLogger, parser func(string) (T, error), name string) *T {
    raw := os.Getenv(name)
    if raw == "" {
        return nil
    }
    parsed, err := parser(raw)
    if err != nil {
        log.Fatalf("Invalid %s environment variable (%v)", name, err)
    }
    return &parsed
}

func parseTime(raw string) (time.Time, error) {
    return time.Parse(time.RFC3339, raw)
}

func parseDuration(raw string) (time.Duration, error) {
    t, err := strconv.ParseInt(raw, 10, 64)
	ret := time.Duration(t) * time.Minute
	return ret, err
}

func unlessDefault[T comparable](value T, defaultVal T) *T {
    if value == defaultVal {
        return nil
    }
    return &value
}

func boolOption(envVar string, set func (bool, *AppConfig)) Option {
	return Option {
		updateJSON: noop,
		updateFromEnv: func (log logging.EventLogger, cfg *AppConfig) {
			raw := os.Getenv(envVar)
			if raw == "" || raw == "0" {
				set(false, cfg)
			} else if raw == "1" {
				set(true, cfg)
			} else {
				log.Fatalf("Unrecognised %s (should be either 0 or 1)!", envVar)
			}
		},
	}
}

func stringOption(envVar string, defVal *string, set func (string, *AppConfig)) Option {
	return Option {
		updateJSON: noop,
		updateFromEnv: func (log logging.EventLogger, cfg *AppConfig) {
			value := os.Getenv(envVar)
			switch {
				case value == "" && defVal != nil: set(*defVal, cfg)
				case value == "": log.Fatalf("Missing %s environment variable", envVar)
				default: set(value, cfg)
			}
		},
	}
}

var (
	empty = ""

    NetworkName = stringOption("CONFIG_NETWORK_NAME", nil, func (value string, cfg *AppConfig) {
		cfg.NetworkName = value
	})

    AwsRegion = stringOption("CONFIG_AWS_REGION", nil, func (value string, cfg *AppConfig) {
		cfg.Aws.Region = value
	})

    AwsAccountId = stringOption("CONFIG_AWS_ACCOUNT_ID", nil, func (value string, cfg *AppConfig) {
		cfg.Aws.AccountId = value
	})

    IgnoreIPs = boolOption("CONFIG_IGNORE_IPS", func (value bool, cfg *AppConfig) {
		cfg.IgnoreIPs = value
	})

	StdOut = boolOption("CONFIG_STDOUT", func (value bool, cfg *AppConfig) {
		cfg.Output.Stdout = value
	})

	LocalOutput = stringOption("CONFIG_LOCAL_OUTPUT", &empty, func (value string, cfg *AppConfig) {
		cfg.Output.Local = value
	})

	S3Output = Option {
		updateJSON: func (log logging.EventLogger, cfg *AppConfig) {
			if (cfg.Output.S3Bucket == "") != (cfg.Output.S3Key == "") {
				log.Fatalf("Either both or neither of S3 bucket and S3 key should be set!")
			}
		},
		updateFromEnv: func (log logging.EventLogger, cfg *AppConfig) {
			bucket := os.Getenv("CONFIG_S3_BUCKET")
			key := os.Getenv("CONFIG_S3_KEY")
			if (bucket == "") && (key == "") {
				cfg.Output.S3Bucket = bucket
				cfg.Output.S3Key = key
			} else {
				log.Fatalf("Either both or neither of S3 bucket and S3 key should be set!")
			}
		},
	}

    Period = Option {
        updateJSON: func (log logging.EventLogger, cfg *AppConfig) {
            // 1st Jan 0001 is the default value, which appears if it is absent
            // from the config file.
            start := unlessDefault(cfg.Period.Start, time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC))
            end := unlessDefault(cfg.Period.End, time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC))
            interval := unlessDefault(cfg.Period.Interval, time.Duration(0))

            cfg.Period = GetPeriodConfig(start, end, interval, log)
        },
        updateFromEnv: func (log logging.EventLogger, cfg *AppConfig) {
            start := getEnvParsed(log, parseTime, "CONFIG_PERIOD_START")
            end := getEnvParsed(log, parseTime, "CONFIG_PERIOD_END")
            interval := getEnvParsed(log, parseDuration, "CONFIG_PERIOD_INTERVAL")

            cfg.Period = GetPeriodConfig(start, end, interval, log)
        },
    }
)
