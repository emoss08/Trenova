package ediservice

import (
	"github.com/emoss08/trenova/internal/modules/edi/app/editransport"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/maputils"
)

const (
	configKeyKnownHostKey = "knownHostKey"
	secretKeyPassword     = "password"
	secretKeyPrivateKey   = "privateKey"
)

type credentialSpec struct {
	keys     []string
	label    string
	kind     ediports.CredentialKind
	required func(profile *edi.EDICommunicationProfile) bool
	message  string
}

func always(*edi.EDICommunicationProfile) bool { return true }

func never(*edi.EDICommunicationProfile) bool { return false }

func configSet(key string) func(*edi.EDICommunicationProfile) bool {
	return func(profile *edi.EDICommunicationProfile) bool {
		return maputils.StringValue(profile.Config, key) != ""
	}
}

func fileTransferCredentials(method string) []credentialSpec {
	return []credentialSpec{
		{
			keys:     []string{configKeyKnownHostKey},
			label:    "Known host key",
			kind:     ediports.CredentialKindSetting,
			required: always,
			message:  "Known host key is required",
		},
		{
			keys:     []string{secretKeyPassword, secretKeyPrivateKey},
			label:    method + " password or private key",
			kind:     ediports.CredentialKindSecret,
			required: always,
			message:  method + " password or private key secret is required",
		},
	}
}

var as2Credentials = []credentialSpec{
	{
		keys:     []string{editransport.ConfigKeyPartnerSigningCertificate},
		label:    "Partner signing certificate",
		kind:     ediports.CredentialKindSetting,
		required: never,
	},
	{
		keys:     []string{editransport.ConfigKeyPartnerEncryptionCertificate},
		label:    "Partner encryption certificate",
		kind:     ediports.CredentialKindSetting,
		required: never,
	},
	{
		keys:     []string{editransport.ConfigKeyLocalCertificate},
		label:    "Your signing certificate",
		kind:     ediports.CredentialKindSetting,
		required: never,
	},
	{
		keys:     []string{editransport.SecretKeyAS2PrivateKey},
		label:    "AS2 private key for your signing certificate",
		kind:     ediports.CredentialKindSecret,
		required: configSet(editransport.ConfigKeyLocalCertificate),
		message:  "AS2 private key secret is required when a local certificate is configured",
	},
	{
		keys:     []string{editransport.SecretKeyBasicAuthPassword},
		label:    "Basic auth password",
		kind:     ediports.CredentialKindSecret,
		required: never,
	},
}

func credentialSpecs(method edi.ConnectionMethod) []credentialSpec {
	switch method {
	case edi.ConnectionMethodSFTP:
		return fileTransferCredentials("SFTP")
	case edi.ConnectionMethodVAN:
		return fileTransferCredentials("VAN")
	case edi.ConnectionMethodAS2:
		return as2Credentials
	default:
		return nil
	}
}

func IsCredentialKey(method edi.ConnectionMethod, key string) bool {
	for _, spec := range credentialSpecs(method) {
		for _, candidate := range spec.keys {
			if candidate == key {
				return true
			}
		}
	}

	return false
}

func CredentialRequirements(
	profile *edi.EDICommunicationProfile,
) []ediports.CredentialRequirement {
	specs := credentialSpecs(profile.Method)
	requirements := make([]ediports.CredentialRequirement, 0, len(specs))
	for _, spec := range specs {
		requirements = append(requirements, ediports.CredentialRequirement{
			Keys:     spec.keys,
			Label:    spec.label,
			Kind:     spec.kind,
			Required: spec.required(profile),
			Missing:  !credentialPresent(profile, spec),
		})
	}

	return requirements
}

func credentialPresent(profile *edi.EDICommunicationProfile, spec credentialSpec) bool {
	for _, key := range spec.keys {
		if spec.kind == ediports.CredentialKindSecret {
			if profile.EncryptedSecrets[key] != "" {
				return true
			}
			continue
		}
		if maputils.StringValue(profile.Config, key) != "" {
			return true
		}
	}

	return false
}

func requireCredentials(multiErr *errortypes.MultiError, profile *edi.EDICommunicationProfile) {
	if profile.Status == domaintypes.StatusInactive {
		return
	}
	for _, spec := range credentialSpecs(profile.Method) {
		if !spec.required(profile) || credentialPresent(profile, spec) {
			continue
		}
		field := "secrets"
		if spec.kind == ediports.CredentialKindSetting {
			field = "config." + spec.keys[0]
		}
		multiErr.Add(field, errortypes.ErrRequired, spec.message)
	}
}
