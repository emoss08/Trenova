package ediprofilesecret

import (
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
)

func (s *Service) TransportSecrets(
	profile *edi.EDICommunicationProfile,
) (map[string]string, error) {
	secrets := make(map[string]string, 3)
	for _, key := range []string{"password", "privateKey", "basicAuthPassword"} {
		value, err := s.decryptProfileSecret(profile, key)
		if err != nil {
			return nil, err
		}
		if value != "" {
			secrets[key] = value
		}
	}
	return secrets, nil
}
