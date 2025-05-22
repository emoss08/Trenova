package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/resourcesqltype"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

// ResourceEditorRepositoryParams defines dependencies required for initializing the ResourceEditorRepository.
// This includes database connection, logger, and resource editor repository.
type ResourceEditorRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

// resourceEditorRepository implements the ResourceEditorRepository interface
// and provides methods to manage resource editor, including fetching table schema,
// columns, indexes, and constraints.
type resourceEditorRepository struct {
	db     db.Connection
	logger *zerolog.Logger
}

// tableAlias holds the resolved schema and table name for a given alias found in
// a SQL query. The schema can be empty when the default schema should be
// assumed.
type tableAlias struct {
	Schema string
	Table  string
}

// parseTableAliases scans a SQL query for FROM/JOIN clauses and extracts table
// aliases. It returns a map keyed by the alias where the value contains the
// resolved schema (if provided) and table name.
//
// Examples recognised by the regex:
//
//	FROM public.users u
//	JOIN shipments s ON ...
//	FROM orders -- no alias, the table name itself becomes an alias
//
// The algorithm is intentionally simple – it does not aim to fully parse SQL
// but covers the majority of ad-hoc queries written in the resource editor.
func parseTableAliases(query string) map[string]tableAlias {
	aliasMap := make(map[string]tableAlias)

	// * (?i) – case-insensitive.
	// * first capture group is FROM|JOIN, second is the table (optionally
	// * schema.table), third is the alias (optional).
	re := regexp.MustCompile(`(?i)(?:from|join)\s+([^\s]+)(?:\s+(?:as\s+)?([a-zA-Z0-9_]+))?`)
	matches := re.FindAllStringSubmatch(query, -1)

	for _, m := range matches {
		if len(m) < 3 {
			continue
		}

		fullTable := strings.Trim(m[1], ",;()") // * may include schema, trim punctuation
		alias := m[2]                           // * may be empty

		var schemaName, tableName string
		parts := strings.Split(fullTable, ".")
		switch len(parts) {
		case 2:
			schemaName = parts[0]
			tableName = parts[1]
		default:
			tableName = fullTable
		}

		// * If no alias specified we use the table name itself as the alias so
		// * that "users." will still resolve.
		if alias == "" {
			alias = tableName
		}

		aliasMap[alias] = tableAlias{Schema: schemaName, Table: tableName}
		// * Also store the unqualified table name so we can resolve
		// * `schema.table` references without alias.
		aliasMap[tableName] = tableAlias{Schema: schemaName, Table: tableName}
	}

	return aliasMap
}

// NewResourceEditorRepository initializes a new instance of resourceEditorRepository with its dependencies.
//
// Parameters:
//   - p: ResourceEditorRepositoryParams containing dependencies.
//
// Returns:
//   - repositories.ResourceEditorRepository: A ready-to-use resource editor repository instance.
func NewResourceEditorRepository(p ResourceEditorRepositoryParams) repositories.ResourceEditorRepository {
	log := p.Logger.With().
		Str("repository", "resourceeditor").
		Logger()

	return &resourceEditorRepository{
		db:     p.DB,
		logger: &log,
	}
}

// GetTableSchema fetches the schema information for a given schema name.
//
// Parameters:
//   - ctx: The context for the database operation.
//   - schemaName: The name of the schema to fetch.
//
// Returns:
//   - *repositories.SchemaInformation: The schema information for the given schema name.
func (r *resourceEditorRepository) GetTableSchema(ctx context.Context, schemaName string) (*repositories.SchemaInformation, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to get database connection")
		return nil, eris.Wrap(err, "failed to get database connection")
	}

	if schemaName == "" {
		schemaName = "public" // * Default to public if not specified
	}
	r.logger.Info().Str("schemaName", schemaName).Msg("Fetching schema information")

	schemaInfo := &repositories.SchemaInformation{
		SchemaName: schemaName,
		Tables:     []repositories.TableDetails{},
	}

	tableRows, err := dba.QueryContext(ctx, `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = ? AND table_type = 'BASE TABLE'
		ORDER BY table_name;
	`, schemaName)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to fetch tables")
		return nil, eris.Wrap(err, "failed to fetch tables")
	}
	defer tableRows.Close()

	var tableNames []string
	for tableRows.Next() {
		var tableName string
		if err = tableRows.Scan(&tableName); err != nil {
			r.logger.Error().Err(err).Msg("Failed to scan table name")
			return nil, eris.Wrap(err, "failed to scan table name")
		}
		tableNames = append(tableNames, tableName)
	}

	if err = tableRows.Err(); err != nil {
		r.logger.Error().Err(err).Msg("Error iterating table rows")
		return nil, eris.Wrap(err, "error iterating table rows")
	}

	if err = tableRows.Close(); err != nil {
		r.logger.Error().Err(err).Msg("Error closing table rows")
		return nil, eris.Wrap(err, "error closing table rows")
	}

	for _, tableName := range tableNames {
		r.logger.Debug().Str("tableName", tableName).Msg("Fetching details for table")
		tableDetail := repositories.TableDetails{
			TableName: tableName,
		}

		columns, cErr := r.fetchColumnsForTable(ctx, schemaName, tableName)
		if cErr != nil {
			// Log already happens in fetchColumnsForTable or here
			return nil, eris.Wrapf(cErr, "failed to fetch columns for table %s", tableName)
		}
		tableDetail.Columns = columns

		indexes, iErr := r.fetchIndexesForTable(ctx, schemaName, tableName)
		if iErr != nil {
			return nil, eris.Wrapf(iErr, "failed to fetch indexes for table %s", tableName)
		}
		tableDetail.Indexes = indexes

		constraints, coErr := r.fetchConstraintsForTable(ctx, schemaName, tableName)
		if coErr != nil {
			return nil, eris.Wrapf(coErr, "failed to fetch constraints for table %s", tableName)
		}
		tableDetail.Constraints = constraints

		schemaInfo.Tables = append(schemaInfo.Tables, tableDetail)
	}

	r.logger.Info().Str("schemaName", schemaName).Msg("Successfully fetched schema information")
	return schemaInfo, nil
}

