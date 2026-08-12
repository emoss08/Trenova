#!/usr/bin/env python3
"""Translate the PostgreSQL migration corpus into another dialect.

PostgreSQL is the reference schema for Trenova. This script produces a
best-effort parallel migration set for contributors who want to run the stack on
another database locally. Output is committed and may be hand-edited afterwards;
re-running reports what it could not translate so hand-edits stay visible.

    python3 scripts/dialect-convert/convert.py sqlite
    python3 scripts/dialect-convert/convert.py sqlite --check
"""

from __future__ import annotations

import argparse
import collections
import os
import re
import sys

from profiles import PROFILES

SPLIT_MARKER = "--bun:split"

HEADER = (
    '-- Code generated from the PostgreSQL migrations by\n'
    '-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you\n'
    '-- stop regenerating this file; see docs/databases.md.\n'
    '-- Source: {source}\n'
)


# --------------------------------------------------------------------------
# Lexing: Postgres strings, dollar-quoted bodies and comments all hide `;`.
# --------------------------------------------------------------------------

def strip_comments(sql: str) -> str:
    out, i, n = [], 0, len(sql)
    while i < n:
        ch = sql[i]
        if ch == "-" and sql.startswith("--", i):
            j = sql.find("\n", i)
            i = n if j < 0 else j
            continue
        if ch == "/" and sql.startswith("/*", i):
            j = sql.find("*/", i)
            i = n if j < 0 else j + 2
            continue
        if ch == "'":
            j = i + 1
            while j < n:
                if sql[j] == "'":
                    j += 1
                    break
                j += 1
            out.append(sql[i:j])
            i = j
            continue
        if ch == '"':
            j = sql.find('"', i + 1)
            j = n if j < 0 else j + 1
            out.append(sql[i:j])
            i = j
            continue
        tag = _dollar_tag(sql, i)
        if tag:
            end = sql.find(tag, i + len(tag))
            end = n if end < 0 else end + len(tag)
            out.append(sql[i:end])
            i = end
            continue
        out.append(ch)
        i += 1
    return "\n".join(l.rstrip() for l in "".join(out).splitlines() if l.strip())


def _dollar_tag(sql: str, i: int) -> str | None:
    if sql[i] != "$":
        return None
    m = re.match(r"\$[A-Za-z_0-9]*\$", sql[i:])
    return m.group(0) if m else None


def split_statements(sql: str) -> list[str]:
    """Split into individual statements, honouring --bun:split and top-level `;`."""
    statements = []
    for chunk in sql.split(SPLIT_MARKER):
        chunk = strip_comments(chunk).strip()
        if not chunk:
            continue
        statements.extend(_split_semicolons(chunk))
    return [s for s in statements if s.strip()]


def _split_semicolons(sql: str) -> list[str]:
    out, buf, depth, i, n = [], [], 0, 0, len(sql)
    while i < n:
        ch = sql[i]
        if ch == "'":
            j = i + 1
            while j < n:
                if sql[j] == "'":
                    j += 1
                    break
                j += 1
            buf.append(sql[i:j])
            i = j
            continue
        if ch == '"':
            j = sql.find('"', i + 1)
            j = n if j < 0 else j + 1
            buf.append(sql[i:j])
            i = j
            continue
        tag = _dollar_tag(sql, i)
        if tag:
            end = sql.find(tag, i + len(tag))
            end = n if end < 0 else end + len(tag)
            buf.append(sql[i:end])
            i = end
            continue
        if ch in "([":
            depth += 1
        elif ch in ")]":
            depth -= 1
        elif ch == ";" and depth <= 0:
            buf.append(";")
            out.append("".join(buf).strip())
            buf = []
            i += 1
            continue
        buf.append(ch)
        i += 1
    if "".join(buf).strip():
        out.append("".join(buf).strip())
    return out


def split_top_level(body: str, sep: str = ",") -> list[str]:
    parts, buf, depth, i, n = [], [], 0, 0, len(body)
    while i < n:
        ch = body[i]
        if ch == "'":
            j = i + 1
            while j < n:
                if body[j] == "'":
                    j += 1
                    break
                j += 1
            buf.append(body[i:j])
            i = j
            continue
        if ch == '"':
            j = body.find('"', i + 1)
            j = n if j < 0 else j + 1
            buf.append(body[i:j])
            i = j
            continue
        if ch in "([":
            depth += 1
        elif ch in ")]":
            depth -= 1
        elif ch == sep and depth == 0:
            parts.append("".join(buf).strip())
            buf = []
            i += 1
            continue
        buf.append(ch)
        i += 1
    if "".join(buf).strip():
        parts.append("".join(buf).strip())
    return parts


def matching_paren(sql: str, open_idx: int) -> int:
    depth, i, n = 0, open_idx, len(sql)
    while i < n:
        ch = sql[i]
        if ch == "'":
            i += 1
            while i < n and sql[i] != "'":
                i += 1
        elif ch == '"':
            i = sql.find('"', i + 1)
            if i < 0:
                return -1
        elif ch == "(":
            depth += 1
        elif ch == ")":
            depth -= 1
            if depth == 0:
                return i
        i += 1
    return -1


