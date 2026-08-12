"""Per-dialect translation profiles.

Adding a dialect means adding one entry here, not a new converter. Each profile
describes how PostgreSQL DDL maps onto the target: type names, which statement
kinds are unsupported, and which expressions survive translation.
"""

SQLITE = {
    "name": "sqlite",
    "display_name": "SQLite",
    "identifier_quote": '"',
    # Postgres base type -> target type.
    "types": {
        "bigint": "INTEGER", "int8": "INTEGER", "integer": "INTEGER", "int": "INTEGER",
        "int4": "INTEGER", "smallint": "INTEGER", "int2": "INTEGER", "bigserial": "INTEGER",
        "serial": "INTEGER", "smallserial": "INTEGER", "boolean": "INTEGER", "bool": "INTEGER",
        "real": "REAL", "float4": "REAL", "float8": "REAL", "double precision": "REAL",
        # REAL rather than NUMERIC: NUMERIC affinity demotes integral values to
        # INTEGER, which then refuses to scan into a Go float64. Precision is the
        # same either way, and SQLite has no exact decimal regardless.
        "numeric": "REAL", "decimal": "REAL", "money": "REAL",
        "text": "TEXT", "citext": "TEXT", "varchar": "TEXT", "character varying": "TEXT",
        "char": "TEXT", "character": "TEXT", "uuid": "TEXT", "json": "TEXT", "jsonb": "TEXT",
        "xml": "TEXT", "inet": "TEXT", "cidr": "TEXT", "macaddr": "TEXT", "interval": "TEXT",
        "date": "TEXT", "time": "TEXT", "timetz": "TEXT",
        "time with time zone": "TEXT", "time without time zone": "TEXT",
        "timestamp": "TEXT", "timestamptz": "TEXT",
        "timestamp with time zone": "TEXT", "timestamp without time zone": "TEXT",
        "int4range": "TEXT", "int8range": "TEXT", "numrange": "TEXT", "tsrange": "TEXT",
        "tstzrange": "TEXT", "daterange": "TEXT", "regconfig": "TEXT",
        "bytea": "BLOB",
    },
    # Column types with no target equivalent: the column is dropped.
    "drop_column_types": {
        "tsvector", "tsquery", "geography", "geometry", "box2d", "box3d",
        "vector", "ltree", "hstore",
    },
    "array_type": "TEXT",
    "enum_type": "TEXT",
    "unknown_type": "TEXT",
    # Functions safe to keep inside CHECK constraints, index expressions and defaults.
    "portable_functions": {
        "lower", "upper", "length", "abs", "coalesce", "nullif", "trim", "ltrim", "rtrim",
        "replace", "substr", "round", "min", "max", "unixepoch", "ifnull", "instr", "cast",
    },
    "function_renames": {
        "char_length": "length", "character_length": "length",
        "strpos": "instr", "substring": "substr",
    },
    "now_expression": "CURRENT_TIMESTAMP",
    "epoch_expression": "(unixepoch())",
    "true_literal": "1",
    "false_literal": "0",
    "index_methods": {"", "btree"},
    "supports_add_constraint": False,
    "supports_alter_column": False,
    "supports_drop_column": True,
    "supports_stored_generated_in_alter": False,
    "materialized_view_as_view": True,
}

# Sketch for a future dialect. Filling this in is the whole cost of adding one.
MSSQL = {
    "name": "mssql",
    "display_name": "Microsoft SQL Server",
    "identifier_quote": "[",
    "types": {
        "bigint": "BIGINT", "int8": "BIGINT", "integer": "INT", "int": "INT", "int4": "INT",
        "smallint": "SMALLINT", "int2": "SMALLINT", "boolean": "BIT", "bool": "BIT",
        "real": "REAL", "float8": "FLOAT", "double precision": "FLOAT",
        "numeric": "DECIMAL", "decimal": "DECIMAL", "money": "MONEY",
        "text": "NVARCHAR(MAX)", "varchar": "NVARCHAR", "character varying": "NVARCHAR",
        "char": "NCHAR", "character": "NCHAR", "uuid": "UNIQUEIDENTIFIER",
        "json": "NVARCHAR(MAX)", "jsonb": "NVARCHAR(MAX)",
        "timestamp": "DATETIME2", "timestamptz": "DATETIMEOFFSET",
        "date": "DATE", "time": "TIME", "bytea": "VARBINARY(MAX)",
    },
    "drop_column_types": {"tsvector", "tsquery", "geography", "geometry", "vector", "ltree", "hstore"},
    "array_type": "NVARCHAR(MAX)",
    "enum_type": "NVARCHAR(64)",
    "unknown_type": "NVARCHAR(MAX)",
    "portable_functions": {
        "lower", "upper", "len", "abs", "coalesce", "nullif", "trim", "ltrim", "rtrim",
        "replace", "substring", "round", "min", "max", "cast",
    },
    "function_renames": {"length": "len", "char_length": "len", "strpos": "charindex"},
    "now_expression": "SYSUTCDATETIME()",
    "epoch_expression": "DATEDIFF_BIG(SECOND, '1970-01-01', SYSUTCDATETIME())",
    "true_literal": "1",
    "false_literal": "0",
    "index_methods": {"", "btree"},
    "supports_add_constraint": True,
    "supports_alter_column": True,
    "supports_drop_column": True,
    "supports_stored_generated_in_alter": False,
    "materialized_view_as_view": True,
}

PROFILES = {"sqlite": SQLITE, "mssql": MSSQL}
