package ediprofilesecret

import (
	"sort"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
)

func WithSecretState(
	profile *edi.EDICommunicationProfile,
) *edi.EDICommunicationProfile {
	if profile == nil {
		return nil
	}

	keys := make([]string, 0, len(profile.EncryptedSecrets))
	for key, value := range profile.EncryptedSecrets {
		if strings.TrimSpace(value) != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	profile.SecretState = make([]edi.CommunicationProfileSecretState, 0, len(keys))
	for _, key := range keys {
		profile.SecretState = append(
			profile.SecretState,
			edi.CommunicationProfileSecretState{Key: key, HasValue: true},
		)
	}
	profile.EncryptedSecrets = nil
	return profile
}