# --------------------------------------------------------------------------
# Patterns
# --------------------------------------------------------------------------

RE = {
    "create_table": re.compile(r"(?is)^CREATE\s+(?:UNLOGGED\s+)?TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(\"?[\w.]+\"?)\s*\("),
    "create_enum": re.compile(r"(?is)^CREATE\s+TYPE\s+(\"?[\w.]+\"?)\s+AS\s+ENUM\s*\("),
    "create_domain": re.compile(r"(?is)^CREATE\s+DOMAIN\s+(\"?[\w.]+\"?)\s+AS\s+([\w\s]+?(?:\(\s*\d+(?:\s*,\s*\d+)?\s*\))?)\s*(?:CONSTRAINT|CHECK|NOT\s+NULL|DEFAULT|;|$)"),
    "enum_add": re.compile(r"(?is)^ALTER\s+TYPE\s+(\"?[\w.]+\"?)\s+ADD\s+VALUE\s+(?:IF\s+NOT\s+EXISTS\s+)?('[^']*')"),
    "create_index": re.compile(r"(?is)^CREATE\s+(UNIQUE\s+)?INDEX\s+(?:CONCURRENTLY\s+)?(?:IF\s+NOT\s+EXISTS\s+)?(\"?[\w.]+\"?)\s+ON\s+(?:ONLY\s+)?(\"?[\w.]+\"?)\s*(?:USING\s+(\w+)\s*)?\("),
    "matview": re.compile(r"(?is)^CREATE\s+MATERIALIZED\s+VIEW\s+(?:IF\s+NOT\s+EXISTS\s+)?"),
    "view_name": re.compile(r"(?is)^CREATE\s+(?:OR\s+REPLACE\s+)?(?:MATERIALIZED\s+)?VIEW\s+(?:IF\s+NOT\s+EXISTS\s+)?(\"?[\w.]+\"?)"),
    "alter_table": re.compile(r"(?is)^ALTER\s+TABLE\s+(?:IF\s+EXISTS\s+)?(?:ONLY\s+)?(\"?[\w.]+\"?)\s+(.*)$"),
    "add_column": re.compile(r"(?is)^ADD\s+COLUMN\s+(?:IF\s+NOT\s+EXISTS\s+)?(.*)$"),
    "drop_column": re.compile(r"(?is)^DROP\s+COLUMN\s+(?:IF\s+EXISTS\s+)?(\"([^\"]+)\"|[A-Za-z_]\w*)"),
    "column_def": re.compile(r"(?s)^(\"([^\"]+)\"|[A-Za-z_]\w*)\s+(.*)$"),
    "do_block": re.compile(r"(?is)^DO\s*[$\n]"),
    "cast": re.compile(r"::\s*\"?[a-zA-Z_]\w*\"?(?:\s*\(\s*\d+(?:\s*,\s*\d+)?\s*\))?(?:\s*\[\s*\])?"),
    "epoch": re.compile(r"(?i)extract\s*\(\s*epoch\s+from\s+(?:current_timestamp|now\s*\(\s*\))\s*\)"),
    "identity": re.compile(r"(?i)\s+GENERATED\s+(?:ALWAYS|BY\s+DEFAULT)\s+AS\s+IDENTITY(?:\s*\([^)]*\))?"),
    "generated": re.compile(r"(?is)\s+GENERATED\s+ALWAYS\s+AS\s*\("),
    "default": re.compile(r"(?is)\s+DEFAULT\s+"),
    "check": re.compile(r"(?is)\s+CHECK\s*\("),
    "references": re.compile(r"(?is)\s+REFERENCES\s+"),
    "collate": re.compile(r"(?i)\s+COLLATE\s+\"?[\w.]+\"?"),
    "array_suffix": re.compile(r"\s*\[\s*\d*\s*\]"),
    "partition": re.compile(r"(?is)\s+PARTITION\s+BY\s+\w+\s*\([^)]*\)\s*$"),
    "opclass": re.compile(r"(?i)\s+(?:text_pattern_ops|varchar_pattern_ops|bpchar_pattern_ops|gin_trgm_ops|gist_trgm_ops|jsonb_path_ops|jsonb_ops|int8_ops|text_ops)"),
    "include": re.compile(r"(?is)\s+INCLUDE\s*\([^)]*\)"),
    "with_opts": re.compile(r"(?is)\s+WITH\s*\([^)]*\)"),
    "nulls_order": re.compile(r"(?i)\s+NULLS\s+(?:FIRST|LAST)"),
    "concurrently": re.compile(r"(?i)\s+CONCURRENTLY"),
    "cascade": re.compile(r"(?i)\s+CASCADE"),
    "not_valid": re.compile(r"(?i)\s+NOT\s+VALID"),
    "func_call": re.compile(r"(?i)\b([a-z_]\w*)\s*\("),
    "quoted": re.compile(r'"([^"]+)"'),
    "aliased_update": re.compile(r"(?is)^UPDATE\s+\"?[\w.]+\"?\s+(?!SET\b)[A-Za-z_]\w*\s"),
    "literal": re.compile(r"(?i)^('([^']|'')*'|-?\d+(\.\d+)?|NULL|TRUE|FALSE)$"),
}

