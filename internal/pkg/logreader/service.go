package logreader

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/fsnotify/fsnotify"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger      *logger.Logger
	Config      *config.Config
	PermService services.PermissionService
}

// Service provides methods to read and stream logs
type Service struct {
	l          *zerolog.Logger
	cfg        *config.Config
	maxEntries int
	mu         sync.RWMutex
	clients    map[*LogClient]bool
}

// NewService creates a new log reader service
func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "logreader").
		Logger()

	service := &Service{
		l:          &log,
		cfg:        p.Config,
		maxEntries: 1000,
		clients:    make(map[*LogClient]bool),
	}

	if err := service.watchLogFile(context.Background()); err != nil {
		log.Error().Err(err).Msg("failed to watch log file")
	}

	return service
}

func (s *Service) GetCurrentLogs(ctx context.Context, opts *repositories.ListLogOptions) ([]repositories.LogEntry, error) {
	log := s.l.With().Str("operation", "GetCurrentLogs").
		Time("startDate", opts.StartDate).
		Time("endDate", opts.EndDate).
		Logger()

	entires, err := s.readLogs(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to read logs")
		return nil, err
	}

	return entires, nil
}

// BroadcastLogEntry sends a log entry to all connected clients
func (s *Service) BroadcastLogEntry(entry *repositories.LogEntry) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for client := range s.clients {
		go func(c *LogClient) {
			if err := c.SendLogEntry(entry); err != nil {
				// Unregister failed clients
				s.UnregisterClient(c)
			}
		}(client)
	}
}

// GetAvailableLogFiles returns a list of available log files
func (s *Service) GetAvailableLogFiles() ([]string, error) {
	log := s.l.With().
		Str("operation", "GetAvailableLogFiles").
		Logger()

	files, err := s.listLogFiles()
	if err != nil {
		log.Error().Err(err).Msg("failed to list log files")
		return nil, eris.Wrap(err, "list log files")
	}

	return files, nil
}

func (s *Service) readLogs(ctx context.Context, opts *repositories.ListLogOptions) ([]repositories.LogEntry, error) {
	log := s.l.With().
		Str("operation", "readLogs").
		Time("startDate", opts.StartDate).
		Time("endDate", opts.EndDate).
		Logger()

	// Open and prepare log file
	file, err := s.openLogFile(log)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Scan and parse log entries
	entries, err := s.scanLogEntries(ctx, file, opts, log)
	if err != nil {
		return nil, err
	}

	// Sort and paginate results
	return s.processResults(entries, opts), nil
}

// openLogFile opens the current log file
func (s *Service) openLogFile(log zerolog.Logger) (*os.File, error) {
	currentLogFile := filepath.Join(s.cfg.Log.FileConfig.Path, s.cfg.Log.FileConfig.FileName)

	file, err := os.Open(currentLogFile)
	if err != nil {
		if os.IsNotExist(err) {
			log.Info().Str("file", currentLogFile).Msg("log file does not exist")
			return nil, eris.Wrap(err, "open log file")
		}
		return nil, eris.Wrap(err, "open log file")
	}

	return file, nil
}

// scanLogEntries scans and parses log entries from the file
func (s *Service) scanLogEntries(ctx context.Context, file *os.File, opts *repositories.ListLogOptions, log zerolog.Logger) ([]repositories.LogEntry, error) {
	// Handle case where file doesn't exist
	if file == nil {
		return []repositories.LogEntry{}, nil
	}

	var entries []repositories.LogEntry
	scanner := bufio.NewScanner(file)

	// Use a larger buffer for scanning
	const maxScanSize = 1024 * 1024 // 1MB
	buf := make([]byte, maxScanSize)
	scanner.Buffer(buf, maxScanSize)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return entries, ctx.Err()
		default:
			entry, ok := s.parseLogEntry(scanner.Bytes(), log)
			if !ok {
				continue
			}

			if s.shouldSkipEntry(entry, opts) {
				continue
			}

			entries = append(entries, entry)

			// Check if we've reached the maximum number of entries
			if len(entries) >= s.maxEntries {
				log.Info().
					Int("maxEntries", s.maxEntries).
					Msg("reached maximum number of entries")
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, eris.Wrap(err, "scan log file")
	}

	return entries, nil
}