// fetchColumnsForTable fetches the column details for a given table.
//
// Parameters:
//   - ctx: The context for the database operation.
//   - schemaName: The name of the schema to fetch.
//   - tableName: The name of the table to fetch.
//
// Returns:
//   - []repositories.ColumnDetails: The column details for the given table.
//   - error: An error if the operation fails.
func (r *resourceEditorRepository) fetchColumnsForTable(ctx context.Context, schemaName string, tableName string) ([]repositories.ColumnDetails, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to get database connection")
		return nil, eris.Wrap(err, "failed to get database connection")
	}

	query := `
	SELECT
		c.column_name,
		c.ordinal_position,
		c.column_default,
		c.is_nullable,
		c.data_type,
		c.character_maximum_length,
		c.numeric_precision,
		c.numeric_scale,
		pgd.description
	FROM
		information_schema.columns c
	LEFT JOIN
		pg_catalog.pg_class tbl ON tbl.relname = c.table_name AND tbl.relnamespace = (SELECT oid FROM pg_catalog.pg_namespace WHERE nspname = c.table_schema)
	LEFT JOIN
		pg_catalog.pg_description pgd ON (pgd.objoid = tbl.oid AND pgd.objsubid = c.ordinal_position)
	WHERE
		c.table_schema = ? AND c.table_name = ?
	ORDER BY
		c.ordinal_position;
	`
	rows, err := dba.QueryContext(ctx, query, schemaName, tableName)
	if err != nil {
		r.logger.Error().Err(err).Str("schemaName", schemaName).Str("tableName", tableName).Msg("Querying columns failed")
		return nil, eris.Wrap(err, "querying columns failed")
	}
	defer rows.Close()

	var columns []repositories.ColumnDetails
	for rows.Next() {
		var col repositories.ColumnDetails
		var charMaxLen, numPrecision, numScale sql.NullInt64
		var colDefault, colComment sql.NullString

		err = rows.Scan(
			&col.ColumnName,
			&col.OrdinalPosition,
			&colDefault,
			&col.IsNullable,
			&col.DataType,
			&charMaxLen,
			&numPrecision,
			&numScale,
			&colComment,
		)
		if err != nil {
			r.logger.Error().Err(err).Msg("Scanning column row failed")
			return nil, eris.Wrap(err, "scanning column row failed")
		}
		if colDefault.Valid {
			col.ColumnDefault = &colDefault.String
		}
		if charMaxLen.Valid {
			col.CharacterMaximumLength = &charMaxLen.Int64
		}
		if numPrecision.Valid {
			col.NumericPrecision = &numPrecision.Int64
		}
		if numScale.Valid {
			col.NumericScale = &numScale.Int64
		}
		if colComment.Valid {
			col.Comment = &colComment.String
		}
		columns = append(columns, col)
	}
	if err = rows.Err(); err != nil {
		r.logger.Error().Err(err).Msg("Error iterating column rows")
		return nil, eris.Wrap(err, "error iterating column rows")
	}
	return columns, nil
}

