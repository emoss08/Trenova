package bunadopt

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/keel/migrate"
	"github.com/jackc/pgx/v5"
)

const (
	bunTable      = "public.bun_migrations"
	maxNamesShown = 20
)

var ErrUnadoptable = errors.New("bunadopt: the database Bun migrated does not match the migrations")

type Result struct {
	Version  string
	Recorded int
}

func (r Result) Adopted() bool {
	return r.Version != ""
}

func Adopt(
	ctx context.Context,
	conn *pgx.Conn,
	set migrate.Set,
	opts migrate.Options,
) (Result, error) {
	ms, err := set.Migrations()
	if err != nil {
		return Result{}, err
	}

	pending, err := needsAdoption(ctx, conn, opts)
	if err != nil || !pending {
		return Result{}, err
	}

	applied, err := bunVersions(ctx, conn)
	if err != nil || len(applied) == 0 {
		return Result{}, err
	}

	version, err := baselineVersion(ms, applied)
	if err != nil {
		return Result{}, err
	}

	m, err := migrate.New(conn, set, opts)
	if err != nil {
		return Result{}, err
	}

	recorded, err := m.Baseline(ctx, version)
	if err != nil {
		return Result{}, fmt.Errorf("bunadopt: record the migrations Bun applied: %w", err)
	}

	return Result{Version: version, Recorded: len(recorded)}, nil
}

func needsAdoption(ctx context.Context, conn *pgx.Conn, opts migrate.Options) (bool, error) {
	var bun, keel bool
	err := conn.QueryRow(
		ctx,
		"SELECT to_regclass($1) IS NOT NULL, to_regclass($2) IS NOT NULL",
		bunTable,
		keelTable(opts),
	).Scan(&bun, &keel)
	if err != nil {
		return false, fmt.Errorf("bunadopt: look for the bookkeeping tables: %w", err)
	}
	if !bun {
		return false, nil
	}
	if !keel {
		return true, nil
	}

	var recorded bool
	if err = conn.QueryRow(
		ctx,
		"SELECT EXISTS (SELECT 1 FROM "+keelTable(opts)+")",
	).Scan(&recorded); err != nil {
		return false, fmt.Errorf("bunadopt: read %s: %w", migrate.Table, err)
	}

	return !recorded, nil
}

func keelTable(opts migrate.Options) string {
	schema := opts.Schema
	if schema == "" {
		schema = "public"
	}

	return pgx.Identifier{schema, migrate.Table}.Sanitize()
}

func bunVersions(ctx context.Context, conn *pgx.Conn) (map[string]bool, error) {
	rows, err := conn.Query(ctx, "SELECT DISTINCT name FROM "+bunTable)
	if err != nil {
		return nil, fmt.Errorf("bunadopt: read bun_migrations: %w", err)
	}

	names, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("bunadopt: read bun_migrations: %w", err)
	}

	applied := make(map[string]bool, len(names))
	for _, name := range names {
		applied[strings.TrimSpace(name)] = true
	}

	return applied, nil
}

func baselineVersion(ms []migrate.Migration, applied map[string]bool) (string, error) {
	known := make(map[string]bool, len(ms))
	newest := ""
	for _, m := range ms {
		if m.Kind == migrate.KindRepeatable {
			continue
		}
		known[m.Version] = true
		if applied[m.Version] && m.Version > newest {
			newest = m.Version
		}
	}

	var unknown []string
	for version := range applied {
		if !known[version] {
			unknown = append(unknown, version)
		}
	}

	var gaps []string
	for _, m := range ms {
		if m.Kind != migrate.KindRepeatable && m.Version <= newest && !applied[m.Version] {
			gaps = append(gaps, m.ID())
		}
	}

	if len(unknown) == 0 && len(gaps) == 0 {
		return newest, nil
	}

	slices.Sort(unknown)
	var problems []string
	if len(gaps) > 0 {
		problems = append(problems, fmt.Sprintf(
			"Bun never applied %d migration(s) older than %s, the newest it applied: %s",
			len(gaps), newest, listed(gaps),
		))
	}
	if len(unknown) > 0 {
		problems = append(problems, fmt.Sprintf(
			"Bun applied %d version(s) this binary has no migration for: %s",
			len(unknown), listed(unknown),
		))
	}

	return "", fmt.Errorf(
		"%w: %s; apply or remove them with the Bun release that wrote them before adopting",
		ErrUnadoptable,
		strings.Join(problems, "; "),
	)
}

func listed(names []string) string {
	if len(names) <= maxNamesShown {
		return strings.Join(names, ", ")
	}

	return fmt.Sprintf(
		"%s and %d more",
		strings.Join(names[:maxNamesShown], ", "),
		len(names)-maxNamesShown,
	)
}