SQL_KEYWORDS = {
    "and", "or", "not", "in", "is", "null", "between", "like", "when", "then",
    "else", "end", "case", "select", "from", "where", "exists", "as", "on",
}

RE_CONSTRAINT_HEAD = re.compile(
    r"(?i)^(CONSTRAINT|PRIMARY\s+KEY|FOREIGN\s+KEY|UNIQUE|CHECK|EXCLUDE|LIKE)\b"
)

COLUMN_KEYWORDS = ("NOT NULL", "NULL", "PRIMARY KEY", "UNIQUE", "CHECK", "REFERENCES", "GENERATED", "COLLATE")

# Statement kinds that simply have no counterpart outside PostgreSQL.
DISCARD_PREFIXES = {
    "CREATE EXTENSION": "extension", "DROP EXTENSION": "extension",
    "CREATE FUNCTION": "plpgsql_function", "CREATE OR REPLACE FUNCTION": "plpgsql_function",
    "DROP FUNCTION": "plpgsql_function", "CREATE PROCEDURE": "plpgsql_function",
    "CREATE OR REPLACE PROCEDURE": "plpgsql_function",
    "CREATE TRIGGER": "trigger", "CREATE OR REPLACE TRIGGER": "trigger", "DROP TRIGGER": "trigger",
    "CREATE EVENT TRIGGER": "trigger", "DROP EVENT TRIGGER": "trigger",
    "CREATE SEQUENCE": "sequence", "DROP SEQUENCE": "sequence",
    "CREATE STATISTICS": "planner_statistics", "DROP STATISTICS": "planner_statistics",
    "CREATE PUBLICATION": "replication", "DROP PUBLICATION": "replication",
    "ALTER PUBLICATION": "replication",
    "CREATE POLICY": "row_level_security", "DROP POLICY": "row_level_security",
    "COMMENT ON": "no_op", "GRANT ": "no_op", "REVOKE ": "no_op", "ANALYZE": "no_op",
    "VACUUM": "no_op", "SET ": "no_op", "RESET ": "no_op", "SELECT": "no_op",
    "REFRESH MATERIALIZED VIEW": "no_op", "ALTER INDEX": "no_op", "ALTER SEQUENCE": "no_op",
    "ALTER VIEW": "no_op", "ALTER MATERIALIZED VIEW": "no_op",
    "CREATE TYPE": "composite_type", "DROP TYPE": "composite_type",
}


