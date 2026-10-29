package editransport

import (
	"context"
	"errors"
	"path"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/config/ediconfig"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editransportports"
	"github.com/emoss08/trenova/shared/httpsafe"
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

type sftpNetwork struct {
	policy       httpsafe.Policy
	maxFileBytes int64
}

func sftpNetworkFromConfig(cfg *ediconfig.Config) sftpNetwork {
	return sftpNetwork{policy: cfg.EgressPolicy(), maxFileBytes: cfg.GetMaxInboundFileBytes()}
}

type SFTPTransport struct {
	network sftpNetwork
}

func NewSFTPTransport(cfg *ediconfig.Config) *SFTPTransport {
	return &SFTPTransport{network: sftpNetworkFromConfig(cfg)}
}

func (t *SFTPTransport) Method() edi.ConnectionMethod {
	return edi.ConnectionMethodSFTP
}

func (t *SFTPTransport) Deliver(
	ctx context.Context,
	req *editransportports.EDITransportRequest,
) (*editransportports.EDITransportResult, error) {
	return deliverOverSFTP(ctx, t.network, req, defaultOutboundDirectory)
}

func deliverOverSFTP(
	ctx context.Context,
	network sftpNetwork,
	req *editransportports.EDITransportRequest,
	fallbackOutboundDirectory string,
) (*editransportports.EDITransportResult, error) {
	if req == nil || req.Profile == nil {
		return nil, errors.New("EDI communication profile is required for delivery")
	}
	if strings.TrimSpace(req.FileName) == "" {
		return nil, errors.New("EDI delivery file name is required")
	}
	cfg := network.endpointConfig(req.Profile, req.Secrets)
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	directory := stringutils.WithDefault(cfg.outboundDirectory, fallbackOutboundDirectory)
	remotePath := path.Join(directory, req.FileName)
	if err := uploadFile(ctx, cfg.Config, remotePath, req.Contents); err != nil {
		return &editransportports.EDITransportResult{RemotePath: remotePath}, err
	}
	return &editransportports.EDITransportResult{RemotePath: remotePath}, nil
}

func (n sftpNetwork) endpointConfig(
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
			EgressPolicy: n.policy,
			MaxFileBytes: n.maxFileBytes,
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

func OutboundFileName(profile *edi.EDICommunicationProfile, message *edimessage.EDIMessage) string {
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
