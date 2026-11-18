package tms_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/emoss08/trenova/operator/internal/tms"
	"github.com/emoss08/trenova/operator/internal/tmstest"
)

func TestReadsWhatTheInstancesDataTakes(t *testing.T) {
	t.Parallel()
	srv := tmstest.New(t, "token")
	measured := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	db, search := int64(5<<30), int64(1<<30)
	srv.SetDataUsage(&tms.DataUsage{MeasuredAt: measured, DatabaseBytes: &db, SearchBytes: &search})
	c := client(t, fake.NewClientset(secretWith("token")), srv.URL)

	got, err := c.DataUsage(t.Context())

	require.NoError(t, err)
	require.True(t, got.MeasuredAt.Equal(measured))
	require.Equal(t, db, *got.DatabaseBytes)
	require.Equal(t, search, *got.SearchBytes)
	require.Nil(t, got.DocumentsBytes, "a figure Trenova could not measure is absent")
	require.Equal(t, 1, srv.DataUsageCalls())
}

func TestAReleaseWithoutDataUsageAnswersErrNoDataUsage(t *testing.T) {
	t.Parallel()
	srv := tmstest.New(t, "token")
	srv.FailDataUsage(http.StatusNotFound)
	c := client(t, fake.NewClientset(secretWith("token")), srv.URL)

	_, err := c.DataUsage(t.Context())
	require.ErrorIs(t, err, tms.ErrNoDataUsage)

	srv.FailDataUsage(http.StatusInternalServerError)
	_, err = c.DataUsage(t.Context())
	require.Error(t, err)
	require.NotErrorIs(t, err, tms.ErrNoDataUsage)
}
