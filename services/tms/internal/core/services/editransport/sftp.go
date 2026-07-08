package editransport

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/maputils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const (
	AuthModePassword = "password"

	sshDialTimeout           = 30 * time.Second
	defaultSFTPPort          = "22"
	defaultOutboundDirectory = "/outbound"
	defaultFileNamingPattern = "{partnerId}-{transactionSet}-{messageId}.x12"
	defaultOutboundExtension = ".x12"
	secretKeyPassword        = "password"
	secretKeyPrivateKey      = "privateKey"
	configKeyHost            = "host"
	configKeyPort            = "port"
	configKeyUsername        = "username"
	configKeyAuthMode        = "authMode"
	configKeyKnownHostKey    = "knownHostKey"
	configKeyOutboundDir     = "outboundDirectory"
	configKeyFileNamePattern = "fileNamingPattern"
	configKeyVANMailboxID    = "mailboxId"
)

type endpointConfig struct {
	host              string
	port              string
	username          string
	authMode          string
	knownHostKey      string
	outboundDirectory string
	fileNamingPattern string
	password          string
	privateKey        string
}

type SFTPTransport struct{}

func NewSFTPTransport() *SFTPTransport {
	return &SFTPTransport{}
}

func (t *SFTPTransport) Method() edi.ConnectionMethod {
	return edi.ConnectionMethodSFTP
}

func (t *SFTPTransport) Deliver(
	ctx context.Context,
	req *services.EDITransportRequest,
) (*services.EDITransportResult, error) {
	return deliverOverSFTP(ctx, req, defaultOutboundDirectory)
}

func deliverOverSFTP(
	ctx context.Context,
	req *services.EDITransportRequest,
	fallbackOutboundDirectory string,
) (*services.EDITransportResult, error) {
	if req == nil || req.Profile == nil {
		return nil, errors.New("EDI communication profile is required for delivery")
	}
	if strings.TrimSpace(req.FileName) == "" {
		return nil, errors.New("EDI delivery file name is required")
	}
	cfg := endpointConfigFromProfile(req.Profile, req.Secrets)
	if err := validateEndpointConfig(&cfg); err != nil {
		return nil, err
	}
	directory := stringOrDefault(cfg.outboundDirectory, fallbackOutboundDirectory)
	remotePath := path.Join(directory, req.FileName)
	if err := uploadFile(ctx, &cfg, remotePath, req.Contents); err != nil {
		return &services.EDITransportResult{RemotePath: remotePath}, err
	}
	return &services.EDITransportResult{RemotePath: remotePath}, nil
}

func endpointConfigFromProfile(
	profile *edi.EDICommunicationProfile,
	secrets map[string]string,
) endpointConfig {
	return endpointConfig{
		host:              maputils.StringValue(profile.Config, configKeyHost),
		port:              maputils.StringValue(profile.Config, configKeyPort),
		username:          maputils.StringValue(profile.Config, configKeyUsername),
		authMode:          maputils.StringValue(profile.Config, configKeyAuthMode),
		knownHostKey:      maputils.StringValue(profile.Config, configKeyKnownHostKey),
		outboundDirectory: maputils.StringValue(profile.Config, configKeyOutboundDir),
		fileNamingPattern: maputils.StringValue(profile.Config, configKeyFileNamePattern),
		password:          strings.TrimSpace(secrets[secretKeyPassword]),
		privateKey:        strings.TrimSpace(secrets[secretKeyPrivateKey]),
	}
}

func validateEndpointConfig(cfg *endpointConfig) error {
	switch {
	case cfg.host == "":
		return errors.New("SFTP host is required for EDI delivery")
	case cfg.username == "":
		return errors.New("SFTP username is required for EDI delivery")
	case cfg.knownHostKey == "":
		return errors.New("SFTP known host key is required for EDI delivery")
	case cfg.authMode == AuthModePassword && cfg.password == "":
		return errors.New("SFTP password secret is required for EDI delivery")
	case cfg.authMode != AuthModePassword && cfg.privateKey == "":
		return errors.New("SFTP private key secret is required for EDI delivery")
	}
	if cfg.port == "" {
		return nil
	}
	if _, err := strconv.Atoi(cfg.port); err != nil {
		return fmt.Errorf("SFTP port must be numeric: %w", err)
	}
	return nil
}

