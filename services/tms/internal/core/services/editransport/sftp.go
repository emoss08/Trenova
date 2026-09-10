package editransport

import (
	"context"
	"errors"
	"path"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/maputils"
	"github.com/emoss08/trenova/shared/sftp"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	AuthModePassword = sftp.AuthModePassword

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
	sftp.Config

	outboundDirectory string
	fileNamingPattern string
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
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	directory := stringutils.WithDefault(cfg.outboundDirectory, fallbackOutboundDirectory)
	remotePath := path.Join(directory, req.FileName)
	if err := uploadFile(ctx, cfg.Config, remotePath, req.Contents); err != nil {
		return &services.EDITransportResult{RemotePath: remotePath}, err
	}
	return &services.EDITransportResult{RemotePath: remotePath}, nil
}

func endpointConfigFromProfile(
	profile *edi.EDICommunicationProfile,
	secrets map[string]string,
) endpointConfig {
	return endpointConfig{
		Config: sftp.Config{
			Host:         maputils.StringValue(profile.Config, configKeyHost),
			Port:         maputils.StringValue(profile.Config, configKeyPort),
			Username:     maputils.StringValue(profile.Config, configKeyUsername),
			AuthMode:     maputils.StringValue(profile.Config, configKeyAuthMode),
			KnownHostKey: maputils.StringValue(profile.Config, configKeyKnownHostKey),
			Password:     strings.TrimSpace(secrets[secretKeyPassword]),
			PrivateKey:   strings.TrimSpace(secrets[secretKeyPrivateKey]),
		},
		outboundDirectory: maputils.StringValue(profile.Config, configKeyOutboundDir),
		fileNamingPattern: maputils.StringValue(profile.Config, configKeyFileNamePattern),
	}
}

func uploadFile(
	ctx context.Context,
	cfg sftp.Config,
	remotePath, contents string,
) error {
	client, err := sftp.Dial(ctx, cfg)
	if err != nil {
		return err
	}
	defer client.Close()

	return client.WriteFile(remotePath, []byte(contents))
}

func OutboundFileName(profile *edi.EDICommunicationProfile, message *edi.EDIMessage) string {
	pattern := defaultFileNamingPattern
	if profile != nil {
		pattern = stringutils.WithDefault(
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