// fetchIndexesForTable fetches the index details for a given table.
//
// Parameters:
//   - ctx: The context for the database operation.
//   - schemaName: The name of the schema to fetch.
//   - tableName: The name of the table to fetch.
//
// Returns:
//   - []repositories.IndexDetails: The index details for the given table.
//   - error: An error if the operation fails.
func (r *resourceEditorRepository) fetchIndexesForTable(ctx context.Context, schemaName string, tableName string) ([]repositories.IndexDetails, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to get database connection")
		return nil, eris.Wrap(err, "failed to get database connection")
	}

	query := `
    SELECT
        i.relname as index_name,
        idx.indisunique AS is_unique,
        idx.indisprimary AS is_primary,
        pg_get_indexdef(idx.indexrelid) as index_definition,
        a.attname AS column_name,
        am.amname AS index_type
    FROM
        pg_catalog.pg_class t
        JOIN pg_catalog.pg_index idx ON t.oid = idx.indrelid
        JOIN pg_catalog.pg_class i ON i.oid = idx.indexrelid
        JOIN pg_catalog.pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(idx.indkey) AND NOT a.attisdropped
        JOIN pg_catalog.pg_namespace n ON n.oid = t.relnamespace
        LEFT JOIN pg_catalog.pg_am am ON i.relam = am.oid
    WHERE
        t.relkind = 'r'
        AND t.relname = ?
        AND n.nspname = ?
    ORDER BY index_name, array_position(idx.indkey, a.attnum);
    `
	rows, err := dba.QueryContext(ctx, query, tableName, schemaName)
	if err != nil {
		r.logger.Error().Err(err).Str("schemaName", schemaName).Str("tableName", tableName).Msg("Querying indexes failed")
		return nil, eris.Wrap(err, "querying indexes failed")
	}
	defer rows.Close()

	indexMap := make(map[string]*repositories.IndexDetails)
	var orderedIndexNames []string

	for rows.Next() {
		var indexName, indexDef, columnName, indexType sql.NullString // * indexType can be null for some index kinds
		var isUnique, isPrimary bool
		err = rows.Scan(&indexName, &isUnique, &isPrimary, &indexDef, &columnName, &indexType)
		if err != nil {
			r.logger.Error().Err(err).Msg("Scanning index row failed")
			return nil, eris.Wrap(err, "scanning index row failed")
		}

		// Ensure indexName, indexDef, and columnName are valid
		if !indexName.Valid || !indexDef.Valid || !columnName.Valid {
			r.logger.Warn().Msg("Skipping index row due to NULL essential fields (indexName, indexDef, or columnName)")
			continue
		}

		idxNameStr := indexName.String
		if _, exists := indexMap[idxNameStr]; !exists {
			indexMap[idxNameStr] = &repositories.IndexDetails{
				IndexName:       idxNameStr,
				IndexDefinition: indexDef.String,
				IsUnique:        isUnique,
				IsPrimary:       isPrimary,
				IndexType:       "", // * Default, will be set if valid
				Columns:         []string{},
			}
			if indexType.Valid {
				indexMap[idxNameStr].IndexType = indexType.String
			}
			orderedIndexNames = append(orderedIndexNames, idxNameStr)
		}
		indexMap[idxNameStr].Columns = append(indexMap[idxNameStr].Columns, columnName.String)
	}
	if err = rows.Err(); err != nil {
		r.logger.Error().Err(err).Msg("Error iterating index rows")
		return nil, eris.Wrap(err, "error iterating index rows")
	}

	var indexes []repositories.IndexDetails
	for _, name := range orderedIndexNames {
		indexes = append(indexes, *indexMap[name])
	}
	return indexes, nil
}