class Converter:
    def __init__(self, profile: dict):
        self.p = profile
        self.enums: dict[str, list[str]] = {}
        self.domains: dict[str, str] = {}
        self.table_columns: dict[str, set[str]] = collections.defaultdict(set)
        self.dropped_columns: dict[str, set[str]] = collections.defaultdict(set)
        self.relations: set[str] = set()
        self.pending_statements: list[str] = []
        # SQLite refuses DROP COLUMN when an index or CHECK still references the
        # column, so track both to decide whether a drop can be emitted.
        self.check_columns: dict[str, set[str]] = collections.defaultdict(set)
        self.table_indexes: dict[str, list[tuple[str, set[str]]]] = collections.defaultdict(list)
        self.skipped: list[tuple[str, str, str]] = []
        self.downgrades: list[tuple[str, str, str]] = []

    # -- reporting ---------------------------------------------------------
    def skip(self, f, reason, detail):
        self.skipped.append((f, reason, _first_line(detail)))

    def downgrade(self, f, reason, detail):
        self.downgrades.append((f, reason, _first_line(detail)))

    # -- types -------------------------------------------------------------
    def map_type(self, sql_type: str) -> str | None:
        """Returns the target type, or None when the column must be dropped."""
        base = _normalize_type(sql_type)
        if base in self.p["drop_column_types"]:
            return None
        if RE["array_suffix"].search(sql_type):
            return self.p["array_type"]
        if base in self.enums:
            return self.p["enum_type"]
        if base in self.domains:
            return self.map_type(self.domains[base])
        return self.p["types"].get(base, self.p["unknown_type"])

    # -- expressions -------------------------------------------------------
    def is_portable(self, expr: str) -> bool:
        if not expr.strip():
            return False
        if "[" in expr or "::" in expr or "~" in expr:
            return False
        if "SELECT" in expr.upper() or "SIMILAR TO" in expr.upper():
            return False
        for name in RE["func_call"].findall(expr):
            low = name.lower()
            if low in SQL_KEYWORDS:
                continue
            if low not in self.p["portable_functions"]:
                return False
        return True

    def portable_check(self, expr: str) -> str | None:
        candidate = _rename_functions(RE["cast"].sub("", expr).strip(), self.p)
        return candidate if self.is_portable(candidate) else None

    def portable_default(self, expr: str, is_array: bool = False) -> str | None:
        value = RE["epoch"].sub("__EPOCH__", expr.strip())
        if is_array and value.strip() in ("'{}'", '"{}"'):
            return "'[]'"
        value = RE["cast"].sub("", value).strip()
        low = value.lower()
        if "nextval(" in low or "gen_random_uuid" in low or "uuid_generate" in low:
            return None
        if value == "__EPOCH__":
            return self.p["epoch_expression"]
        if low in ("now()", "current_timestamp", "clock_timestamp()", "statement_timestamp()"):
            return self.p["now_expression"]
        if low == "true":
            return self.p["true_literal"]
        if low == "false":
            return self.p["false_literal"]
        if RE["literal"].match(value):
            return value
        if value.startswith("(") and value.endswith(")") and self.is_portable(value[1:-1]):
            return value
        return None

    # -- statement dispatch ------------------------------------------------
    def convert_file(self, name: str, sql: str) -> str:
        out = []
        for statement in split_statements(sql):
            result = self.convert_statement(name, statement)
            if result and result.strip():
                out.append(result.strip())
        if not out:
            return ""
        return ("\n\n" + SPLIT_MARKER + "\n\n").join(out) + "\n"

    def convert_statement(self, f: str, sql: str) -> str:
        s = sql.strip()
        up = s.upper()

        for prefix, reason in DISCARD_PREFIXES.items():
            if up.startswith(prefix):
                # CREATE TYPE ... AS ENUM is handled below, not discarded.
                if prefix == "CREATE TYPE" and RE["create_enum"].match(s):
                    break
                self.skip(f, reason, s)
                return ""

        if RE["do_block"].match(s):
            self.skip(f, "procedural_block", s)
            return ""
        if RE["create_enum"].match(s):
            self._record_enum(s)
            return ""
        if RE["enum_add"].match(s):
            self._record_enum_value(s)
            return ""
        if up.startswith("ALTER TYPE"):
            self.skip(f, "enum_alteration", s)
            return ""
        if RE["create_domain"].match(s):
            m = RE["create_domain"].match(s)
            self.domains[_unquote(m.group(1)).lower()] = m.group(2).strip()
            return ""
        if up.startswith("DROP DOMAIN"):
            return ""
        if RE["create_table"].match(s):
            return self._create_table(f, s)
        if RE["create_index"].match(s):
            return self._create_index(f, s)
        if RE["matview"].match(s):
            return self._matview(f, s)
        if up.startswith("DROP MATERIALIZED VIEW"):
            return s.replace("MATERIALIZED VIEW", "VIEW", 1)
        if RE["alter_table"].match(s):
            return self._alter_table(f, s)
        if up.startswith("DROP TABLE"):
            return RE["cascade"].sub("", s)
        if up.startswith("DROP INDEX"):
            return RE["concurrently"].sub("", s)
        if up.startswith(("INSERT", "UPDATE", "DELETE", "WITH")):
            if RE["aliased_update"].match(s):
                self.skip(f, "aliased_update_unsupported", s)
                return ""
            if not self.is_portable(s):
                self.skip(f, "postgres_only_expression", s)
                return ""
            return RE["cast"].sub("", s)
        if up.startswith(("CREATE VIEW", "CREATE OR REPLACE VIEW")):
            return self._view(f, s)

        self.skip(f, "unrecognized_statement", s)
        return ""

    # -- enums -------------------------------------------------------------
    def _record_enum(self, s: str):
        m = RE["create_enum"].match(s)
        open_idx = m.end() - 1
        close_idx = matching_paren(s, open_idx)
        if close_idx < 0:
            return
        values = [v.strip() for v in split_top_level(s[open_idx + 1:close_idx]) if v.strip()]
        self.enums[_unquote(m.group(1)).lower()] = values

    def _record_enum_value(self, s: str):
        m = RE["enum_add"].match(s)
        key = _unquote(m.group(1)).lower()
        self.enums.setdefault(key, [])
        if m.group(2) not in self.enums[key]:
            self.enums[key].append(m.group(2))

    # -- CREATE TABLE ------------------------------------------------------
    def _create_table(self, f: str, s: str) -> str:
        m = RE["create_table"].match(s)
        table = _unquote(m.group(1))
        open_idx = m.end() - 1
        close_idx = matching_paren(s, open_idx)
        if close_idx < 0:
            self.skip(f, "unrecognized_statement", s)
            return ""

        header, body, tail = s[:open_idx], s[open_idx + 1:close_idx], s[close_idx + 1:]

        if RE["partition"].search(tail):
            self.downgrade(f, "partitioning_removed", s)
            tail = RE["partition"].sub("", tail)

        items = []
        for item in split_top_level(body):
            converted = self._table_item(f, table, item)
            if converted:
                items.append("    " + converted)

        if not items:
            self.skip(f, "table_had_no_portable_columns", s)
            return ""

        self.relations.add(table.lower())
        header = header.replace("UNLOGGED ", "")
        if "IF NOT EXISTS" not in header.upper():
            header = re.sub(r"(?i)CREATE\s+TABLE\s+", "CREATE TABLE IF NOT EXISTS ", header)

        tail = tail.strip() or ";"
        return f"{header}(\n" + ",\n".join(items) + f"\n){tail}"

    def _table_item(self, f: str, table: str, item: str) -> str:
        item = item.strip()
        if not item:
            return ""
        if RE_CONSTRAINT_HEAD.match(item):
            return self._table_constraint(f, table, item)
        return self._column(f, table, item)

    def _pin(self, table: str, expr: str) -> None:
        """Record columns a surviving constraint references.

        SQLite refuses DROP COLUMN while any index or constraint mentions the
        column, so every emitted constraint pins whatever it names."""
        self.check_columns[table.lower()].update(_identifiers(expr))

    def _table_constraint(self, f: str, table: str, c: str) -> str:
        up = c.upper()
        if "EXCLUDE USING" in up:
            self.skip(f, "exclusion_constraint", c)
            return ""
        if up.startswith("LIKE "):
            self.skip(f, "unrecognized_statement", c)
            return ""
        for col in RE["quoted"].findall(c):
            if col.lower() in self.dropped_columns[table.lower()]:
                self.skip(f, "references_dropped_column", c)
                return ""

        out = _rename_functions(RE["not_valid"].sub("", RE["cast"].sub("", c)), self.p)
        out = re.sub(r"(?i)\s+USING\s+INDEX\s+TABLESPACE\s+\w+", "", out)
        out = re.sub(
            r"(?i)(ON\s+(?:DELETE|UPDATE)\s+SET\s+(?:NULL|DEFAULT))\s*\([^)]*\)", r"\1", out
        )

        cm = RE["check"].search(out)
        if cm:
            open_idx = cm.end() - 1
            close_idx = matching_paren(out, open_idx)
            if close_idx < 0:
                self.skip(f, "non_portable_check", c)
                return ""
            portable = self.portable_check(out[open_idx + 1:close_idx])
            if portable is None:
                self.skip(f, "non_portable_check", c)
                return ""
            self._pin(table, portable)
            out = out[:open_idx + 1] + portable + out[close_idx:]

        self._pin(table, out)
        return out

    def _column(self, f: str, table: str, col: str) -> str:
        m = RE["column_def"].match(col)
        if not m:
            self.skip(f, "unrecognized_statement", col)
            return ""
        name = m.group(2) or m.group(1)
        rest = m.group(3).strip()

        if RE["generated"].search(rest):
            return self._generated_column(f, table, name, rest)

        sql_type, modifiers = _split_type(rest)
        mapped = self.map_type(sql_type)
        if mapped is None:
            self.dropped_columns[table.lower()].add(name.lower())
            self.skip(f, "unsupported_column_type", f"{table}.{name} {sql_type}")
            return ""

        self.table_columns[table.lower()].add(name.lower())
        suffix = self._modifiers(
            f, table, name, modifiers, is_array=bool(RE["array_suffix"].search(sql_type))
        )
        return f'"{name}" {mapped}' + (f" {suffix}" if suffix else "")

    def _generated_column(self, f: str, table: str, name: str, rest: str) -> str:
        gm = RE["generated"].search(rest)
        open_idx = gm.end() - 1
        close_idx = matching_paren(rest, open_idx)
        if close_idx < 0:
            self.dropped_columns[table.lower()].add(name.lower())
            return ""
        expr = rest[open_idx + 1:close_idx]
        portable = self.portable_check(expr)
        if portable is None:
            self.dropped_columns[table.lower()].add(name.lower())
            self.skip(f, "generated_column_expression", f"{table}.{name}")
            return ""
        mapped = self.map_type(rest[:gm.start()].strip())
        if mapped is None:
            self.dropped_columns[table.lower()].add(name.lower())
            self.skip(f, "unsupported_column_type", f"{table}.{name}")
            return ""
        stored = "STORED" if "STORED" in rest[close_idx:].upper() else "VIRTUAL"
        self.table_columns[table.lower()].add(name.lower())
        return f'"{name}" {mapped} GENERATED ALWAYS AS ({portable}) {stored}'

    def _modifiers(
        self, f: str, table: str, column: str, mods: str, is_array: bool = False
    ) -> str:
        value = RE["collate"].sub("", RE["identity"].sub("", mods.strip()))
        value = RE["not_valid"].sub("", value)
        parts: list[str] = []

        dm = RE["default"].search(value)
        if dm:
            expr, remainder = _split_default(value[dm.end():])
            value = (value[:dm.start()] + " " + remainder).strip()
            portable = self.portable_default(expr, is_array=is_array)
            if portable is not None:
                parts.append("DEFAULT " + portable)
            else:
                self.skip(f, "non_portable_default", f"{table}.{column} = {expr}")

        cm = RE["check"].search(value)
        if cm:
            open_idx = cm.end() - 1
            close_idx = matching_paren(value, open_idx)
            if close_idx >= 0:
                portable = self.portable_check(value[open_idx + 1:close_idx])
                if portable is not None:
                    parts.append(f"CHECK ({portable})")
                    self._pin(table, portable)
                    self._pin(table, column)
                else:
                    self.skip(f, "non_portable_check", f"{table}.{column}")
                value = (value[:cm.start()] + value[close_idx + 1:]).strip()

        rm = RE["references"].search(value)
        if rm:
            parts.append(RE["cast"].sub("", value[rm.start():].strip()))
            value = value[:rm.start()].strip()

        up = value.upper()
        leading = []
        if "PRIMARY KEY" in up:
            leading.append("PRIMARY KEY")
        if "NOT NULL" in up:
            leading.append("NOT NULL")
        if "UNIQUE" in up:
            leading.append("UNIQUE")
        return " ".join(leading + parts).strip()

    # -- ALTER TABLE -------------------------------------------------------
    def _alter_table(self, f: str, s: str) -> str:
        m = RE["alter_table"].match(s)
        table = _unquote(m.group(1))
        actions = m.group(2).strip().rstrip(";")

        emitted = []
        for action in split_top_level(actions):
            converted = self._alter_action(f, table, action)
            if self.pending_statements:
                emitted.extend(self.pending_statements)
                self.pending_statements = []
            if converted:
                emitted.append(f'ALTER TABLE "{table}" {converted};')
        return ("\n\n" + SPLIT_MARKER + "\n\n").join(emitted)

    def _alter_action(self, f: str, table: str, action: str) -> str:
        a = action.strip()
        up = a.upper()

        if RE["add_column"].match(a):
            return self._add_column(f, table, a)
        if up.startswith("DROP COLUMN"):
            return self._drop_column(f, table, a)
        if up.startswith(("RENAME COLUMN", "RENAME TO")):
            return a
        if up.startswith(("ADD CONSTRAINT", "ADD PRIMARY KEY", "ADD FOREIGN KEY", "ADD UNIQUE", "ADD CHECK", "ADD EXCLUDE")):
            if not self.p["supports_add_constraint"]:
                self.skip(f, "alter_add_constraint", f"{table}: {a}")
                return ""
            return self._table_constraint(f, table, a[4:])
        if up.startswith("DROP CONSTRAINT"):
            self.skip(f, "alter_drop_constraint", f"{table}: {a}")
            return ""
        if up.startswith("ALTER COLUMN"):
            if not self.p["supports_alter_column"]:
                self.skip(f, "alter_column", f"{table}: {a}")
                return ""
            return a
        self.skip(f, "no_op", f"{table}: {a}")
        return ""

    def _drop_column(self, f: str, table: str, action: str) -> str:
        dm = RE["drop_column"].match(action)
        if not dm:
            self.skip(f, "unrecognized_statement", f"{table}: {action}")
            return ""

        col = (dm.group(2) or dm.group(1)).lower()
        key = table.lower()

        if not self.p["supports_drop_column"]:
            self.skip(f, "drop_column_unsupported", f"{table}.{col}")
            return ""

        # A surviving CHECK pinned to the column makes the drop illegal, and there
        # is no way to alter a CHECK out of the way in SQLite.
        if col not in self.table_columns[key]:
            self.skip(f, "drop_column_never_created", f"{table}.{col}")
            return ""

        if col in self.check_columns[key]:
            self.skip(f, "drop_column_pinned_by_constraint", f"{table}.{col}")
            return ""

        blocking = [name for name, cols in self.table_indexes[key] if col in cols]
        for name in blocking:
            self.pending_statements.append(f'DROP INDEX IF EXISTS "{name}";')
        self.table_indexes[key] = [
            (name, cols) for name, cols in self.table_indexes[key] if col not in cols
        ]

        self.dropped_columns[key].add(col)
        self.table_columns[key].discard(col)

        return f'DROP COLUMN "{col}"'

    def _add_column(self, f: str, table: str, action: str) -> str:
        definition = RE["add_column"].match(action).group(1).strip()
        m = RE["column_def"].match(definition)
        if m:
            name = (m.group(2) or m.group(1)).lower()
            if name in self.table_columns[table.lower()]:
                self.skip(f, "column_already_exists", f"{table}.{name}")
                return ""

        converted = self._column(f, table, definition)
        if not converted:
            return ""
        up = converted.upper()
        if "GENERATED ALWAYS AS" in up and "STORED" in up and not self.p["supports_stored_generated_in_alter"]:
            converted = converted.replace("STORED", "VIRTUAL", 1)
            self.downgrade(f, "stored_column_to_virtual", f"{table}: {definition}")
        if "PRIMARY KEY" in up or "UNIQUE" in up:
            self.skip(f, "alter_add_constraint", f"{table}: {definition}")
            return ""
        return "ADD COLUMN " + converted

    # -- indexes and views -------------------------------------------------
    def _create_index(self, f: str, s: str) -> str:
        m = RE["create_index"].match(s)
        method = (m.group(4) or "").lower()
        table = _unquote(m.group(3))

        if method not in self.p["index_methods"]:
            self.skip(f, "index_method", f"{s} ({method})")
            return ""
        if table.lower() not in self.relations:
            self.skip(f, "relation_not_created", s)
            return ""

        open_idx = m.end() - 1
        close_idx = matching_paren(s, open_idx)
        if close_idx < 0:
            self.skip(f, "unrecognized_statement", s)
            return ""

        columns, tail = s[open_idx + 1:close_idx], s[close_idx + 1:]
        if not self.is_portable(columns) or (tail.strip(" ;") and not self.is_portable(tail)):
            self.skip(f, "postgres_only_expression", s)
            return ""
        for col in RE["quoted"].findall(columns):
            if col.lower() in self.dropped_columns[table.lower()]:
                self.skip(f, "references_dropped_column", s)
                return ""

        header = RE["concurrently"].sub("", re.sub(r"(?i)\s+USING\s+\w+\s*$", " ", s[:open_idx]))
        if "IF NOT EXISTS" not in header.upper():
            header = re.sub(r"(?i)(CREATE\s+(?:UNIQUE\s+)?INDEX)\s+", r"\1 IF NOT EXISTS ", header)

        columns = RE["cast"].sub("", RE["nulls_order"].sub("", RE["opclass"].sub("", columns)))
        tail = RE["cast"].sub("", RE["with_opts"].sub("", RE["include"].sub("", tail)))
        tail = re.sub(r"(?i)\s+TABLESPACE\s+\w+", "", tail).strip() or ";"

        self.table_indexes[table.lower()].append(
            (_unquote(m.group(2)), _identifiers(columns) | _identifiers(tail))
        )

        return f"{header.rstrip()} ({columns}){tail}"

    def _view(self, f: str, s: str) -> str:
        if not self.is_portable(s):
            self.skip(f, "postgres_only_expression", s)
            return ""
        out = s.replace("CREATE OR REPLACE VIEW", "CREATE VIEW IF NOT EXISTS", 1)
        vm = RE["view_name"].match(out)
        if vm:
            self.relations.add(_unquote(vm.group(1)).lower())
        return RE["cast"].sub("", out)

    def _matview(self, f: str, s: str) -> str:
        if not self.p["materialized_view_as_view"] or not self.is_portable(s):
            self.skip(f, "materialized_view", s)
            return ""
        out = RE["matview"].sub("CREATE VIEW IF NOT EXISTS ", s)
        out = re.sub(r"(?i)\s+WITH\s+NO\s+DATA\s*;?\s*$", "", out)
        vm = RE["view_name"].match(out)
        if vm:
            self.relations.add(_unquote(vm.group(1)).lower())
        self.downgrade(f, "materialized_view_to_view", s)
        return RE["cast"].sub("", out)