// parseLogEntry attempts to parse a log entry from a line
func (s *Service) parseLogEntry(line []byte, log zerolog.Logger) (repositories.LogEntry, bool) {
	var entry repositories.LogEntry
	if err := sonic.Unmarshal(line, &entry); err != nil {
		log.Warn().
			Err(err).
			Str("line", string(line)).
			Msg("failed to parse log entry, skipping")
		return repositories.LogEntry{}, false
	}
	return entry, true
}

// shouldSkipEntry checks if an entry should be skipped based on filters
func (s *Service) shouldSkipEntry(entry repositories.LogEntry, opts *repositories.ListLogOptions) bool {
	// Apply time filter if specified
	if !opts.StartDate.IsZero() && entry.Time.Before(opts.StartDate) {
		return true
	}
	if !opts.EndDate.IsZero() && entry.Time.After(opts.EndDate) {
		return true
	}
	return false
}

// processResults sorts and applies pagination to log entries
func (s *Service) processResults(entries []repositories.LogEntry, opts *repositories.ListLogOptions) []repositories.LogEntry {
	if len(entries) == 0 {
		return entries
	}

	// Sort entries by timestamp in descending order (newest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Time.After(entries[j].Time)
	})

	// Apply limit and offset if specified
	if opts.LimitOffsetQueryOptions != nil {
		return s.applyPagination(entries, opts)
	}

	return entries
}

// applyPagination applies pagination parameters to the result set
func (s *Service) applyPagination(entries []repositories.LogEntry, opts *repositories.ListLogOptions) []repositories.LogEntry {
	start := opts.Offset
	end := opts.Offset + opts.Limit

	if start >= len(entries) {
		return []repositories.LogEntry{}
	}
	if end > len(entries) {
		end = len(entries)
	}

	return entries[start:end]
}

func (s *Service) listLogFiles() ([]string, error) {
	log := s.l.With().
		Str("operation", "listLogFiles").
		Logger()

	// Ensure the log directory exists
	if err := os.MkdirAll(s.cfg.Log.FileConfig.Path, 0o755); err != nil {
		return nil, eris.Wrap(err, "create log directory")
	}

	// Get all log files (including compressed ones)
	pattern := s.cfg.Log.FileConfig.Path
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, eris.Wrap(err, "glob log files")
	}

	// Create a slice to hold file information
	type fileInfo struct {
		path    string
		modTime time.Time
	}

	fileInfos := make([]fileInfo, 0, len(files))
	for _, file := range files {
		info, sErr := os.Stat(file)
		if sErr != nil {
			log.Warn().
				Err(sErr).
				Str("file", file).
				Msg("failed to stat file, skipping")
			continue
		}

		fileInfos = append(fileInfos, fileInfo{
			path:    file,
			modTime: info.ModTime(),
		})
	}

	// Sort files by modification time (newest first)
	sort.Slice(fileInfos, func(i, j int) bool {
		return fileInfos[i].modTime.After(fileInfos[j].modTime)
	})

	// Extract just the paths
	result := make([]string, len(fileInfos))
	for i, fi := range fileInfos {
		// Convert to relative path for better presentation
		relPath, rErr := filepath.Rel(s.cfg.Log.FileConfig.Path, fi.path)
		if rErr != nil {
			log.Warn().
				Err(rErr).
				Str("file", fi.path).
				Msg("failed to get relative path, using absolute")
			result[i] = fi.path
		} else {
			result[i] = relPath
		}
	}

	return result, nil
}

// Helper method to check if a file is compressed
func isCompressedFile(filename string) bool {
	return strings.HasSuffix(filename, ".gz") ||
		strings.HasSuffix(filename, ".zip") ||
		strings.HasSuffix(filename, ".bz2")
}

