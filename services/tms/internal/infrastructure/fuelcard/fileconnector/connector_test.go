package fileconnector_test

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/fuelcard/fileconnector"
	"github.com/emoss08/trenova/shared/sftp"
	"github.com/stretchr/testify/require"
)

type fakeTransport struct {
	files    map[string]string
	sizes    map[string]int64
	archived []string
	listErr  error
	readErr  error
	closed   bool
}

func newFakeTransport(files map[string]string) *fakeTransport {
	return &fakeTransport{files: files, sizes: map[string]int64{}}
}

func (f *fakeTransport) ListFiles(directory string) ([]sftp.RemoteFile, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}

	entries := make([]sftp.RemoteFile, 0, len(f.files))
	for name, contents := range f.files {
		size := int64(len(contents))
		if override, ok := f.sizes[name]; ok {
			size = override
		}
		entries = append(entries, sftp.RemoteFile{
			Path: directory + "/" + name,
			Name: name,
			Size: size,
		})
	}

	return entries, nil
}

func (f *fakeTransport) ReadFile(remotePath string) (string, error) {
	if f.readErr != nil {
		return "", f.readErr
	}

	for name, contents := range f.files {
		if remotePath == "/outbound/"+name {
			return contents, nil
		}
	}

	return "", errors.New("no such file: " + remotePath)
}

func (f *fakeTransport) Archive(remotePath, _ string) error {
	f.archived = append(f.archived, remotePath)
	return nil
}

func (f *fakeTransport) Close() error {
	f.closed = true
	return nil
}

func newConnector(transport fileconnector.Transport) *fileconnector.Connector {
	return fileconnector.NewWithTransport(
		integration.TypeComdataFuel,
		func(context.Context, sftp.Config) (fileconnector.Transport, error) {
			return transport, nil
		},
	)
}

func baseConfig() map[string]string {
	return map[string]string{
		integration.ConfigKeyFuelHost:         "sftp.example.com",
		integration.ConfigKeyFuelUsername:     "trenova",
		integration.ConfigKeyFuelAuthMode:     integration.FuelAuthModePassword,
		integration.ConfigKeyFuelPassword:     "secret",
		integration.ConfigKeyFuelKnownHostKey: "ssh-ed25519 AAAA",
		integration.ConfigKeyFuelRemoteDir:    "/outbound",
	}
}

func fetch(
	t *testing.T,
	connector *fileconnector.Connector,
	config map[string]string,
) *services.FetchFuelTransactionsResult {
	t.Helper()

	result, err := connector.FetchTransactions(t.Context(), &services.FetchFuelTransactionsRequest{
		Provider: fuelpurchase.CardProviderComdata,
		Config:   config,
	})
	require.NoError(t, err)

	return result
}

const delimitedExport = "Trans Date,Card Number,Unit,State,Product,Gallons,Amount,Trans ID\n" +
	"01/05/2026,4411,TRC-1,TX,Diesel,125.400,501.60,C-1001\n" +
	"01/06/2026,4412,TRC-2,OK,Diesel,98.200,392.80,C-1002\n"

func TestFetchStagesADelimitedExport(t *testing.T) {
	t.Parallel()

	transport := newFakeTransport(map[string]string{"AC00029_20260106.csv": delimitedExport})
	result := fetch(t, newConnector(transport), baseConfig())

	require.Len(t, result.Staged.Rows, 2)
	for _, row := range result.Staged.Rows {
		require.NoError(t, row.Err)
		require.NotNil(t, row.Parsed)
	}
	require.Equal(t, "TRC-1", result.Staged.Rows[0].Parsed.TractorCode)
	require.Equal(t, "4411", result.Staged.Rows[0].Parsed.CardLastFour)
	require.Equal(t, "C-1001", result.Staged.Rows[0].Parsed.Reference)
	require.Equal(t, fuelpurchase.SourceFormatCSV, result.Format)
}