# --------------------------------------------------------------------------
# helpers
# --------------------------------------------------------------------------

IDENTIFIER_RE = re.compile(r'"([^"]+)"|\b([a-z_][a-z0-9_]*)\b', re.I)


def _identifiers(expr: str) -> set[str]:
    """Lowercased identifiers appearing in an expression, keywords included.

    Over-approximating is safe here: an extra name only makes a DROP COLUMN more
    conservative, never less."""
    found = set()
    for quoted, bare in IDENTIFIER_RE.findall(expr or ""):
        found.add((quoted or bare).lower())
    return found


def _unquote(v: str) -> str:
    return v.strip().strip('"')


def _first_line(s: str) -> str:
    line = s.split("\n", 1)[0].strip()
    return line[:117] + "..." if len(line) > 120 else line


def _normalize_type(sql_type: str) -> str:
    v = RE["array_suffix"].sub("", sql_type.strip())
    v = re.sub(r"\s+", " ", v)
    if "(" in v:
        v = v.split("(", 1)[0].strip()
    return v.strip('"').lower()


TRIM_FORMS = (
    (re.compile(r"(?i)\bTRIM\s*\(\s*BOTH\s+FROM\s+"), "TRIM("),
    (re.compile(r"(?i)\bTRIM\s*\(\s*LEADING\s+FROM\s+"), "LTRIM("),
    (re.compile(r"(?i)\bTRIM\s*\(\s*TRAILING\s+FROM\s+"), "RTRIM("),
)


