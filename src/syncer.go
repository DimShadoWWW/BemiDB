package main

import (
	"context"
	"encoding/csv"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
)

const (
	BATCH_SIZE = 10000
)

type Syncer struct {
	config        *Config
	icebergWriter *IcebergWriter
	icebergReader *IcebergReader
}

func NewSyncer(config *Config) *Syncer {
	if config.Pg.DatabaseUrl == "" {
		panic("Missing PostgreSQL database URL")
	}

	icebergWriter := NewIcebergWriter(config)
	icebergReader := NewIcebergReader(config)
	return &Syncer{config: config, icebergWriter: icebergWriter, icebergReader: icebergReader}
}

func (syncer *Syncer) SyncFromPostgres() {
	ctx := context.Background()

	onlyTablesMap := make(map[string]bool)
	if syncer.config.OnlyTables != "*" {
		entries := strings.Split(syncer.config.OnlyTables, ",")
		for _, e := range entries {
			onlyTablesMap[e] = false
		}
	}

	conn, err := pgx.Connect(ctx, syncer.config.Pg.DatabaseUrl)
	PanicIfError(err)
	defer conn.Close(ctx)

	_, err = conn.Exec(ctx, "BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ ONLY DEFERRABLE")
	PanicIfError(err)

	pgSchemaTables := []SchemaTable{}
	for _, schema := range syncer.listPgSchemas(conn) {
		for _, pgSchemaTable := range syncer.listPgSchemaTables(conn, schema) {
			tableName := pgSchemaTable.Table
			syncTable := tableName == "*"
			if _, ok := onlyTablesMap[tableName]; ok {
				syncTable = true
				onlyTablesMap[tableName] = true
			}
			if syncTable {
				pgSchemaTables = append(pgSchemaTables, pgSchemaTable)
				syncer.syncFromPgTable(conn, pgSchemaTable)
			}
		}
	}
	syncer.deleteOldIcebergSchemaTables(pgSchemaTables)
}

func (syncer *Syncer) listPgSchemas(conn *pgx.Conn) []string {
	var schemas []string

	schemasRows, err := conn.Query(
		context.Background(),
		"SELECT schema_name FROM information_schema.schemata WHERE schema_name NOT IN ('pg_catalog', 'pg_toast', 'information_schema')",
	)
	PanicIfError(err)
	defer schemasRows.Close()

	for schemasRows.Next() {
		var schema string
		err = schemasRows.Scan(&schema)
		PanicIfError(err)
		schemas = append(schemas, schema)
	}

	return schemas
}

func (syncer *Syncer) listPgSchemaTables(conn *pgx.Conn, schema string) []SchemaTable {
	var pgSchemaTables []SchemaTable

	tablesRows, err := conn.Query(
		context.Background(),
		`
		SELECT pg_class.relname AS table, COALESCE(parent.relname, '') AS parent_partitioned_table
		FROM pg_class
		JOIN pg_namespace ON pg_namespace.oid = pg_class.relnamespace
		LEFT JOIN pg_inherits ON pg_inherits.inhrelid = pg_class.oid
		LEFT JOIN pg_class AS parent ON pg_inherits.inhparent = parent.oid
		WHERE pg_namespace.nspname = $1 AND pg_class.relkind = 'r';
		`,
		schema,
	)
	PanicIfError(err)
	defer tablesRows.Close()

	for tablesRows.Next() {
		pgSchemaTable := SchemaTable{Schema: schema}
		err = tablesRows.Scan(&pgSchemaTable.Table, &pgSchemaTable.ParentPartitionedTable)
		PanicIfError(err)
		pgSchemaTables = append(pgSchemaTables, pgSchemaTable)
	}

	return pgSchemaTables
}

func (syncer *Syncer) syncFromPgTable(conn *pgx.Conn, pgSchemaTable SchemaTable) {
	LogInfo(syncer.config, "Syncing "+pgSchemaTable.String()+"...")

	csvFile, err := syncer.exportPgTableToCsv(conn, pgSchemaTable)
	PanicIfError(err)
	defer csvFile.Close()

	csvReader := csv.NewReader(csvFile)
	csvHeader, err := csvReader.Read()
	PanicIfError(err)

	pgSchemaColumns := syncer.pgTableSchemaColumns(conn, pgSchemaTable, csvHeader)
	reachedEnd := false

	syncer.icebergWriter.Write(pgSchemaTable, pgSchemaColumns, func() [][]string {
		if reachedEnd {
			return [][]string{}
		}

		var rows [][]string
		for {
			row, err := csvReader.Read()
			if err != nil {
				reachedEnd = true
				break
			}

			rows = append(rows, row)
			if len(rows) >= BATCH_SIZE {
				break
			}
		}
		return rows
	})
}