// fetchConstraintsForTable fetches the constraint details for a given table.
//
// Parameters:
//   - ctx: The context for the database operation.
//   - schemaName: The name of the schema to fetch.
//   - tableName: The name of the table to fetch.
//
// Returns:
//   - []repositories.ConstraintDetails: The constraint details for the given table.
//   - error: An error if the operation fails.
func (r *resourceEditorRepository) fetchConstraintsForTable(ctx context.Context, schemaName string, tableName string) ([]repositories.ConstraintDetails, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to get database connection")
		return nil, eris.Wrap(err, "failed to get database connection")
	}

	constraintsMap := make(map[string]*repositories.ConstraintDetails)
	var orderedConstraintNames []string

	// * Query for PK, UNIQUE, FK (base info)
	keyConstraintsQuery := `
    SELECT
        tc.constraint_name,
        tc.constraint_type,
        kcu.column_name,
        tc.is_deferrable,
        tc.initially_deferred
    FROM
        information_schema.table_constraints tc
    JOIN
        information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name AND tc.table_schema = kcu.table_schema AND tc.table_name = kcu.table_name
    WHERE
        tc.table_schema = ? AND tc.table_name = ?
        AND tc.constraint_type IN ('PRIMARY KEY', 'FOREIGN KEY', 'UNIQUE')
    ORDER BY
        tc.constraint_name, kcu.ordinal_position;
    `
	keyRows, err := dba.QueryContext(ctx, keyConstraintsQuery, schemaName, tableName)
	if err != nil {
		r.logger.Error().Err(err).Str("schemaName", schemaName).Str("tableName", tableName).Msg("Querying key constraints failed")
		return nil, eris.Wrap(err, "querying key constraints failed")
	}
	defer keyRows.Close()

	for keyRows.Next() {
		var consName, consType, colName, isDeferrableStr, initiallyDeferredStr string
		if err = keyRows.Scan(&consName, &consType, &colName, &isDeferrableStr, &initiallyDeferredStr); err != nil {
			r.logger.Error().Err(err).Msg("Scanning key constraint row failed")
			return nil, eris.Wrap(err, "scanning key constraint row")
		}
		if _, exists := constraintsMap[consName]; !exists {
			constraintsMap[consName] = &repositories.ConstraintDetails{
				ConstraintName:    consName,
				ConstraintType:    consType,
				ColumnNames:       []string{},
				Deferrable:        isDeferrableStr == string(resourcesqltype.KeywordYes),
				InitiallyDeferred: initiallyDeferredStr == string(resourcesqltype.KeywordYes),
			}
			orderedConstraintNames = append(orderedConstraintNames, consName)
		}
		constraintsMap[consName].ColumnNames = append(constraintsMap[consName].ColumnNames, colName)
	}
	if err = keyRows.Err(); err != nil {
		r.logger.Error().Err(err).Msg("Error iterating key constraint rows")
		return nil, eris.Wrap(err, "error iterating key constraint rows")
	}
	if err = keyRows.Close(); err != nil {
		return nil, oops.
			In("resource_edit_repository").
			Tags("fetch_constraints_for_table").
			With("query", keyConstraintsQuery).
			Wrapf(err, "close key constraint rows")
	}

	// * Enhance FKs with foreign table/column details
	fkDetailsQuery := `
    SELECT
        rc.constraint_name,
        ccu.table_name AS foreign_table_name,
        ccu.column_name AS foreign_column_name
    FROM
        information_schema.referential_constraints rc
    JOIN
        information_schema.key_column_usage kcu ON rc.constraint_name = kcu.constraint_name AND rc.constraint_schema = kcu.table_schema
    JOIN
        information_schema.constraint_column_usage ccu ON rc.unique_constraint_name = ccu.constraint_name AND rc.unique_constraint_schema = ccu.table_schema
    WHERE
        kcu.table_schema = ? AND kcu.table_name = ?
    ORDER BY
        rc.constraint_name, kcu.ordinal_position; -- Using kcu.ordinal_position to match local column order
    `
	fkRows, err := dba.QueryContext(ctx, fkDetailsQuery, schemaName, tableName)
	if err != nil {
		r.logger.Error().Err(err).Str("schemaName", schemaName).Str("tableName", tableName).Msg("Querying foreign key details failed")
		return nil, eris.Wrap(err, "querying foreign key details failed")
	}
	defer fkRows.Close()

	tempFkStore := make(map[string]struct {
		FTable string
		FCols  []string
	})
	for fkRows.Next() {
		var consName, fTable, fCol string
		if err = fkRows.Scan(&consName, &fTable, &fCol); err != nil {
			r.logger.Error().Err(err).Msg("Scanning foreign key detail row failed")
			return nil, eris.Wrap(err, "scanning foreign key detail row")
		}
		entry := tempFkStore[consName]
		entry.FTable = fTable // * Will be the same for all columns of a given FK
		entry.FCols = append(entry.FCols, fCol)
		tempFkStore[consName] = entry
	}
	if err = fkRows.Err(); err != nil {
		r.logger.Error().Err(err).Msg("Error iterating foreign key detail rows")
		return nil, eris.Wrap(err, "error iterating foreign key detail rows")
	}
	if err = fkRows.Close(); err != nil {
		return nil, oops.
			In("resource_edit_repository").
			Tags("fetch_constraints_for_table").
			With("query", fkDetailsQuery).
			Wrapf(err, "close foreign key detail rows")
	}

	for consName, fkData := range tempFkStore {
		if constraint, ok := constraintsMap[consName]; ok && constraint.ConstraintType == "FOREIGN KEY" {
			constraint.ForeignTableName = &fkData.FTable
			constraint.ForeignColumnNames = fkData.FCols
		}
	}

	// * Fetch CHECK constraints
	checkConstraintsQuery := `
    SELECT
        tc.constraint_name,
        cc.check_clause,
        tc.is_deferrable,
        tc.initially_deferred
    FROM
        information_schema.check_constraints cc
    JOIN
        information_schema.table_constraints tc ON cc.constraint_name = tc.constraint_name AND cc.constraint_schema = tc.constraint_schema
    WHERE
        tc.table_schema = ? AND tc.table_name = ?;
    `
	checkRows, err := dba.QueryContext(ctx, checkConstraintsQuery, schemaName, tableName)
	if err != nil {
		r.logger.Error().Err(err).Str("schemaName", schemaName).Str("tableName", tableName).Msg("Querying check constraints failed")
		return nil, eris.Wrap(err, "querying check constraints failed")
	}
	defer checkRows.Close()

	for checkRows.Next() {
		var consName, checkClause, isDeferrableStr, initiallyDeferredStr string
		if err = checkRows.Scan(&consName, &checkClause, &isDeferrableStr, &initiallyDeferredStr); err != nil {
			r.logger.Error().Err(err).Msg("Scanning check constraint row failed")
			return nil, eris.Wrap(err, "scanning check constraint row")
		}
		if _, exists := constraintsMap[consName]; !exists {
			constraintsMap[consName] = &repositories.ConstraintDetails{
				ConstraintName:    consName,
				ConstraintType:    resourcesqltype.Check.String(),
				CheckClause:       &checkClause,
				Deferrable:        isDeferrableStr == string(resourcesqltype.KeywordYes),
				InitiallyDeferred: initiallyDeferredStr == string(resourcesqltype.KeywordYes),
			}
			orderedConstraintNames = append(orderedConstraintNames, consName) // Add if it's a new constraint
		} else {
			// * This case should ideally not be hit if CHECK constraints are always in table_constraints
			// * but if it is, update the existing entry.
			constraintsMap[consName].CheckClause = &checkClause
			constraintsMap[consName].ConstraintType = resourcesqltype.Check.String() // Ensure type is correct
		}
	}
	if err = checkRows.Err(); err != nil {
		r.logger.Error().Err(err).Msg("Error iterating check constraint rows")
		return nil, eris.Wrap(err, "error iterating check constraint rows")
	}
	if err = checkRows.Close(); err != nil {
		return nil, oops.
			In("resource_edit_repository").
			Tags("fetch_constraints_for_table").
			With("query", checkConstraintsQuery).
			Wrapf(err, "close check constraint rows")
	}

	var finalConstraints []repositories.ConstraintDetails
	for _, name := range orderedConstraintNames {
		if c, ok := constraintsMap[name]; ok {
			finalConstraints = append(finalConstraints, *c)
		}
	}
	return finalConstraints, nil
}