// CleanupOldLogs removes log files that exceed the configured retention period
func (s *Service) CleanupOldLogs(ctx context.Context) error {
	log := s.l.With().
		Str("operation", "CleanupOldLogs").
		Logger()

	files, err := s.listLogFiles()
	if err != nil {
		return eris.Wrap(err, "list log files")
	}

	maxAge := time.Duration(s.cfg.Log.FileConfig.MaxAge) * 24 * time.Hour
	cutoffTime := time.Now().Add(-maxAge)

	for _, file := range files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			fullPath := filepath.Join(s.cfg.Log.FileConfig.Path, file)
			info, iErr := os.Stat(fullPath)
			if iErr != nil {
				log.Warn().
					Err(iErr).
					Str("file", fullPath).
					Msg("failed to stat file, skipping")
				continue
			}

			if info.ModTime().Before(cutoffTime) {
				if rErr := os.Remove(fullPath); rErr != nil {
					log.Error().
						Err(rErr).
						Str("file", fullPath).
						Msg("failed to remove old log file")
					continue
				}
				log.Info().
					Str("file", fullPath).
					Time("modTime", info.ModTime()).
					Msg("removed old log file")
			}
		}
	}

	return nil
}

// GetLogFileInfo returns detailed information about a specific log file
func (s *Service) GetLogFileInfo(filename string) (*repositories.LogFileInfo, error) {
	fullPath := filepath.Join(s.cfg.Log.FileConfig.Path, filename)

	// Prevent directory traversal
	if !strings.HasPrefix(fullPath, s.cfg.Log.FileConfig.Path) {
		return nil, errors.NewValidationError("filename", errors.ErrInvalid, "Invalid file path")
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.NewNotFoundError("Log file not found")
		}
		return nil, eris.Wrap(err, "stat log file")
	}

	// Get the file size in a human-readable format
	size := byteCountIEC(info.Size())

	return &repositories.LogFileInfo{
		Name:         filename,
		Size:         size,
		ModTime:      info.ModTime(),
		IsCompressed: isCompressedFile(filename),
	}, nil
}

func (s *Service) watchLogFile(ctx context.Context) error {
	s.l.Info().
		Str("logPath", s.cfg.Log.FileConfig.Path).
		Str("fileName", s.cfg.Log.FileConfig.FileName).
		Msg("starting log file watcher")

	// Setup file watcher
	watcher, file, reader, err := s.setupFileWatcher()
	if err != nil {
		return err
	}

	// Start goroutine to handle file events
	go s.handleFileEvents(ctx, watcher, file, reader)

	// Start goroutine to handle file rotation
	go s.checkFileRotation(ctx, file, reader)

	return nil
}

// setupFileWatcher initializes the file watcher and opens the log file
func (s *Service) setupFileWatcher() (*fsnotify.Watcher, *os.File, *bufio.Reader, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, nil, nil, eris.Wrap(err, "create file watcher")
	}

	// Watch the log directory for changes
	logPath := s.cfg.Log.FileConfig.Path
	s.l.Info().
		Str("watchPath", logPath).
		Msg("adding path to watcher")

	if err = watcher.Add(logPath); err != nil {
		watcher.Close()
		return nil, nil, nil, eris.Wrap(err, "add log directory to watcher")
	}

	// Open the file for reading
	fullPath := filepath.Join(s.cfg.Log.FileConfig.Path, s.cfg.Log.FileConfig.FileName)
	s.l.Info().
		Str("fullPath", fullPath).
		Msg("opening log file")

	file, err := os.Open(fullPath)
	if err != nil {
		watcher.Close()
		return nil, nil, nil, eris.Wrap(err, "open log file")
	}

	// Seek to the end of the file
	if _, err = file.Seek(0, io.SeekEnd); err != nil {
		file.Close()
		watcher.Close()
		return nil, nil, nil, eris.Wrap(err, "seek to end of file")
	}

	reader := bufio.NewReader(file)
	return watcher, file, reader, nil
}

