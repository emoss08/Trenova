package factoringtransport

import (
	"context"
	"errors"
	"net"
	"path"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringsubmissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/fileutils"
	"github.com/emoss08/trenova/shared/httpsafe"
	"github.com/emoss08/trenova/shared/sftp"
)

const maxRemoteFileName = 120

type Dialer func(ctx context.Context, cfg *sftp.Config) (RemoteClient, error)

type RemoteClient interface {
	WriteFile(remotePath string, contents []byte) error
	Stat(remotePath string) error
	Close() error
}

type Resolver func(ctx context.Context, host string) ([]net.IP, error)

type Sftp struct {
	dial    Dialer
	resolve Resolver
}

var _ factoringsubmissionports.FactoringTransport = (*Sftp)(nil)

func NewSftp() factoringsubmissionports.FactoringTransport {
	return NewSftpWith(dialSftp, resolveHost)
}

func NewSftpWith(dial Dialer, resolve Resolver) *Sftp {
	return &Sftp{dial: dial, resolve: resolve}
}

func (t *Sftp) Method() factoring.TransportMethod {
	return factoring.TransportMethodSftp
}

func (t *Sftp) Send(
	ctx context.Context,
	req *factoringsubmissionports.TransportSendRequest,
) (*factoringsubmissionports.TransportSendResult, error) {
	if err := checkSend(req); err != nil {
		return nil, err
	}
	if len(req.Files) == 0 {
		return nil, errortypes.NewBusinessError("The schedule has no files to send")
	}
	client, err := t.connect(ctx, req.Transport, req.Secret)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	directory := RemoteDirectory(req.Transport, req.Submission)
	for _, file := range req.Files {
		if err = ctx.Err(); err != nil {
			return nil, err
		}
		remotePath := path.Join(directory, RemoteName(file.Name))
		if err = client.WriteFile(remotePath, file.Content); err != nil {
			return nil, err
		}
	}

	return &factoringsubmissionports.TransportSendResult{Reference: directory, Parts: 1}, nil
}

func (t *Sftp) Test(ctx context.Context, req *factoringsubmissionports.TransportTestRequest) error {
	if req == nil || req.Transport == nil {
		return errortypes.NewBusinessError("There is no delivery setting to test")
	}
	client, err := t.connect(ctx, req.Transport, req.Secret)
	if err != nil {
		return err
	}
	defer client.Close()

	if err = client.Stat(req.Transport.SftpDirectory); err != nil {
		return errortypes.NewBusinessError(
			"Connected, but the directory {0} could not be read: {1}",
			req.Transport.SftpDirectory,
			err.Error(),
		)
	}
	return nil
}

func (t *Sftp) connect(
	ctx context.Context,
	transport *factoring.Transport,
	secret string,
) (RemoteClient, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, errortypes.NewBusinessError(
			"The SFTP password or private key is missing; enter it again on the factor",
		)
	}
	address, err := t.publicAddress(ctx, transport.SftpHost)
	if err != nil {
		return nil, err
	}
	cfg := sftp.Config{
		Host:         address,
		Port:         strconv.Itoa(transport.SftpPort),
		Username:     transport.SftpUsername,
		KnownHostKey: transport.SftpKnownHostKey,
	}
	if transport.SftpAuthMethod == factoring.SftpAuthPassword {
		cfg.AuthMode = sftp.AuthModePassword
		cfg.Password = secret
	} else {
		cfg.PrivateKey = secret
	}
	if err = cfg.Validate(); err != nil {
		return nil, errortypes.NewBusinessError("{0}", err.Error())
	}
	if _, err = sftp.HostKeyCallbackFor(cfg.KnownHostKey); err != nil {
		return nil, errortypes.NewBusinessError(
			"The SFTP host key could not be read: {0}",
			err.Error(),
		)
	}
	return t.dial(ctx, &cfg)
}

func (t *Sftp) publicAddress(ctx context.Context, host string) (string, error) {
	ips, err := t.resolve(ctx, host)
	if err != nil {
		return "", errortypes.NewBusinessError("The SFTP host {0} could not be found", host)
	}
	if len(ips) == 0 {
		return "", errortypes.NewBusinessError("The SFTP host {0} could not be found", host)
	}
	for _, ip := range ips {
		if httpsafe.IsBlockedIP(ip) {
			return "", errortypes.NewBusinessError(
				"The SFTP host {0} is on a private or reserved network; use the factor's public address",
				host,
			)
		}
	}
	return ips[0].String(), nil
}

func RemoteDirectory(transport *factoring.Transport, submission *factoring.Submission) string {
	base := transport.SftpDirectory
	if base == "" {
		base = factoring.DefaultSftpDirectory
	}
	return path.Join(base, RemoteName(submission.Number))
}

func RemoteName(name string) string {
	cleaned := fileutils.SanitizeDisplayFilename(name, "file", maxRemoteFileName)
	cleaned = strings.TrimLeft(cleaned, ".")
	if cleaned == "" {
		return "file"
	}
	return cleaned
}

func resolveHost(ctx context.Context, host string) ([]net.IP, error) {
	if ip := net.ParseIP(host); ip != nil {
		return []net.IP{ip}, nil
	}
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	ips := make([]net.IP, 0, len(addrs))
	for _, addr := range addrs {
		ips = append(ips, addr.IP)
	}
	return ips, nil
}

type sftpClient struct {
	client *sftp.Client
}

func dialSftp(ctx context.Context, cfg *sftp.Config) (RemoteClient, error) {
	client, err := sftp.Dial(ctx, *cfg)
	if err != nil {
		return nil, err
	}
	return &sftpClient{client: client}, nil
}

func (c *sftpClient) WriteFile(remotePath string, contents []byte) error {
	return c.client.WriteFile(remotePath, contents)
}

func (c *sftpClient) Stat(remotePath string) error {
	info, err := c.client.Stat(remotePath)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("not a directory")
	}
	return nil
}

func (c *sftpClient) Close() error {
	return c.client.Close()
}