func TestFetchStagesAFixedWidthExport(t *testing.T) {
	t.Parallel()

	config := baseConfig()
	config[integration.ConfigKeyFuelFileFormat] = integration.FuelFileFormatFixedWidth
	config[integration.ConfigKeyFuelFixedWidthLayout] =
		"Trans Date:1-10,Card Number:11-14,Unit:15-19,State:20-21,Product:22-27,Gallons:28-34,Amount:35-41,Trans ID:42-47"

	transport := newFakeTransport(map[string]string{
		"AC00029": "01/05/20264411TRC-1TXDiesel125.400 501.60C-1001\n",
	})
	result := fetch(t, newConnector(transport), config)

	require.Len(t, result.Staged.Rows, 1)
	row := result.Staged.Rows[0]
	require.NoError(t, row.Err)
	require.Equal(t, "TRC-1", row.Parsed.TractorCode)
	require.Equal(t, "4411", row.Parsed.CardLastFour)
	require.Equal(t, "TX", row.Parsed.JurisdictionCode)
	require.Equal(t, fuelpurchase.SourceFormatFixedWidth, result.Format)
}

// A fixed-width connection without a layout must fail loudly. Reading the file
// with a guessed layout would take every field from the wrong columns while
// looking like it had worked.
func TestFetchRefusesFixedWidthWithoutALayout(t *testing.T) {
	t.Parallel()

	config := baseConfig()
	config[integration.ConfigKeyFuelFileFormat] = integration.FuelFileFormatFixedWidth

	transport := newFakeTransport(map[string]string{"AC00029": "anything\n"})
	_, err := newConnector(transport).FetchTransactions(
		t.Context(),
		&services.FetchFuelTransactionsRequest{
			Provider: fuelpurchase.CardProviderComdata,
			Config:   config,
		},
	)
	require.ErrorContains(t, err, "layout is empty")
}

func TestFetchReportsTheCardsAFileTouched(t *testing.T) {
	t.Parallel()

	transport := newFakeTransport(map[string]string{"export.csv": delimitedExport})
	result := fetch(t, newConnector(transport), baseConfig())

	require.Len(t, result.Cards, 2)
	require.Equal(t, "4411", result.Cards[0].LastFour)
	require.Equal(t, fuelpurchase.CardProviderComdata, result.Cards[0].Provider)
}

func TestFetchReportsEachCardOnlyOnce(t *testing.T) {
	t.Parallel()

	transport := newFakeTransport(map[string]string{
		"export.csv": "Trans Date,Card Number,Unit,State,Product,Gallons,Amount,Trans ID\n" +
			"01/05/2026,4411,TRC-1,TX,Diesel,125.400,501.60,C-1001\n" +
			"01/06/2026,4411,TRC-1,TX,Diesel,98.200,392.80,C-1002\n",
	})
	result := fetch(t, newConnector(transport), baseConfig())

	require.Len(t, result.Cards, 1)
	require.Equal(t, "4411", result.Cards[0].LastFour)
}

func TestFetchArchivesFilesItRead(t *testing.T) {
	t.Parallel()

	transport := newFakeTransport(map[string]string{"export.csv": delimitedExport})
	fetch(t, newConnector(transport), baseConfig())

	require.Equal(t, []string{"/outbound/export.csv"}, transport.archived)
}

// A file that could not be mapped onto Trenova's fields stays where it is, so
// fixing the configuration and running again picks it up rather than losing it.
func TestFetchLeavesAnUnmappableFileInPlace(t *testing.T) {
	t.Parallel()

	transport := newFakeTransport(map[string]string{
		"export.csv": "Colour,Shape\nred,round\n",
	})
	result := fetch(t, newConnector(transport), baseConfig())

	require.True(t, result.Staged.HasProblems())
	require.Empty(t, transport.archived)
}

