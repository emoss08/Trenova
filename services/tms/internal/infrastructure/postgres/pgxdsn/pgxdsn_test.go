package pgxdsn_test

import (
	"net/url"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/pgxdsn"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeTurnsTheDialTimeoutIntoAConnectTimeout(t *testing.T) {
	t.Parallel()

	dsn, err := pgxdsn.Normalize(
		"postgres://app:secret@db:5432/trenova?sslmode=disable&application_name=trenova&dial_timeout=10s",
		pgxdsn.Options{},
	)
	require.NoError(t, err)

	u, err := url.Parse(dsn)
	require.NoError(t, err)
	assert.Equal(t, "10", u.Query().Get("connect_timeout"))
	assert.Empty(t, u.Query().Get("dial_timeout"), "pgx would send it to the server as a setting")
	assert.Equal(t, "trenova", u.Query().Get("application_name"))

	cfg, err := pgx.ParseConfig(dsn)
	require.NoError(t, err)
	assert.NotContains(t, cfg.RuntimeParams, "dial_timeout")
	assert.Equal(t, "secret", cfg.Password)
}

func TestNormalizeRoundsAPartialSecondUp(t *testing.T) {
	t.Parallel()

	dsn, err := pgxdsn.Normalize("postgres://db/trenova?dial_timeout=1500ms", pgxdsn.Options{})
	require.NoError(t, err)

	u, err := url.Parse(dsn)
	require.NoError(t, err)
	assert.Equal(t, "2", u.Query().Get("connect_timeout"))
}

func TestNormalizeNamesTheApplication(t *testing.T) {
	t.Parallel()

	dsn, err := pgxdsn.Normalize(
		"postgres://db/trenova?application_name=trenova",
		pgxdsn.Options{ApplicationName: "trenova-db"},
	)
	require.NoError(t, err)

	cfg, err := pgx.ParseConfig(dsn)
	require.NoError(t, err)
	assert.Equal(t, "trenova-db", cfg.RuntimeParams["application_name"])
}

func TestNormalizeRefusesWhatIsNotAPostgresURL(t *testing.T) {
	t.Parallel()

	_, err := pgxdsn.Normalize("host=db dbname=trenova", pgxdsn.Options{})
	require.Error(t, err)

	_, err = pgxdsn.Normalize("postgres://db/trenova?dial_timeout=soon", pgxdsn.Options{})
	require.Error(t, err)
}

func TestNormalizeKeepsThePasswordOutOfItsErrors(t *testing.T) {
	t.Parallel()

	_, err := pgxdsn.Normalize("mysql://app:hunter2@db/trenova", pgxdsn.Options{})
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "hunter2")
}