// handleDotNotation processes dot notation completions (e.g. "table.") and adds relevant suggestions
//
// Parameters:
//   - ctx: The context for the database operation.
//   - req: The autocomplete request containing the current query and prefix.
//   - aliasMap: A map of table aliases to their corresponding table information.
//   - response: The autocomplete response to which suggestions will be added.
//   - columnHighScore: The score to use for column suggestions.
func (r *resourceEditorRepository) handleDotNotation(ctx context.Context, req repositories.AutocompleteRequest, aliasMap map[string]tableAlias, response *repositories.AutocompleteResponse, columnHighScore int) {
	dotIdx := strings.LastIndex(req.Prefix, ".")
	if dotIdx == -1 {
		return
	}

	aliasCandidate := req.Prefix[:dotIdx]
	columnPrefix := req.Prefix[dotIdx+1:]

	tbl, ok := aliasMap[aliasCandidate]
	if !ok {
		return
	}

	schemaName := tbl.Schema
	if schemaName == "" {
		schemaName = req.SchemaName
	}

	cols, err := r.fetchColumnsForTable(ctx, schemaName, tbl.Table)
	if err != nil {
		r.logger.Warn().Err(err).Str("table", tbl.Table).Msg("Failed to fetch columns for alias completion")
		return
	}

	for _, col := range cols {
		if columnPrefix == "" || strings.HasPrefix(strings.ToLower(col.ColumnName), strings.ToLower(columnPrefix)) {
			response.Suggestions = append(response.Suggestions, repositories.AutocompleteSuggestion{
				Value:   col.ColumnName,
				Caption: col.ColumnName + " (" + col.DataType + ")",
				Meta:    "column",
				Score:   columnHighScore,
			})
		}
	}
}

