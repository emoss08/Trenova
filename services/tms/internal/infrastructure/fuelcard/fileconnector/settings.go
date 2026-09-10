package fileconnector

import (
	"errors"
	"path"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/fuelimport"
	"github.com/emoss08/trenova/pkg/rateimport"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/sftp"
	"github.com/emoss08/trenova/shared/stringutils"
)

const defaultArchiveFolder = "processed"

type settings struct {
	sftp             sftp.Config
	remoteDirectory  string
	archiveDirectory string
	filePattern      string
	sourceFormat     fuelpurchase.SourceFormat
	layout           fuelimport.FixedWidthLayout
	defaultFuelType  domaintypes.IFTAFuelType
	defaultCurrency  string
}

func (s *settings) emptyResult() *services.FetchFuelTransactionsResult {
	return &services.FetchFuelTransactionsResult{
		Staged: &fuelimport.StageResult{},
		Format: s.sourceFormat,
	}
}

// readSheet turns one file into a sheet. Delimited exports go through the same
// reader a spreadsheet upload uses; a fixed-width export is sliced by the
// configured layout first, and then behaves identically.
func (s *settings) readSheet(content []byte) (*rateimport.Sheet, error) {
	if s.sourceFormat == fuelpurchase.SourceFormatFixedWidth {
		return fuelimport.ReadFixedWidth(content, s.layout)
	}

	return rateimport.ReadCSV(content)
}

func settingsFrom(config map[string]string) (*settings, error) {
	remoteDirectory := strings.TrimSpace(config[integration.ConfigKeyFuelRemoteDir])
	if remoteDirectory == "" {
		return nil, errors.New("a remote directory is required to read fuel card exports")
	}

	sftpCfg, err := sftpConfigFrom(config)
	if err != nil {
		return nil, err
	}

	resolved := &settings{
		sftp:            sftpCfg,
		remoteDirectory: remoteDirectory,
		archiveDirectory: stringutils.WithDefault(
			config[integration.ConfigKeyFuelArchiveDir],
			path.Join(remoteDirectory, defaultArchiveFolder),
		),
		filePattern:  strings.TrimSpace(config[integration.ConfigKeyFuelFilePattern]),
		sourceFormat: fuelpurchase.SourceFormatCSV,
		defaultCurrency: stringutils.WithDefault(
			config[integration.ConfigKeyFuelCurrency],
			money.DefaultCurrencyCode,
		),
	}

	if fuelType := domaintypes.IFTAFuelType(
		strings.TrimSpace(config[integration.ConfigKeyFuelDefaultFuelType]),
	); fuelType.IsValid() {
		resolved.defaultFuelType = fuelType
	}

	if !strings.EqualFold(
		config[integration.ConfigKeyFuelFileFormat],
		integration.FuelFileFormatFixedWidth,
	) {
		return resolved, nil
	}

	layout, err := fuelimport.ParseFixedWidthLayout(config[integration.ConfigKeyFuelFixedWidthLayout])
	if err != nil {
		return nil, err
	}

	resolved.sourceFormat = fuelpurchase.SourceFormatFixedWidth
	resolved.layout = layout

	return resolved, nil
}
