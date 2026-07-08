package editransport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/maputils"
	"github.com/pkg/sftp"
)

const (
	configKeyInboundDir = "inboundDirectory"
	configKeyArchiveDir = "archiveDirectory"
)

func (t *SFTPTransport) FetchInboundFiles(
	ctx context.Context,
	req *services.EDIInboundFetchRequest,
) ([]*services.EDIInboundRemoteFile, error) {
	return fetchInboundOverSFTP(ctx, req)
}

func (t *SFTPTransport) ArchiveInboundFile(
	ctx context.Context,
	req *services.EDIInboundFetchRequest,
	remotePath string,
) error {
	return archiveInboundOverSFTP(ctx, req, remotePath)
}

func (t *VANTransport) FetchInboundFiles(
	ctx context.Context,
	req *services.EDIInboundFetchRequest,
) ([]*services.EDIInboundRemoteFile, error) {
	return fetchInboundOverSFTP(ctx, req)
}

func (t *VANTransport) ArchiveInboundFile(
	ctx context.Context,
	req *services.EDIInboundFetchRequest,
	remotePath string,
) error {
	return archiveInboundOverSFTP(ctx, req, remotePath)
}

func fetchInboundOverSFTP(
	ctx context.Context,
	req *services.EDIInboundFetchRequest,
) ([]*services.EDIInboundRemoteFile, error) {
	cfg, inboundDirectory, err := inboundEndpointConfig(req)
	if err != nil {
		return nil, err
	}
	client, sshClient, err := dialSFTP(ctx, cfg)
	if err != nil {
		return nil, err
	}
	defer sshClient.Close()
	defer client.Close()

	entries, err := client.ReadDir(inboundDirectory)
	if err != nil {
		return nil, fmt.Errorf("list inbound directory: %w", err)
	}
	files := make([]*services.EDIInboundRemoteFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		remotePath := path.Join(inboundDirectory, entry.Name())
		contents, readErr := readRemoteFile(client, remotePath)
		if readErr != nil {
			return nil, fmt.Errorf("read inbound file %s: %w", remotePath, readErr)
		}
		files = append(files, &services.EDIInboundRemoteFile{
			Path:     remotePath,
			Name:     entry.Name(),
			Contents: contents,
			Size:     entry.Size(),
		})
	}
	return files, nil
}

func archiveInboundOverSFTP(
	ctx context.Context,
	req *services.EDIInboundFetchRequest,
	remotePath string,
) error {
	cfg, inboundDirectory, err := inboundEndpointConfig(req)
	if err != nil {
		return err
	}
	client, sshClient, err := dialSFTP(ctx, cfg)
	if err != nil {
		return err
	}
	defer sshClient.Close()
	defer client.Close()

	archiveDirectory := stringOrDefault(
		maputils.StringValue(req.Profile.Config, configKeyArchiveDir),
		path.Join(inboundDirectory, "processed"),
	)
	if err = client.MkdirAll(archiveDirectory); err != nil {
		return fmt.Errorf("create archive directory: %w", err)
	}
	archivePath := path.Join(archiveDirectory, path.Base(remotePath))
	if err = client.PosixRename(remotePath, archivePath); err != nil {
		if renameErr := client.Rename(remotePath, archivePath); renameErr != nil {
			return fmt.Errorf("archive inbound file: %w", renameErr)
		}
	}
	return nil
}

func inboundEndpointConfig(
	req *services.EDIInboundFetchRequest,
) (*endpointConfig, string, error) {
	if req == nil || req.Profile == nil {
		return nil, "", errors.New("EDI communication profile is required for inbound polling")
	}
	inboundDirectory := maputils.StringValue(req.Profile.Config, configKeyInboundDir)
	if inboundDirectory == "" {
		return nil, "", errors.New(
			"inbound directory is required for EDI inbound polling",
		)
	}
	cfg := endpointConfigFromProfile(req.Profile, req.Secrets)
	if err := validateEndpointConfig(&cfg); err != nil {
		return nil, "", err
	}
	return &cfg, inboundDirectory, nil
}

func readRemoteFile(client *sftp.Client, remotePath string) (string, error) {
	file, err := client.Open(remotePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
