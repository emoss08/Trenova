package logreader

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/ports"
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
func (s *Service) GetAvailableLogFiles(ctx context.Context, opts *ports.LimitOffsetQueryOptions) ([]string, error) {
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

	currentLogFile := filepath.Join(s.cfg.Log.FileConfig.Path, s.cfg.Log.FileConfig.FileName)

	file, err := os.Open(currentLogFile)
	if err != nil {
		if os.IsNotExist(err) {
			log.Info().Str("file", currentLogFile).Msg("log file does not exist")
			return []repositories.LogEntry{}, nil
		}
		return nil, eris.Wrap(err, "open log file")
	}
	defer file.Close()

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
			line := scanner.Bytes()
			var entry repositories.LogEntry
			if err := json.Unmarshal(line, &entry); err != nil {
				log.Warn().
					Err(err).
					Str("line", string(line)).
					Msg("failed to parse log entry, skipping")
				continue
			}

			// Apply time filter if specified
			if !opts.StartDate.IsZero() && entry.Time.Before(opts.StartDate) {
				continue
			}
			if !opts.EndDate.IsZero() && entry.Time.After(opts.EndDate) {
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

	// Sort entries by timestamp in descending order (newest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Time.After(entries[j].Time)
	})

	// Apply limit and offset if specified
	if opts.LimitOffsetQueryOptions != nil {
		start := opts.Offset
		end := opts.Offset + opts.Limit

		if start >= len(entries) {
			return []repositories.LogEntry{}, nil
		}
		if end > len(entries) {
			end = len(entries)
		}
		entries = entries[start:end]
	}

	return entries, nil
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
		info, err := os.Stat(file)
		if err != nil {
			log.Warn().
				Err(err).
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
		relPath, err := filepath.Rel(s.cfg.Log.FileConfig.Path, fi.path)
		if err != nil {
			log.Warn().
				Err(err).
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
			info, err := os.Stat(fullPath)
			if err != nil {
				log.Warn().
					Err(err).
					Str("file", fullPath).
					Msg("failed to stat file, skipping")
				continue
			}

			if info.ModTime().Before(cutoffTime) {
				if err := os.Remove(fullPath); err != nil {
					log.Error().
						Err(err).
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
func (s *Service) GetLogFileInfo(ctx context.Context, filename string) (*repositories.LogFileInfo, error) {
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

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return eris.Wrap(err, "create file watcher")
	}

	// Watch the log directory for changes
	logPath := s.cfg.Log.FileConfig.Path
	s.l.Info().
		Str("watchPath", logPath).
		Msg("adding path to watcher")

	if err := watcher.Add(logPath); err != nil {
		watcher.Close()
		return eris.Wrap(err, "add log directory to watcher")
	}

	// Open the file for reading
	fullPath := filepath.Join(s.cfg.Log.FileConfig.Path, s.cfg.Log.FileConfig.FileName)
	s.l.Info().
		Str("fullPath", fullPath).
		Msg("opening log file")

	file, err := os.Open(fullPath)
	if err != nil {
		watcher.Close()
		return eris.Wrap(err, "open log file")
	}
	// Seek to the end of the file
	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		file.Close()
		watcher.Close()
		return eris.Wrap(err, "seek to end of file")
	}

	reader := bufio.NewReader(file)

	go func() {
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

				s.l.Trace().
					Str("event", event.String()).
					Str("operation", event.Op.String()).
					Str("path", event.Name).
					Msg("received file event")

				if event.Op&fsnotify.Write == fsnotify.Write {
					s.l.Trace().Msg("processing write event")
					// Read new lines
					for {
						line, err := reader.ReadBytes('\n')
						if err != nil {
							if err == io.EOF {
								s.l.Trace().Msg("reached end of file")
								break
							}
							s.l.Error().Err(err).Msg("error reading log file")
							break
						}

						s.l.Trace().
							Str("line", string(line)).
							Msg("read new log line")

						entry := new(repositories.LogEntry)
						if err := sonic.Unmarshal(line, &entry); err != nil {
							s.l.Warn().
								Err(err).
								Str("line", string(line)).
								Interface("entry", entry).
								Msg("failed to unmarshal log entry, skipping")
							continue
						}

						s.l.Trace().
							Interface("entry", entry).
							Msg("broadcasting log entry")

						// Broadcast directly to all clients
						s.BroadcastLogEntry(entry)
					}
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					s.l.Info().Msg("watcher errors channel closed")
					return
				}
				if err != nil {
					s.l.Error().Err(err).Msg("file watcher error")
				}
			}
		}
	}()

	// Start file rotation check
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Check if the file has been rotated
				if _, err := os.Stat(filepath.Join(s.cfg.Log.FileConfig.Path, s.cfg.Log.FileConfig.FileName)); err != nil {
					if os.IsNotExist(err) {
						// File has been rotated, reopen it
						newFile, err := os.Open(filepath.Join(s.cfg.Log.FileConfig.Path, s.cfg.Log.FileConfig.FileName))
						if err != nil {
							s.l.Error().
								Err(err).
								Msg("failed to open rotated log file")
							continue
						}
						file.Close()
						file = newFile
						reader = bufio.NewReader(file)
					}
				}
			}
		}
	}()

	return nil
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
