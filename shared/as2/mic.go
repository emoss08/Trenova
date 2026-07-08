package as2

import (
	"crypto/sha1" //nolint:gosec // SHA-1 MICs remain required for AS2 interop with legacy partners.
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"hash"
	"strings"
)

const (
	MICAlgorithmSHA1   = "sha1"
	MICAlgorithmSHA256 = "sha256"
	MICAlgorithmSHA384 = "sha384"
	MICAlgorithmSHA512 = "sha512"
)

func ComputeMIC(content []byte, algorithm string) (string, error) {
	normalized := normalizeAlgorithm(algorithm, MICAlgorithmSHA256)
	digest, err := micHash(normalized)
	if err != nil {
		return "", err
	}
	digest.Write(content)
	return base64.StdEncoding.EncodeToString(digest.Sum(nil)) + ", " + normalized, nil
}

func MICDigest(mic string) string {
	digest, _, _ := strings.Cut(mic, ",")
	return strings.TrimSpace(digest)
}

func MICMatches(expected, actual string) bool {
	return MICDigest(expected) != "" && MICDigest(expected) == MICDigest(actual)
}

func micHash(algorithm string) (hash.Hash, error) {
	switch algorithm {
	case MICAlgorithmSHA1:
		return sha1.New(), nil //nolint:gosec // Legacy AS2 partners still negotiate SHA-1 MICs.
	case MICAlgorithmSHA256:
		return sha256.New(), nil
	case MICAlgorithmSHA384:
		return sha512.New384(), nil
	case MICAlgorithmSHA512:
		return sha512.New(), nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedSigningAlgorithm, algorithm)
	}
}
