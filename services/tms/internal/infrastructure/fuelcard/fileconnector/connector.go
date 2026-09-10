package fileconnector

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/fuelimport"
	"github.com/emoss08/trenova/pkg/rateimport"
	"github.com/emoss08/trenova/shared/sftp"
)

const (
	maxFilesPerRun = 25
	maxFileBytes   = 32 << 20
)

// Transport is the slice of an SFTP session this connector needs, so a test can
// stand in for a server.
type Transport interface {
	ListFiles(directory string) ([]sftp.RemoteFile, error)
	ReadFile(remotePath string) (string, error)
	Archive(remotePath, archiveDirectory string) error
	Close() error
}

// Connector reads transaction exports a card network drops on a customer's SFTP
// server. It is the only path that works without a partner agreement with the
// network, because the export is something the customer already receives.
//
// One connector serves every file-based network; the integration type it was
// built for decides which header dictionary the rows are matched against.
type Connector struct {
	integrationType integration.Type
	dial            func(ctx context.Context, cfg sftp.Config) (Transport, error)
}

func New(integrationType integration.Type) *Connector {
	return NewWithTransport(integrationType, nil)
}

func NewWithTransport(
	integrationType integration.Type,
	dial func(ctx context.Context, cfg sftp.Config) (Transport, error),
) *Connector {
	if dial == nil {
		dial = func(ctx context.Context, cfg sftp.Config) (Transport, error) {
			return sftp.Dial(ctx, cfg)
		}
	}

	return &Connector{integrationType: integrationType, dial: dial}
}

func (c *Connector) IntegrationType() integration.Type { return c.integrationType }

// Capability is IFTA-grade because both fleet networks report Level III detail on
// a fuel purchase. Whether a given row actually carries gallons is decided when it
// is parsed, not here.
func (c *Connector) Capability() services.FuelFeedCapability {
	return services.FuelFeedCapabilityIFTAGrade
}

func (c *Connector) TestConnection(ctx context.Context, config map[string]string) error {
	settings, err := settingsFrom(config)
	if err != nil {
		return err
	}

	transport, err := c.dial(ctx, settings.sftp)
	if err != nil {
		return err
	}
	defer transport.Close()

	if _, err = transport.ListFiles(settings.remoteDirectory); err != nil {
		return fmt.Errorf("%s is not readable: %w", settings.remoteDirectory, err)
	}

	return nil
}

func (c *Connector) FetchTransactions(
	ctx context.Context,
	req *services.FetchFuelTransactionsRequest,
) (*services.FetchFuelTransactionsResult, error) {
	settings, err := settingsFrom(req.Config)
	if err != nil {
		return nil, err
	}

	transport, err := c.dial(ctx, settings.sftp)
	if err != nil {
		return nil, err
	}
	defer transport.Close()

	files, err := transport.ListFiles(settings.remoteDirectory)
	if err != nil {
		return nil, err
	}

	selected := selectFiles(files, settings.filePattern)
	if len(selected) == 0 {
		return settings.emptyResult(), nil
	}

	sheet, references, err := c.readSheets(transport, selected, settings)
	if err != nil {
		return nil, err
	}
	if sheet == nil {
		return settings.emptyResult(), nil
	}

	staged := fuelimport.Stage(sheet, fuelimport.StageOptions{
		Provider:        req.Provider,
		DefaultFuelType: settings.defaultFuelType,
		DefaultCurrency: settings.defaultCurrency,
	})

	// Archiving is what stops a file being read again, so it only happens once
	// the file has been staged without a structural problem. A file left in place
	// is re-read on the next run and deduplicated by transaction reference.
	if !staged.HasProblems() {
		for _, file := range selected {
			_ = transport.Archive(file.Path, settings.archiveDirectory)
		}
	}

	return &services.FetchFuelTransactionsResult{
		Staged:    staged,
		Reference: strings.Join(references, ", "),
		Format:    settings.sourceFormat,
		Cards:     cardsFromStaged(staged, req.Provider),
	}, nil
}

// readSheets folds every selected file into one sheet. The files come from one
// account with one layout, so treating them as a single statement keeps the
// in-file duplicate detection working across a batch of daily exports.
func (c *Connector) readSheets(
	transport Transport,
	files []sftp.RemoteFile,
	settings *settings,
) (*rateimport.Sheet, []string, error) {
	var combined *rateimport.Sheet
	references := make([]string, 0, len(files))

	for _, file := range files {
		if file.Size > maxFileBytes {
			return nil, nil, fmt.Errorf("%s is larger than the 32 MiB import limit", file.Name)
		}

		contents, err := transport.ReadFile(file.Path)
		if err != nil {
			return nil, nil, fmt.Errorf("read %s: %w", file.Name, err)
		}

		sheet, err := settings.readSheet([]byte(contents))
		if err != nil {
			// A file with a header and nothing under it is a normal quiet day,
			// not a failure worth stopping the run for.
			if errors.Is(err, rateimport.ErrNoRows) ||
				errors.Is(err, fuelimport.ErrNoFixedWidthRecords) {
				references = append(references, file.Path)
				continue
			}

			return nil, nil, fmt.Errorf("read %s: %w", file.Name, err)
		}

		references = append(references, file.Path)
		combined = appendSheet(combined, sheet)
	}

	return combined, references, nil
}

// appendSheet joins sheets that share a header row. A file whose headers differ
// is left out rather than misaligned onto another file's columns.
func appendSheet(combined, next *rateimport.Sheet) *rateimport.Sheet {
	if combined == nil {
		return next
	}
	if !sameHeaders(combined.Headers, next.Headers) {
		return combined
	}

	combined.Rows = append(combined.Rows, next.Rows...)

	return combined
}

func sameHeaders(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if !strings.EqualFold(strings.TrimSpace(left[i]), strings.TrimSpace(right[i])) {
			return false
		}
	}

	return true
}

func selectFiles(files []sftp.RemoteFile, pattern string) []sftp.RemoteFile {
	selected := make([]sftp.RemoteFile, 0, len(files))
	for _, file := range files {
		if !matchesPattern(file.Name, pattern) {
			continue
		}

		selected = append(selected, file)
		if len(selected) == maxFilesPerRun {
			break
		}
	}

	return selected
}

func matchesPattern(name, pattern string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" || pattern == "*" {
		return true
	}

	matched, err := path.Match(pattern, name)
	if err != nil {
		// An unusable pattern must not silently swallow every file. Reading them
		// all is the safe failure: a row that should not be there is caught
		// downstream, while a file never read is simply lost.
		return true
	}

	return matched
}

// cardsFromStaged reports the distinct cards the file touched. What to do about
// ones nobody has registered is the sync's decision, not the connector's.
func cardsFromStaged(
	staged *fuelimport.StageResult,
	provider fuelpurchase.CardProvider,
) []services.ProviderCard {
	seen := make(map[string]struct{}, 8)
	cards := make([]services.ProviderCard, 0, 8)

	for _, row := range staged.Rows {
		if row.Parsed == nil || row.Parsed.CardLastFour == "" {
			continue
		}
		if _, ok := seen[row.Parsed.CardLastFour]; ok {
			continue
		}

		seen[row.Parsed.CardLastFour] = struct{}{}
		cards = append(cards, services.ProviderCard{
			LastFour: row.Parsed.CardLastFour,
			Provider: provider,
			Label:    provider.Label() + " " + row.Parsed.CardLastFour,
		})
	}

	return cards
}
