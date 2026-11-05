package editransport

import (
	"context"
	"errors"
	"fmt"
	"path"

	"github.com/emoss08/trenova/internal/modules/edi/ports/editransportports"
	"github.com/emoss08/trenova/shared/maputils"
	"github.com/emoss08/trenova/shared/sftp"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	configKeyInboundDir = "inboundDirectory"
	configKeyArchiveDir = "archiveDirectory"

	maxInboundFilesPerPoll = 200
	maxInboundBytesPerPoll = 128 << 20
)

type inboundFileReader interface {
	ReadFile(remotePath string) (string, error)
}

func (t *SFTPTransport) FetchInboundFiles(
	ctx context.Context,
	req *editransportports.EDIInboundFetchRequest,
) ([]*editransportports.EDIInboundRemoteFile, error) {
	return fetchInboundOverSFTP(ctx, t.network, req)
}

func (t *SFTPTransport) ArchiveInboundFile(
	ctx context.Context,
	req *editransportports.EDIInboundFetchRequest,
	remotePath string,
) error {
	return archiveInboundOverSFTP(ctx, t.network, req, remotePath)
}

func (t *VANTransport) FetchInboundFiles(
	ctx context.Context,
	req *editransportports.EDIInboundFetchRequest,
) ([]*editransportports.EDIInboundRemoteFile, error) {
	return fetchInboundOverSFTP(ctx, t.network, req)
}

func (t *VANTransport) ArchiveInboundFile(
	ctx context.Context,
	req *editransportports.EDIInboundFetchRequest,
	remotePath string,
) error {
	return archiveInboundOverSFTP(ctx, t.network, req, remotePath)
}

func fetchInboundOverSFTP(
	ctx context.Context,
	network sftpNetwork,
	req *editransportports.EDIInboundFetchRequest,
) ([]*editransportports.EDIInboundRemoteFile, error) {
	cfg, inboundDirectory, err := network.inboundEndpointConfig(req)
	if err != nil {
		return nil, err
	}
	client, err := sftp.Dial(ctx, cfg.Config)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	entries, err := client.ListFiles(inboundDirectory)
	if err != nil {
		return nil, err
	}
	return readInboundBatch(client, entries), nil
}

func readInboundBatch(
	reader inboundFileReader,
	entries []sftp.RemoteFile,
) []*editransportports.EDIInboundRemoteFile {
	files := make(
		[]*editransportports.EDIInboundRemoteFile,
		0,
		min(len(entries), maxInboundFilesPerPoll),
	)
	read, failed := 0, 0
	total := int64(0)
	for _, entry := range entries {
		if read >= maxInboundFilesPerPoll || failed >= maxInboundFilesPerPoll {
			break
		}
		if read > 0 && total+entry.Size > maxInboundBytesPerPoll {
			break
		}
		contents, err := reader.ReadFile(entry.Path)
		if err != nil {
			failed++
			files = append(files, &editransportports.EDIInboundRemoteFile{
				Path:      entry.Path,
				Name:      entry.Name,
				Size:      entry.Size,
				ReadError: fmt.Errorf("read inbound file %s: %w", entry.Path, err),
			})
			continue
		}
		read++
		total += int64(len(contents))
		files = append(files, &editransportports.EDIInboundRemoteFile{
			Path:     entry.Path,
			Name:     entry.Name,
			Contents: contents,
			Size:     entry.Size,
		})
	}
	return files
}

func archiveInboundOverSFTP(
	ctx context.Context,
	network sftpNetwork,
	req *editransportports.EDIInboundFetchRequest,
	remotePath string,
) error {
	cfg, inboundDirectory, err := network.inboundEndpointConfig(req)
	if err != nil {
		return err
	}
	client, err := sftp.Dial(ctx, cfg.Config)
	if err != nil {
		return err
	}
	defer client.Close()

	archiveDirectory := stringutils.WithDefault(
		maputils.StringValue(req.Profile.Config, configKeyArchiveDir),
		path.Join(inboundDirectory, "processed"),
	)
	return client.Archive(remotePath, archiveDirectory)
}

func (n sftpNetwork) inboundEndpointConfig(
	req *editransportports.EDIInboundFetchRequest,
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
	cfg := n.endpointConfig(req.Profile, req.Secrets)
	if err := cfg.Validate(); err != nil {
		return nil, "", err
	}
	return &cfg, inboundDirectory, nil
}
