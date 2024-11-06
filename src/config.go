package main

import (
	"flag"
	"os"
	"slices"
	"strings"
)

const (
	ENV_PORT              = "BEMIDB_PORT"
	ENV_DATABASE          = "BEMIDB_DATABASE"
	ENV_INIT_SQL_FILEPATH = "BEMIDB_INIT_SQL"
	ENV_ICEBERG_PATH      = "BEMIDB_ICEBERG_PATH"
	ENV_LOG_LEVEL         = "BEMIDB_LOG_LEVEL"
	ENV_STORAGE_TYPE      = "BEMIDB_STORAGE_TYPE"

	ENV_AWS_REGION            = "BEMIDB_AWS_REGION"
	ENV_AWS_S3_BUCKET         = "BEMIDB_AWS_S3_BUCKET"
	ENV_AWS_ACCESS_KEY_ID     = "BEMIDB_AWS_ACCESS_KEY_ID"
	ENV_AWS_SECRET_ACCESS_KEY = "BEMIDB_AWS_SECRET_ACCESS_KEY"

	ENV_PG_DATABASE_URL = "PG_DATABASE_URL"

	DEFAULT_PORT              = "54321"
	DEFAULT_DATABASE          = "bemidb"
	DEFAULT_INIT_SQL_FILEPATH = "./init.sql"
	DEFAULT_ICEBERG_PATH      = "iceberg"
	DEFAULT_LOG_LEVEL         = "INFO"
	DEFAULT_DB_STORAGE_TYPE   = "LOCAL"
)

type Config struct {
	Port            string
	Database        string
	IcebergPath     string
	InitSqlFilepath string
	LogLevel        string
	StorageType     string
	PgDatabaseUrl   string
	Aws             AwsConfig
}

type AwsConfig struct {
	Region          string
	S3Bucket        string
	AccessKeyId     string
	SecretAccessKey string
}

var _config Config

func init() {
	registerFlags()
}

func registerFlags() {
	flag.StringVar(&_config.Port, "port", os.Getenv(ENV_PORT), "Port for BemiDB to listen on (default: "+DEFAULT_PORT+")")
	if _config.Port == "" {
		_config.Port = DEFAULT_PORT
	}

	flag.StringVar(&_config.Database, "database", os.Getenv(ENV_DATABASE), "Database name (default: "+DEFAULT_DATABASE+")")
	if _config.Database == "" {
		_config.Database = DEFAULT_DATABASE
	}

	flag.StringVar(&_config.IcebergPath, "iceberg-path", os.Getenv(ENV_ICEBERG_PATH), "Path to the Iceberg folder (default: "+DEFAULT_ICEBERG_PATH+")")
	if _config.IcebergPath == "" {
		_config.IcebergPath = DEFAULT_ICEBERG_PATH
	}

	flag.StringVar(&_config.InitSqlFilepath, "init-sql", os.Getenv(ENV_INIT_SQL_FILEPATH), "Path to the initialization SQL file (default: "+DEFAULT_INIT_SQL_FILEPATH+")")
	if _config.InitSqlFilepath == "" {
		_config.InitSqlFilepath = DEFAULT_INIT_SQL_FILEPATH
	}

	flag.StringVar(&_config.LogLevel, "log-level", os.Getenv(ENV_LOG_LEVEL), "Log level: DEBUG, INFO, ERROR (default: "+DEFAULT_LOG_LEVEL+")")
	if _config.LogLevel == "" {
		_config.LogLevel = DEFAULT_LOG_LEVEL
	} else if !slices.Contains(LOG_LEVELS, _config.LogLevel) {
		panic("Invalid log level " + _config.LogLevel + ". Must be one of " + strings.Join(LOG_LEVELS, ", "))
	}

	flag.StringVar(&_config.StorageType, "storage-type", os.Getenv(ENV_STORAGE_TYPE), "Storage type: LOCAL, AWS_S3 (default: "+DEFAULT_DB_STORAGE_TYPE+")")
	if _config.StorageType == "" {
		_config.StorageType = DEFAULT_DB_STORAGE_TYPE
	} else if !slices.Contains(STORAGE_TYPES, _config.StorageType) {
		panic("Invalid storage type " + _config.StorageType + ". Must be one of " + strings.Join(STORAGE_TYPES, ", "))
	}

	flag.StringVar(&_config.PgDatabaseUrl, "pg-database-url", os.Getenv(ENV_PG_DATABASE_URL), "PostgreSQL database URL")

	if _config.StorageType == STORAGE_TYPE_AWS_S3 {
		_config.Aws = AwsConfig{}

		flag.StringVar(&_config.Aws.Region, "aws-region", os.Getenv(ENV_AWS_REGION), "AWS region")
		if _config.Aws.Region == "" {
			panic("AWS region is required")
		}

		flag.StringVar(&_config.Aws.S3Bucket, "aws-s3-bucket", os.Getenv(ENV_AWS_S3_BUCKET), "AWS S3 bucket name")
		if _config.Aws.S3Bucket == "" {
			panic("AWS S3 bucket name is required")
		}

		flag.StringVar(&_config.Aws.AccessKeyId, "aws-access-key-id", os.Getenv(ENV_AWS_ACCESS_KEY_ID), "AWS access key ID")
		if _config.Aws.AccessKeyId == "" {
			panic("AWS access key ID is required")
		}

		flag.StringVar(&_config.Aws.SecretAccessKey, "aws-secret-access-key", os.Getenv(ENV_AWS_SECRET_ACCESS_KEY), "AWS secret access key")
		if _config.Aws.SecretAccessKey == "" {
			panic("AWS secret access key is required")
		}
	}
}

func LoadConfig(reRegisterFlags ...bool) *Config {
	if reRegisterFlags != nil && reRegisterFlags[0] {
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		registerFlags()
	}
	return &_config
}