// GetAutocompleteSuggestions generates autocomplete suggestions based on the current query and prefix.
//
// Parameters:
//   - ctx: The context for the database operation.
//   - req: The autocomplete request containing the current query and prefix.
//
// Returns:
//   - *repositories.AutocompleteResponse: The autocomplete response containing the suggestions.
//   - error: An error if the operation fails.
func (r *resourceEditorRepository) GetAutocompleteSuggestions(ctx context.Context, req repositories.AutocompleteRequest) (*repositories.AutocompleteResponse, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to get database connection for autocomplete")
		return nil, eris.Wrap(err, "failed to get database connection for autocomplete")
	}

	response := &repositories.AutocompleteResponse{
		Suggestions: []repositories.AutocompleteSuggestion{},
	}

	// * ------------------------------------------------------------------------------------------------
	// * 1. Parse query context – table aliases & last significant keyword
	// * ------------------------------------------------------------------------------------------------

	aliasMap := parseTableAliases(req.CurrentQuery)

	// * Determine the portion of the query that appears *before* the current prefix being typed.
	// * We can't rely on the prefix being at the very end of CurrentQuery, so we look for the last
	// * occurrence (case-insensitive) of the prefix and slice everything before that index.
	lowerQuery := strings.ToLower(req.CurrentQuery)
	lowerPrefix := strings.ToLower(req.Prefix)
	cutPos := strings.LastIndex(lowerQuery, lowerPrefix)
	var trimmedWithoutPrefix string
	if cutPos != -1 {
		trimmedWithoutPrefix = req.CurrentQuery[:cutPos]
	} else {
		trimmedWithoutPrefix = req.CurrentQuery
	}

	// * Heuristic to decide if the user is editing the SELECT list (columns).
	// * 1. If the last non-space char before the cursor is a comma, we assume they are adding another column.
	// * 2. Otherwise, if SELECT appears before the cursor and the first FROM after that SELECT is located *after* the cursor (or absent), we are still in the list.

	inSelectList := false

	// * Rule 1 – check for trailing comma before cursor.
	if cutPos > 0 {
		i := cutPos - 1
		for i >= 0 && unicode.IsSpace(rune(req.CurrentQuery[i])) {
			i--
		}
		if i >= 0 && req.CurrentQuery[i] == ',' {
			inSelectList = true
		}
	}

	// * Rule 2 – fallback heuristic.
	if !inSelectList {
		upperQuery := strings.ToUpper(req.CurrentQuery)
		selectIdx := strings.Index(upperQuery, resourcesqltype.Select.String())
		if selectIdx != -1 {
			// * Position of first FROM after SELECT (if any)
			fromIdxRel := strings.Index(upperQuery[selectIdx+6:], resourcesqltype.From.String())
			var fromIdxAbs int
			if fromIdxRel != -1 {
				fromIdxAbs = selectIdx + 6 + fromIdxRel
			} else {
				fromIdxAbs = -1
			}

			// * Find the last comma after SELECT (could be none)
			commaIdxRel := strings.LastIndex(upperQuery[selectIdx+6:], ",")
			if commaIdxRel != -1 {
				commaIdxAbs := selectIdx + 6 + commaIdxRel

				// * If comma occurs and either there is no FROM yet OR comma is before FROM, we likely are editing select list
				if fromIdxAbs == -1 || commaIdxAbs < fromIdxAbs {
					inSelectList = true
				}
			} else if fromIdxAbs == -1 {
				// * No comma but also no FROM -> still editing first column(s)
				inSelectList = true
			}
		}
	}

	tokens := strings.Fields(trimmedWithoutPrefix)
	lastKeyword := ""
	if len(tokens) > 0 {
	OUTER:
		// * Walk backwards until we find something that looks like a keyword
		for i := len(tokens) - 1; i >= 0; i-- {
			tokUpper := strings.ToUpper(tokens[i])
			switch tokUpper {
			case resourcesqltype.Select.String(),
				resourcesqltype.From.String(),
				resourcesqltype.Join.String(),
				resourcesqltype.Where.String(),
				resourcesqltype.On.String(),
				resourcesqltype.GroupBy.String(),
				resourcesqltype.OrderBy.String(),
				resourcesqltype.Update.String(),
				resourcesqltype.InsertInto.String(),
				resourcesqltype.Into.String(),
				resourcesqltype.DeleteFrom.String(),
				resourcesqltype.Set.String():
				lastKeyword = tokUpper
				break OUTER
			default:
				// continue scanning
			}
		}
	}

	// * Base scores that can shift depending on context
	columnHighScore := 115
	tableHighScore := 120
	if inSelectList {
		// * Prioritise columns over tables in select list
		columnHighScore = 130
		tableHighScore = 70
	}

	// * ------------------------------------------------------------------------------------------------
	// * 2. Dot-notation – alias/table column suggestions (e.g. `u.`)
	// * ------------------------------------------------------------------------------------------------
	r.handleDotNotation(ctx, req, aliasMap, response, columnHighScore)

	// * ------------------------------------------------------------------------------------------------
	// * 3. Keyword suggestions (generic)
	// * ------------------------------------------------------------------------------------------------
	for _, kw := range resourcesqltype.AvailableKeywords {
		if strings.HasPrefix(strings.ToUpper(kw.String()), strings.ToUpper(req.Prefix)) || req.Prefix == "" {
			response.Suggestions = append(response.Suggestions, repositories.AutocompleteSuggestion{
				Value:   kw.String(),
				Caption: kw.String(),
				Meta:    "keyword",
				Score:   40, // * sslightly lower than before to prioritise context-aware results
			})
		}
	}

	// * ------------------------------------------------------------------------------------------------
	// * 4. Schema suggestions (simple – we currently only expose req.SchemaName)
	// * ------------------------------------------------------------------------------------------------
	if req.SchemaName != "" && (strings.HasPrefix(strings.ToLower(req.SchemaName), strings.ToLower(req.Prefix)) || req.Prefix == "") {
		response.Suggestions = append(response.Suggestions, repositories.AutocompleteSuggestion{
			Value:   req.SchemaName,
			Caption: req.SchemaName,
			Meta:    "schema",
			Score:   90,
		})
	}

	// * ------------------------------------------------------------------------------------------------
	// * 5. Table & column suggestions depending on detected context
	// * ------------------------------------------------------------------------------------------------

	addTableSuggestions := func() {
		if req.SchemaName == "" {
			return
		}
		tableQuery := `SELECT table_name FROM information_schema.tables WHERE table_schema = ? AND (table_name ILIKE ? OR ? = '') ORDER BY table_name;`
		tableRows, qErr := dba.QueryContext(ctx, tableQuery, req.SchemaName, req.Prefix+"%", req.Prefix)
		if qErr != nil {
			r.logger.Error().Err(qErr).Str("schema", req.SchemaName).Msg("Failed to query tables for autocomplete")
			return
		}
		defer tableRows.Close()
		for tableRows.Next() {
			var tableName string
			if scanErr := tableRows.Scan(&tableName); scanErr == nil {
				score := tableHighScore
				if !inSelectList && (lastKeyword == resourcesqltype.From.String() || lastKeyword == resourcesqltype.Join.String() || lastKeyword == resourcesqltype.Update.String() || lastKeyword == resourcesqltype.Into.String()) {
					score = 120 // * raise score when context expects a table name
				}
				response.Suggestions = append(response.Suggestions, repositories.AutocompleteSuggestion{
					Value:   tableName,
					Caption: tableName,
					Meta:    "table",
					Score:   score,
				})
			}
		}
		if err = tableRows.Err(); err != nil {
			r.logger.Error().Err(err).Msg("Error iterating table rows")
			return
		}
	}

	addColumnSuggestions := func() {
		// * Prefer explicit table context, fall back to alias map, finally req.TableName
		columnAdded := make(map[string]struct{})

		// * Helper to add column if not seen and matches prefix
		addColumn := func(columnName, dataType string, score int) {
			if _, ok := columnAdded[columnName]; ok {
				return
			}
			if req.Prefix != "" && !strings.HasPrefix(strings.ToLower(columnName), strings.ToLower(req.Prefix)) {
				return
			}
			response.Suggestions = append(response.Suggestions, repositories.AutocompleteSuggestion{
				Value:   columnName,
				Caption: columnName + " (" + dataType + ")",
				Meta:    "column",
				Score:   score,
			})
			columnAdded[columnName] = struct{}{}
		}

		// * Columns from alias map tables
		for _, tbl := range aliasMap {
			schema := tbl.Schema
			if schema == "" {
				schema = req.SchemaName
			}
			cols, cErr := r.fetchColumnsForTable(ctx, schema, tbl.Table)
			if cErr != nil {
				continue
			}
			for _, col := range cols {
				addColumn(col.ColumnName, col.DataType, columnHighScore)
			}
		}

		// * Fallback: columns from explicit TableName in request
		if req.TableName != "" {
			cols, cErr := r.fetchColumnsForTable(ctx, req.SchemaName, req.TableName)
			if cErr == nil {
				for _, col := range cols {
					addColumn(col.ColumnName, col.DataType, columnHighScore)
				}
			}
		}
	}

	switch lastKeyword {
	case resourcesqltype.From.String(),
		resourcesqltype.Join.String(),
		resourcesqltype.Update.String(),
		resourcesqltype.Into.String():
		if inSelectList {
			addColumnSuggestions()
			addTableSuggestions()
		} else {
			addTableSuggestions()
			addColumnSuggestions()
		}
	case resourcesqltype.Select.String(),
		resourcesqltype.Where.String(),
		resourcesqltype.On.String(),
		resourcesqltype.GroupBy.String(),
		resourcesqltype.OrderBy.String(),
		resourcesqltype.Set.String(),
		resourcesqltype.And.String(),
		resourcesqltype.Or.String():
		addColumnSuggestions()
	default:
		if inSelectList {
			addColumnSuggestions()
			addTableSuggestions()
		} else {
			addTableSuggestions()
			addColumnSuggestions()
		}
	}

	// * ------------------------------------------------------------------------------------------------
	// * 6. Final sorting & deduplication
	// * ------------------------------------------------------------------------------------------------

	sort.SliceStable(response.Suggestions, func(i, j int) bool {
		if response.Suggestions[i].Score != response.Suggestions[j].Score {
			return response.Suggestions[i].Score > response.Suggestions[j].Score
		}
		return response.Suggestions[i].Value < response.Suggestions[j].Value
	})

	seen := make(map[string]repositories.AutocompleteSuggestion)
	finalSuggestions := make([]repositories.AutocompleteSuggestion, 0, len(response.Suggestions))
	for _, s := range response.Suggestions {
		if existing, ok := seen[s.Value]; !ok || s.Score > existing.Score {
			seen[s.Value] = s
		}
	}
	for _, s := range response.Suggestions {
		if stored, ok := seen[s.Value]; ok && stored.Caption == s.Caption {
			finalSuggestions = append(finalSuggestions, s)
			delete(seen, s.Value)
		}
	}
	response.Suggestions = finalSuggestions

	r.logger.Info().Int("suggestion_count", len(response.Suggestions)).Msg("Autocomplete suggestions provided")
	return response, nil
}