def _rename_functions(expr: str, profile: dict) -> str:
    for pattern, replacement in TRIM_FORMS:
        expr = pattern.sub(replacement, expr)
    for src, dst in profile["function_renames"].items():
        expr = re.sub(r"(?i)\b" + re.escape(src) + r"\s*\(", dst + "(", expr)
    return expr


def _split_type(rest: str) -> tuple[str, str]:
    depth = 0
    for i, ch in enumerate(rest):
        if ch == "(":
            depth += 1
        elif ch == ")":
            depth -= 1
        elif ch in " \t\n" and depth == 0:
            remainder = rest[i:].strip()
            if remainder.startswith("(") or remainder.upper().startswith(
                ("PRECISION", "VARYING", "WITH TIME ZONE", "WITHOUT TIME ZONE")
            ):
                continue
            return rest[:i].strip(), remainder
    return rest.strip(), ""


def _split_default(value: str) -> tuple[str, str]:
    depth, in_quote = 0, False
    i = 0
    while i < len(value):
        ch = value[i]
        if in_quote:
            if ch == "'":
                in_quote = False
            i += 1
            continue
        if ch == "'":
            in_quote = True
        elif ch == "(":
            depth += 1
        elif ch == ")":
            depth -= 1
        elif ch in " \t\n" and depth == 0:
            remainder = value[i:].strip()
            if remainder.upper().startswith(COLUMN_KEYWORDS):
                return value[:i].strip(), remainder
        i += 1
    return value.strip(), ""


