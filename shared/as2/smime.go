package as2

import (
	"crypto"
	"crypto/x509"
	"encoding/asn1"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/smallstep/pkcs7"
)

const (
	SigningAlgorithmSHA1   = "sha1"
	SigningAlgorithmSHA256 = "sha256"
	SigningAlgorithmSHA384 = "sha384"
	SigningAlgorithmSHA512 = "sha512"

	EncryptionAlgorithmTripleDES = "3des"
	EncryptionAlgorithmAES128CBC = "aes128-cbc"
	EncryptionAlgorithmAES192CBC = "aes192-cbc"
	EncryptionAlgorithmAES256CBC = "aes256-cbc"
	EncryptionAlgorithmAES128GCM = "aes128-gcm"
	EncryptionAlgorithmAES256GCM = "aes256-gcm"
)

var (
	ErrUnsupportedSigningAlgorithm    = errors.New("as2: unsupported signing algorithm")
	ErrUnsupportedEncryptionAlgorithm = errors.New("as2: unsupported encryption algorithm")
)

func Sign(
	content []byte,
	certificate *x509.Certificate,
	key crypto.PrivateKey,
	algorithm string,
) ([]byte, error) {
	signedData, err := pkcs7.NewSignedData(content)
	if err != nil {
		return nil, fmt.Errorf("as2: build signed data: %w", err)
	}
	digest, err := signingDigestOID(algorithm)
	if err != nil {
		return nil, err
	}
	signedData.SetDigestAlgorithm(digest)
	if err = signedData.AddSigner(certificate, key, pkcs7.SignerInfoConfig{}); err != nil {
		return nil, fmt.Errorf("as2: add signer: %w", err)
	}
	signedData.Detach()
	signature, err := signedData.Finish()
	if err != nil {
		return nil, fmt.Errorf("as2: finish signature: %w", err)
	}
	return signature, nil
}

func Verify(content, signature []byte, certificate *x509.Certificate) error {
	parsed, err := pkcs7.Parse(signature)
	if err != nil {
		return fmt.Errorf("as2: parse signature: %w", err)
	}
	parsed.Content = content
	if err = parsed.Verify(); err != nil {
		return fmt.Errorf("as2: verify signature: %w", err)
	}
	if certificate != nil {
		signer := parsed.GetOnlySigner()
		if signer == nil {
			return errors.New("as2: signature does not carry a signer certificate")
		}
		if !signer.Equal(certificate) {
			return errors.New("as2: signature was not produced by the expected certificate")
		}
	}
	return nil
}

var encryptMutex sync.Mutex

func Encrypt(content []byte, certificate *x509.Certificate, algorithm string) ([]byte, error) {
	encryption, err := encryptionAlgorithmID(algorithm)
	if err != nil {
		return nil, err
	}
	encryptMutex.Lock()
	defer encryptMutex.Unlock()
	pkcs7.ContentEncryptionAlgorithm = encryption
	ciphertext, err := pkcs7.Encrypt(content, []*x509.Certificate{certificate})
	if err != nil {
		return nil, fmt.Errorf("as2: encrypt content: %w", err)
	}
	return ciphertext, nil
}

func Decrypt(
	ciphertext []byte,
	certificate *x509.Certificate,
	key crypto.PrivateKey,
) ([]byte, error) {
	parsed, err := pkcs7.Parse(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("as2: parse enveloped data: %w", err)
	}
	content, err := parsed.Decrypt(certificate, key)
	if err != nil {
		return nil, fmt.Errorf("as2: decrypt content: %w", err)
	}
	return content, nil
}

func VerifySignedData(der []byte, certificate *x509.Certificate) ([]byte, error) {
	parsed, err := pkcs7.Parse(der)
	if err != nil {
		return nil, fmt.Errorf("as2: parse signed data: %w", err)
	}
	if err = parsed.Verify(); err != nil {
		return nil, fmt.Errorf("as2: verify signed data: %w", err)
	}
	if certificate != nil {
		signer := parsed.GetOnlySigner()
		if signer == nil {
			return nil, errors.New("as2: signed data does not carry a signer certificate")
		}
		if !signer.Equal(certificate) {
			return nil, errors.New("as2: signed data was not produced by the expected certificate")
		}
	}
	return parsed.Content, nil
}

func normalizeAlgorithm(algorithm, fallback string) string {
	normalized := strings.ToLower(strings.TrimSpace(algorithm))
	if normalized == "" {
		return fallback
	}
	return normalized
}

func signingDigestOID(algorithm string) (asn1.ObjectIdentifier, error) {
	switch normalizeAlgorithm(algorithm, SigningAlgorithmSHA256) {
	case SigningAlgorithmSHA1:
		return pkcs7.OIDDigestAlgorithmSHA1, nil
	case SigningAlgorithmSHA256:
		return pkcs7.OIDDigestAlgorithmSHA256, nil
	case SigningAlgorithmSHA384:
		return pkcs7.OIDDigestAlgorithmSHA384, nil
	case SigningAlgorithmSHA512:
		return pkcs7.OIDDigestAlgorithmSHA512, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedSigningAlgorithm, algorithm)
	}
}

func encryptionAlgorithmID(algorithm string) (int, error) {
	switch normalizeAlgorithm(algorithm, EncryptionAlgorithmAES256CBC) {
	case EncryptionAlgorithmTripleDES:
		return pkcs7.EncryptionAlgorithmDESCBC, nil
	case EncryptionAlgorithmAES128CBC:
		return pkcs7.EncryptionAlgorithmAES128CBC, nil
	case EncryptionAlgorithmAES192CBC, EncryptionAlgorithmAES256CBC:
		return pkcs7.EncryptionAlgorithmAES256CBC, nil
	case EncryptionAlgorithmAES128GCM:
		return pkcs7.EncryptionAlgorithmAES128GCM, nil
	case EncryptionAlgorithmAES256GCM:
		return pkcs7.EncryptionAlgorithmAES256GCM, nil
	default:
		return 0, fmt.Errorf("%w: %s", ErrUnsupportedEncryptionAlgorithm, algorithm)
	}
}