// ExecuteSQLQuery executes a SQL query and returns the result.
//
// Parameters:
//   - ctx: The context for the database operation.
//   - req: The execute query request containing the query to execute.
//
// Returns:
//   - *repositories.ExecuteQueryResponse: The execute query response containing the result.
//   - error: An error if the operation fails.
func (r *resourceEditorRepository) ExecuteSQLQuery(ctx context.Context, req repositories.ExecuteQueryRequest) (*repositories.ExecuteQueryResponse, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to get database connection for query execution")
		return nil, eris.Wrap(err, "failed to get database connection for query execution")
	}

	log := r.logger.With().
		Str("operation", "ExecuteSQLQuery").
		Str("query", req.Query).
		Logger()

	response := &repositories.ExecuteQueryResponse{}

	log.Info().Msg("Executing user SQL query")

	var resultsData [][]any

	// ! Some read-only operations might not work as expected in a transaction (e.g. some SHOW commands on some DBs)
	err = dba.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		rows, rErr := tx.QueryContext(ctx, req.Query)
		if rErr != nil {
			return rErr
		}
		defer rows.Close()

		cols, cErr := rows.Columns()
		if cErr != nil {
			log.Error().Err(cErr).Msg("Error getting columns from query result")
			response.Result.Error = "Failed to get columns from result: " + cErr.Error()
			return cErr
		}
		response.Result.Columns = cols

		for rows.Next() {
			// * Create a slice of anys to represent a row, a bit like python's tuple
			rowValues := make([]any, len(cols))
			// * Create a slice of pointers to the above slice's elements for Scan
			rowPointers := make([]any, len(cols))
			for i := range rowValues {
				rowPointers[i] = &rowValues[i]
			}

			if err = rows.Scan(rowPointers...); err != nil {
				log.Error().Err(err).Msg("Error scanning row from query result")
				response.Result.Error = "Failed to scan row: " + err.Error()
				return err
			}
			resultsData = append(resultsData, rowValues)
		}

		if err = rows.Err(); err != nil {
			log.Error().Err(err).Msg("Error iterating query result rows")
			response.Result.Error = "Error iterating result rows: " + err.Error()
			return err
		}

		response.Result.Rows = resultsData
		switch {
		case len(resultsData) == 0 && len(cols) > 0:
			response.Result.Message = "Query executed successfully, 0 rows returned."
		case len(resultsData) > 0:
			response.Result.Message = fmt.Sprintf("Query executed successfully, %d rows returned.", len(resultsData))
		default:
			// * This case might be for non-SELECT statements if QueryContext was used and didn't error.
			// * However, dba.ExecContext should be used for non-SELECTs.
			// * For now, assume QueryContext might return no cols/rows for successful non-SELECTs that don't return rows.
			// * A more robust solution would involve trying ExecContext if QueryContext fails or based on query type detection.
			log.Warn().Msg("Query executed, but returned no columns. Possibly a non-SELECT statement or empty result.")

			res, execErr := dba.ExecContext(ctx, req.Query)
			if execErr != nil {
				log.Error().Err(execErr).Msg("Error executing SQL query with ExecContext after QueryContext yielded no columns")
				response.Result.Error = execErr.Error()
				return execErr
			}

			rowsAffected, _ := res.RowsAffected()
			response.Result.Message = fmt.Sprintf("Command executed successfully. Rows affected: %d", rowsAffected)
		}

		return nil
	})

	log.Info().Int("rowsReturned", len(response.Result.Rows)).Msg("SQL query executed successfully")
	return response, nil
}
