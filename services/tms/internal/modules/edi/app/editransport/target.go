package editransport

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/shared/maputils"
)

func TargetChanged(before, after map[string]any) bool {
	for _, key := range []string{
		ConfigKeyEndpointURL,
		configKeyHost,
		configKeyPort,
		configKeyKnownHostKey,
	} {
		if strings.TrimRight(maputils.StringValue(before, key), "/") !=
			strings.TrimRight(maputils.StringValue(after, key), "/") {
			return true
		}
	}

	return false
}

func TargetBoundSecretKeys() []string {
	return []string{secretKeyPassword, secretKeyPrivateKey, SecretKeyBasicAuthPassword}
}

func SecretInUse(method edi.ConnectionMethod, config map[string]any, key string) bool {
	switch method {
	case edi.ConnectionMethodAS2:
		return key == SecretKeyAS2PrivateKey || key == SecretKeyBasicAuthPassword
	case edi.ConnectionMethodSFTP, edi.ConnectionMethodVAN:
		if maputils.StringValue(config, configKeyAuthMode) == AuthModePassword {
			return key == secretKeyPassword
		}
		return key == secretKeyPrivateKey
	case edi.ConnectionMethodInternal:
		return false
	default:
		return false
	}
}