func TestFetchOnlyReadsFilesMatchingThePattern(t *testing.T) {
	t.Parallel()

	config := baseConfig()
	config[integration.ConfigKeyFuelFilePattern] = "AC00029*.csv"

	transport := newFakeTransport(map[string]string{
		"AC00029_20260106.csv": delimitedExport,
		"unrelated.txt":        "ignore me",
	})
	result := fetch(t, newConnector(transport), config)

	require.Len(t, result.Staged.Rows, 2)
	require.Equal(t, []string{"/outbound/AC00029_20260106.csv"}, transport.archived)
}

func TestFetchReturnsNothingWhenTheDirectoryIsEmpty(t *testing.T) {
	t.Parallel()

	result := fetch(t, newConnector(newFakeTransport(nil)), baseConfig())

	require.Empty(t, result.Staged.Rows)
	require.Empty(t, result.Cards)
}

// A header row with nothing under it is an ordinary quiet day for a daily export,
// not a failure that should stop the run.
func TestFetchToleratesAFileWithNoRows(t *testing.T) {
	t.Parallel()

	transport := newFakeTransport(map[string]string{
		"empty.csv": "Trans Date,Card Number,Unit,State,Product,Gallons,Amount,Trans ID\n",
	})
	result := fetch(t, newConnector(transport), baseConfig())

	require.Empty(t, result.Staged.Rows)
}

func TestFetchRefusesAFileOverTheImportLimit(t *testing.T) {
	t.Parallel()

	transport := newFakeTransport(map[string]string{"huge.csv": delimitedExport})
	transport.sizes["huge.csv"] = 64 << 20

	_, err := newConnector(transport).FetchTransactions(
		t.Context(),
		&services.FetchFuelTransactionsRequest{
			Provider: fuelpurchase.CardProviderComdata,
			Config:   baseConfig(),
		},
	)
	require.ErrorContains(t, err, "32 MiB")
}

func TestFetchRefusesAnUndialableConfiguration(t *testing.T) {
	t.Parallel()

	config := baseConfig()
	delete(config, integration.ConfigKeyFuelKnownHostKey)

	_, err := newConnector(newFakeTransport(nil)).FetchTransactions(
		t.Context(),
		&services.FetchFuelTransactionsRequest{
			Provider: fuelpurchase.CardProviderComdata,
			Config:   config,
		},
	)
	require.ErrorContains(t, err, "known host key is required")
}

func TestFetchRefusesAConfigurationWithoutARemoteDirectory(t *testing.T) {
	t.Parallel()

	config := baseConfig()
	delete(config, integration.ConfigKeyFuelRemoteDir)

	_, err := newConnector(newFakeTransport(nil)).FetchTransactions(
		t.Context(),
		&services.FetchFuelTransactionsRequest{
			Provider: fuelpurchase.CardProviderComdata,
			Config:   config,
		},
	)
	require.ErrorContains(t, err, "remote directory is required")
}

func TestFetchClosesTheTransport(t *testing.T) {
	t.Parallel()

	transport := newFakeTransport(map[string]string{"export.csv": delimitedExport})
	fetch(t, newConnector(transport), baseConfig())

	require.True(t, transport.closed)
}

func TestTestConnectionReportsAnUnreadableDirectory(t *testing.T) {
	t.Parallel()

	transport := newFakeTransport(nil)
	transport.listErr = errors.New("permission denied")

	err := newConnector(transport).TestConnection(t.Context(), baseConfig())
	require.ErrorContains(t, err, "/outbound is not readable")
}

func TestTestConnectionPassesForAReadableDirectory(t *testing.T) {
	t.Parallel()

	require.NoError(
		t,
		newConnector(newFakeTransport(nil)).TestConnection(t.Context(), baseConfig()),
	)
}

func TestCapabilityIsIFTAGrade(t *testing.T) {
	t.Parallel()

	require.True(t, newConnector(newFakeTransport(nil)).Capability().IsIFTAGrade())
}