# --------------------------------------------------------------------------
# entry point
# --------------------------------------------------------------------------

def run(profile_name: str, source: str, target: str, report_path: str | None) -> Converter:
    profile = PROFILES[profile_name]
    converter = Converter(profile)

    names = sorted(n for n in os.listdir(source) if n.endswith(".up.sql"))

    os.makedirs(target, exist_ok=True)
    for existing in os.listdir(target):
        if existing.endswith(".sql"):
            os.remove(os.path.join(target, existing))

    written = 0
    for name in names:
        with open(os.path.join(source, name), encoding="utf-8") as fh:
            converted = converter.convert_file(name, fh.read())
        if not converted.strip():
            continue
        with open(os.path.join(target, name), "w", encoding="utf-8") as fh:
            fh.write(HEADER.format(source=name) + "\n" + converted)
        down = name.replace(".up.sql", ".down.sql")
        with open(os.path.join(target, down), "w", encoding="utf-8") as fh:
            fh.write(HEADER.format(source=down) + "\nSELECT 1;\n")
        written += 1

    _write_embed(target)

    counts = collections.Counter(reason for _, reason, _ in converter.skipped)
    print(f"{profile['display_name']}: {len(names)} source files -> {written} migration files")
    print(f"skipped {len(converter.skipped)} statements")
    for reason, count in counts.most_common():
        print(f"  {reason:<34} {count}")
    if converter.downgrades:
        print(f"downgraded {len(converter.downgrades)} statements")

    if report_path:
        with open(report_path, "w", encoding="utf-8") as fh:
            for kind, rows in (("DOWNGRADED", converter.downgrades), ("SKIPPED", converter.skipped)):
                fh.write(f"{kind}\n")
                for f, reason, detail in sorted(rows, key=lambda r: (r[1], r[0])):
                    fh.write(f"  {f}\t{reason}\t{detail}\n")
                fh.write("\n")
        print(f"wrote report to {report_path}")

    return converter