func (syncer *Syncer) pgTableSchemaColumns(conn *pgx.Conn, pgSchemaTable SchemaTable, csvHeader []string) []PgSchemaColumn {
	var pgSchemaColumns []PgSchemaColumn

	rows, err := conn.Query(
		context.Background(),
		`SELECT
			column_name,
			data_type,
			udt_name,
			is_nullable,
			ordinal_position,
			COALESCE(character_maximum_length, 0),
			COALESCE(numeric_precision, 0),
			COALESCE(numeric_scale, 0),
			COALESCE(datetime_precision, 0)
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY array_position($3, column_name)`,
		pgSchemaTable.Schema,
		pgSchemaTable.Table,
		csvHeader,
	)
	PanicIfError(err)
	defer rows.Close()

	for rows.Next() {
		var pgSchemaColumn PgSchemaColumn
		err = rows.Scan(
			&pgSchemaColumn.ColumnName,
			&pgSchemaColumn.DataType,
			&pgSchemaColumn.UdtName,
			&pgSchemaColumn.IsNullable,
			&pgSchemaColumn.OrdinalPosition,
			&pgSchemaColumn.CharacterMaximumLength,
			&pgSchemaColumn.NumericPrecision,
			&pgSchemaColumn.NumericScale,
			&pgSchemaColumn.DatetimePrecision,
		)
		PanicIfError(err)
		pgSchemaColumns = append(pgSchemaColumns, pgSchemaColumn)
	}

	return pgSchemaColumns
}

func (syncer *Syncer) exportPgTableToCsv(conn *pgx.Conn, pgSchemaTable SchemaTable) (csvFile *os.File, err error) {
	tempFile, err := CreateTemporaryFile(pgSchemaTable.String())
	PanicIfError(err)
	defer DeleteTemporaryFile(tempFile)

	result, err := conn.PgConn().CopyTo(
		context.Background(),
		tempFile,
		"COPY "+pgSchemaTable.String()+" TO STDOUT WITH CSV HEADER NULL '"+PG_NULL_STRING+"'",
	)
	PanicIfError(err)
	LogDebug(syncer.config, "Copied", result.RowsAffected(), "row(s) into", tempFile.Name())

	return os.Open(tempFile.Name())
}

func (syncer *Syncer) deleteOldIcebergSchemaTables(pgSchemaTables []SchemaTable) {
	var prefixedPgSchemaTables []SchemaTable
	for _, pgSchemaTable := range pgSchemaTables {
		prefixedPgSchemaTables = append(
			prefixedPgSchemaTables,
			SchemaTable{Schema: syncer.config.Pg.SchemaPrefix + pgSchemaTable.Schema, Table: pgSchemaTable.Table},
		)
	}

	icebergSchemas, err := syncer.icebergReader.Schemas()
	PanicIfError(err)

	for _, icebergSchema := range icebergSchemas {
		found := false
		for _, pgSchemaTable := range prefixedPgSchemaTables {
			if icebergSchema == pgSchemaTable.Schema {
				found = true
				break
			}
		}

		if !found {
			LogInfo(syncer.config, "Deleting", icebergSchema, "...")
			syncer.icebergWriter.DeleteSchema(icebergSchema)
		}
	}

	icebergSchemaTables, err := syncer.icebergReader.SchemaTables()
	PanicIfError(err)

	for _, icebergSchemaTable := range icebergSchemaTables {
		found := false
		for _, pgSchemaTable := range prefixedPgSchemaTables {
			if icebergSchemaTable.String() == pgSchemaTable.String() {
				found = true
				break
			}
		}

		if !found {
			LogInfo(syncer.config, "Deleting", icebergSchemaTable.String(), "...")
			syncer.icebergWriter.DeleteSchemaTable(icebergSchemaTable)
		}
	}
}