// handleFileEvents processes events from the file watcher
func (s *Service) handleFileEvents(ctx context.Context, watcher *fsnotify.Watcher, file *os.File, reader *bufio.Reader) {
	defer func() {
		file.Close()
		watcher.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			s.l.Info().Msg("stopping file watcher due to context cancellation")
			return

		case event, ok := <-watcher.Events:
			if !ok {
				s.l.Info().Msg("watcher events channel closed")
				return
			}

			s.handleWatcherEvent(event, reader)

		case wErr, ok := <-watcher.Errors:
			if !ok {
				s.l.Info().Msg("watcher errors channel closed")
				return
			}
			if wErr != nil {
				s.l.Error().Err(wErr).Msg("file watcher error")
			}
		}
	}
}

// handleWatcherEvent processes a single file watcher event
func (s *Service) handleWatcherEvent(event fsnotify.Event, reader *bufio.Reader) {
	s.l.Trace().
		Str("event", event.String()).
		Str("operation", event.Op.String()).
		Str("path", event.Name).
		Msg("received file event")

	if event.Op&fsnotify.Write == fsnotify.Write {
		s.l.Trace().Msg("processing write event")
		s.processNewLogLines(reader)
	}
}

// processNewLogLines reads and processes new lines from the log file
func (s *Service) processNewLogLines(reader *bufio.Reader) {
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if eris.As(err, io.EOF) {
				// EOF is expected, not an error condition
				s.l.Trace().Msg("reached end of file")
				break
			}
			break
		}

		s.l.Trace().
			Str("line", string(line)).
			Msg("read new log line")

		entry, ok := s.parseLogLine(line)
		if !ok {
			continue
		}

		s.l.Trace().
			Interface("entry", entry).
			Msg("broadcasting log entry")

		// Broadcast directly to all clients
		s.BroadcastLogEntry(entry)
	}
}

// parseLogLine parses a log line into a LogEntry
func (s *Service) parseLogLine(line []byte) (*repositories.LogEntry, bool) {
	entry := new(repositories.LogEntry)
	if err := sonic.Unmarshal(line, entry); err != nil {
		s.l.Warn().
			Err(err).
			Str("line", string(line)).
			Interface("entry", entry).
			Msg("failed to unmarshal log entry, skipping")
		return nil, false
	}
	return entry, true
}

// checkFileRotation periodically checks if the log file has been rotated
func (s *Service) checkFileRotation(ctx context.Context, file *os.File, reader *bufio.Reader) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.handlePossibleRotation(file, reader)
		}
	}
}

// handlePossibleRotation checks and handles log file rotation
func (s *Service) handlePossibleRotation(file *os.File, reader *bufio.Reader) {
	logFilePath := filepath.Join(s.cfg.Log.FileConfig.Path, s.cfg.Log.FileConfig.FileName)

	if _, err := os.Stat(logFilePath); err != nil {
		if os.IsNotExist(err) {
			// File has been rotated, reopen it
			newFile, nErr := os.Open(logFilePath)
			if nErr != nil {
				s.l.Error().
					Err(nErr).
					Msg("failed to open rotated log file")
				return
			}

			// Get the current file descriptor for cleanup
			oldFile := *file

			// Replace the pointer values with the new file and reader
			*file = *newFile
			*reader = *bufio.NewReader(file)

			// Close the old file descriptor to prevent leaks
			oldFile.Close()

			s.l.Info().Msg("log file rotation detected, reopened new file")
		}
	}
}

func (s *Service) RegisterClient(lc *LogClient) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.l.Info().
		Str("userID", lc.UserID.String()).
		Msg("registering client")

	s.clients[lc] = true
	return nil
}

func (s *Service) UnregisterClient(lc *LogClient) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.l.Info().
		Str("userID", lc.UserID.String()).
		Msg("unregistering client")

	delete(s.clients, lc)
}

// byteCountIEC converts bytes to a human-readable string
func byteCountIEC(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