EMBED_GO = '''package migrations

import (
	"embed"
	"fmt"

	"github.com/uptrace/bun/migrate"
)

var Migrations = migrate.NewMigrations()

//go:embed *.sql
var sqlMigrations embed.FS

func init() {
	if err := Migrations.Discover(sqlMigrations); err != nil {
		panic(err)
	}
}

func Setup() *migrate.Migrations {
	migrations := migrate.NewMigrations()

	if err := migrations.Discover(sqlMigrations); err != nil {
		panic(fmt.Errorf("failed to discover migrations: %w", err))
	}

	return migrations
}
'''


def _write_embed(target: str):
    with open(os.path.join(target, "migrations.go"), "w", encoding="utf-8") as fh:
        fh.write(EMBED_GO)


def check(target: str) -> int:
    """Apply the generated migrations to a throwaway database and report failures."""
    import sqlite3
    import tempfile

    path = os.path.join(tempfile.mkdtemp(), "check.db")
    conn = sqlite3.connect(path)
    conn.execute("PRAGMA foreign_keys=OFF")

    applied = failed = 0
    failures: collections.Counter = collections.Counter()
    samples: list[str] = []

    for name in sorted(n for n in os.listdir(target) if n.endswith(".up.sql")):
        with open(os.path.join(target, name), encoding="utf-8") as fh:
            for statement in split_statements(fh.read()):
                try:
                    conn.execute(statement)
                    applied += 1
                except sqlite3.Error as exc:
                    failed += 1
                    failures[str(exc)[:70]] += 1
                    if len(samples) < 15:
                        samples.append(f"{name}: {exc}\n    {_first_line(statement)}")

    tables = conn.execute(
        "SELECT count(*) FROM sqlite_master WHERE type='table'"
    ).fetchone()[0]
    conn.close()

    print(f"\napplied {applied}/{applied + failed} statements; {tables} tables created")
    if failures:
        print("failures:")
        for msg, count in failures.most_common(15):
            print(f"  {msg:<72} {count}")
        print()
        for sample in samples:
            print(" -", sample)
    return failed


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("dialect", choices=sorted(PROFILES))
    parser.add_argument("--source", default="services/tms/internal/infrastructure/postgres/migrations")
    parser.add_argument("--target", default=None)
    parser.add_argument("--report", default=None)
    parser.add_argument("--check", action="store_true", help="apply the output to a scratch database")
    args = parser.parse_args()

    target = args.target or f"services/tms/internal/infrastructure/{args.dialect}/migrations"
    run(args.dialect, args.source, target, args.report)

    if args.check:
        if args.dialect != "sqlite":
            print(f"--check is only implemented for sqlite, not {args.dialect}")
            return 0
        return 1 if check(target) else 0
    return 0


if __name__ == "__main__":
    sys.exit(main())
