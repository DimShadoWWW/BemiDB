package main

import (
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Run("Uses default config values with local storage", func(t *testing.T) {
		config := LoadConfig(true)

		if config.Port != "54321" {
			t.Errorf("Expected port to be 54321, got %s", config.Port)
		}
		if config.Database != "bemidb" {
			t.Errorf("Expected database to be bemidb, got %s", config.Database)
		}
		if config.InitSqlFilepath != "./init.sql" {
			t.Errorf("Expected duckdbInitFilepath to be ./init.sql, got %s", config.InitSqlFilepath)
		}
		if config.StoragePath != "iceberg" {
			t.Errorf("Expected StoragePath to be iceberg, got %s", config.StoragePath)
		}
		if config.LogLevel != "INFO" {
			t.Errorf("Expected logLevel to be INFO, got %s", config.LogLevel)
		}
		if config.StorageType != "LOCAL" {
			t.Errorf("Expected storageType to be LOCAL, got %s", config.StorageType)
		}
		if config.Pg.DatabaseUrl != "" {
			t.Errorf("Expected pgDatabaseUrl to be empty, got %s", config.Pg.DatabaseUrl)
		}
		if config.Pg.SyncInterval != "" {
			t.Errorf("Expected interval to be empty, got %s", config.Pg.SyncInterval)
		}
		if config.Pg.SchemaPrefix != "" {
			t.Errorf("Expected schemaPrefix to be empty, got %s", config.Pg.SchemaPrefix)
		}
	})

	t.Run("Uses config values from environment variables with LOCAL storage", func(t *testing.T) {
		t.Setenv("BEMIDB_PORT", "12345")
		t.Setenv("BEMIDB_DATABASE", "mydb")
		t.Setenv("BEMIDB_INIT_SQL", "./init/duckdb.sql")
		t.Setenv("BEMIDB_STORAGE_PATH", "storage-path")
		t.Setenv("BEMIDB_LOG_LEVEL", "ERROR")
		t.Setenv("BEMIDB_STORAGE_TYPE", "LOCAL")

		config := LoadConfig(true)

		if config.Port != "12345" {
			t.Errorf("Expected port to be 12345, got %s", config.Port)
		}
		if config.Database != "mydb" {
			t.Errorf("Expected database to be mydb, got %s", config.Database)
		}
		if config.InitSqlFilepath != "./init/duckdb.sql" {
			t.Errorf("Expected duckdbInitFilepath to be ./init/duckdb.sql, got %s", config.InitSqlFilepath)
		}
		if config.StoragePath != "storage-path" {
			t.Errorf("Expected StoragePath to be storage-path, got %s", config.StoragePath)
		}
		if config.LogLevel != "ERROR" {
			t.Errorf("Expected logLevel to be ERROR, got %s", config.LogLevel)
		}
		if config.StorageType != "LOCAL" {
			t.Errorf("Expected storageType to be local, got %s", config.StorageType)
		}
	})

	t.Run("Uses config values from environment variables with AWS S3 storage", func(t *testing.T) {
		t.Setenv("BEMIDB_PORT", "12345")
		t.Setenv("BEMIDB_DATABASE", "mydb")
		t.Setenv("BEMIDB_INIT_SQL", "./init/duckdb.sql")
		t.Setenv("BEMIDB_STORAGE_PATH", "storage-path")
		t.Setenv("BEMIDB_LOG_LEVEL", "ERROR")
		t.Setenv("BEMIDB_STORAGE_TYPE", "S3")
		t.Setenv("AWS_REGION", "us-west-1")
		t.Setenv("AWS_S3_BUCKET", "my_bucket")
		t.Setenv("AWS_ACCESS_KEY_ID", "my_access_key_id")
		t.Setenv("AWS_SECRET_ACCESS_KEY", "my_secret_access_key")

		config := LoadConfig(true)

		if config.Port != "12345" {
			t.Errorf("Expected port to be 12345, got %s", config.Port)
		}
		if config.Database != "mydb" {
			t.Errorf("Expected database to be mydb, got %s", config.Database)
		}
		if config.InitSqlFilepath != "./init/duckdb.sql" {
			t.Errorf("Expected duckdbInitFilepath to be ./init/duckdb.sql, got %s", config.InitSqlFilepath)
		}
		if config.StoragePath != "storage-path" {
			t.Errorf("Expected StoragePath to be storage-path, got %s", config.StoragePath)
		}
		if config.LogLevel != "ERROR" {
			t.Errorf("Expected logLevel to be ERROR, got %s", config.LogLevel)
		}
		if config.StorageType != "S3" {
			t.Errorf("Expected storageType to be S3, got %s", config.StorageType)
		}
		if config.Aws.Region != "us-west-1" {
			t.Errorf("Expected awsRegion to be us-west-1, got %s", config.Aws.Region)
		}
		if config.Aws.S3Bucket != "my_bucket" {
			t.Errorf("Expected awsS3Bucket to be mybucket, got %s", config.Aws.S3Bucket)
		}
		if config.Aws.AccessKeyId != "my_access_key_id" {
			t.Errorf("Expected awsAccessKeyId to be my_access_key_id, got %s", config.Aws.AccessKeyId)
		}
		if config.Aws.SecretAccessKey != "my_secret_access_key" {
			t.Errorf("Expected awsSecretAccessKey to be my_secret_access_key, got %s", config.Aws.SecretAccessKey)
		}
	})

	t.Run("Uses config values from environment variables for PG", func(t *testing.T) {
		t.Setenv("PG_DATABASE_URL", "postgres://user:password@localhost:5432/template1")
		t.Setenv("PG_SYNC_INTERVAL", "1h")
		t.Setenv("PG_SCHEMA_PREFIX", "mydb_")

		config := LoadConfig(true)

		if config.Pg.DatabaseUrl != "postgres://user:password@localhost:5432/template1" {
			t.Errorf("Expected pgDatabaseUrl to be postgres://user:password@localhost:5432/template1, got %s", config.Pg.DatabaseUrl)
		}
		if config.Pg.SyncInterval != "1h" {
			t.Errorf("Expected interval to be 1h, got %s", config.Pg.SyncInterval)
		}
		if config.Pg.SchemaPrefix != "mydb_" {
			t.Errorf("Expected schemaPrefix to be empty, got %s", config.Pg.SchemaPrefix)
		}
	})

	t.Run("Uses command line arguments", func(t *testing.T) {
		setTestArgs([]string{
			"--port", "12345",
			"--database", "mydb",
			"--init-sql", "./init/duckdb.sql",
			"--storage-path", "storage-path",
			"--log-level", "ERROR",
			"--storage-type", "local",
			"--pg-database-url", "postgres://user:password@localhost:5432/db",
			"--pg-sync-interval", "2h30m",
			"--pg-schema-prefix", "mydb_",
			"--only-tables", "table1,table2",
		})

		config := LoadConfig()

		if config.Port != "12345" {
			t.Errorf("Expected port to be 12345, got %s", config.Port)
		}
		if config.Database != "mydb" {
			t.Errorf("Expected database to be mydb, got %s", config.Database)
		}
		if config.InitSqlFilepath != "./init/duckdb.sql" {
			t.Errorf("Expected duckdbInitFilepath to be ./init/duckdb.sql, got %s", config.InitSqlFilepath)
		}
		if config.StoragePath != "storage-path" {
			t.Errorf("Expected StoragePath to be storage-path, got %s", config.StoragePath)
		}
		if config.LogLevel != "ERROR" {
			t.Errorf("Expected logLevel to be ERROR, got %s", config.LogLevel)
		}
		if config.StorageType != "local" {
			t.Errorf("Expected storageType to be local, got %s", config.StorageType)
		}
		if config.Pg.DatabaseUrl != "postgres://user:password@localhost:5432/db" {
			t.Errorf("Expected pgDatabaseUrl to be postgres://user:password@localhost:5432/db, got %s", config.Pg.DatabaseUrl)
		}
		if config.Pg.SyncInterval != "2h30m" {
			t.Errorf("Expected interval to be 2h30m, got %s", config.Pg.SyncInterval)
		}
		if config.Pg.SchemaPrefix != "mydb_" {
			t.Errorf("Expected schemaPrefix to be mydb_, got %s", config.Pg.SchemaPrefix)
		}
		if config.OnlyTables != "table1,table2" {
			t.Errorf("Expected onlyTables to be table1,table2, got %s", config.OnlyTables)
		}
	})
}