func uploadFile(ctx context.Context, cfg *endpointConfig, remotePath, contents string) error {
	client, sshClient, err := dialSFTP(ctx, cfg)
	if err != nil {
		return err
	}
	defer sshClient.Close()
	defer client.Close()

	if err = client.MkdirAll(path.Dir(remotePath)); err != nil {
		return fmt.Errorf("create remote directory: %w", err)
	}
	file, err := client.Create(remotePath)
	if err != nil {
		return fmt.Errorf("create remote file: %w", err)
	}
	defer file.Close()
	if _, err = file.Write([]byte(contents)); err != nil {
		return fmt.Errorf("write remote file: %w", err)
	}
	return nil
}

func dialSFTP(
	ctx context.Context,
	cfg *endpointConfig,
) (sftpClient *sftp.Client, sshClient *ssh.Client, err error) {
	authMethod, err := authMethodForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}
	hostKeyCallback, err := hostKeyCallbackFor(cfg.knownHostKey)
	if err != nil {
		return nil, nil, err
	}
	address := net.JoinHostPort(cfg.host, stringOrDefault(cfg.port, defaultSFTPPort))
	clientConfig := &ssh.ClientConfig{
		User:            cfg.username,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: hostKeyCallback,
		Timeout:         sshDialTimeout,
	}
	dialer := net.Dialer{Timeout: sshDialTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, nil, fmt.Errorf("connect SFTP server: %w", err)
	}
	sshConn, channels, requests, err := ssh.NewClientConn(conn, address, clientConfig)
	if err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("establish SSH connection: %w", err)
	}
	sshClient = ssh.NewClient(sshConn, channels, requests)
	sftpClient, err = sftp.NewClient(sshClient)
	if err != nil {
		_ = sshClient.Close()
		return nil, nil, fmt.Errorf("open SFTP session: %w", err)
	}
	return sftpClient, sshClient, nil
}

func authMethodForConfig(cfg *endpointConfig) (ssh.AuthMethod, error) {
	if cfg.authMode == AuthModePassword {
		return ssh.Password(cfg.password), nil
	}
	signer, err := ssh.ParsePrivateKey([]byte(cfg.privateKey))
	if err != nil {
		return nil, fmt.Errorf("parse SFTP private key: %w", err)
	}
	return ssh.PublicKeys(signer), nil
}

func hostKeyCallbackFor(knownHostKey string) (ssh.HostKeyCallback, error) {
	publicKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(strings.TrimSpace(knownHostKey)))
	if err != nil {
		fields := strings.Fields(knownHostKey)
		if len(fields) >= 3 {
			publicKey, _, _, _, err = ssh.ParseAuthorizedKey(
				[]byte(strings.Join(fields[1:], " ")),
			)
		}
		if err != nil {
			return nil, fmt.Errorf("parse SFTP known host key: %w", err)
		}
	}
	return func(_ string, _ net.Addr, key ssh.PublicKey) error {
		if !bytes.Equal(key.Marshal(), publicKey.Marshal()) {
			return errors.New("SFTP host key does not match configured known host key")
		}
		return nil
	}, nil
}

func OutboundFileName(profile *edi.EDICommunicationProfile, message *edi.EDIMessage) string {
	pattern := defaultFileNamingPattern
	if profile != nil {
		pattern = stringOrDefault(
			maputils.StringValue(profile.Config, configKeyFileNamePattern),
			defaultFileNamingPattern,
		)
	}
	partnerID := ""
	transactionSet := ""
	messageID := ""
	if message != nil {
		if message.EDIPartnerID.IsNotNil() {
			partnerID = message.EDIPartnerID.String()
		}
		transactionSet = string(message.TransactionSet)
		messageID = message.ID.String()
	}
	replacer := strings.NewReplacer(
		"{partner}", partnerID,
		"{partnerId}", partnerID,
		"{transactionSet}", transactionSet,
		"{messageId}", messageID,
		"{timestamp}", strconv.FormatInt(timeutils.NowUnix(), 10),
	)
	name := replacer.Replace(pattern)
	name = strings.NewReplacer("/", "_", "\\", "_", " ", "_").Replace(name)
	if strings.TrimSpace(name) == "" {
		return messageID + defaultOutboundExtension
	}
	if path.Ext(name) == "" {
		name += defaultOutboundExtension
	}
	return name
}

func stringOrDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
