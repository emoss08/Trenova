package as2

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
)

var (
	ErrNoPEMBlock         = errors.New("as2: no PEM block found")
	ErrNotCertificate     = errors.New("as2: PEM block is not a certificate")
	ErrUnsupportedKeyType = errors.New("as2: unsupported private key type")
)

func ParseCertificate(pemData []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, ErrNoPEMBlock
	}
	if block.Type != "CERTIFICATE" {
		return nil, ErrNotCertificate
	}
	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("as2: parse certificate: %w", err)
	}
	return certificate, nil
}

func ParsePrivateKey(pemData []byte) (crypto.PrivateKey, error) {
	for block, rest := pem.Decode(pemData); block != nil; block, rest = pem.Decode(rest) {
		switch block.Type {
		case "RSA PRIVATE KEY":
			key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("as2: parse PKCS#1 private key: %w", err)
			}
			return key, nil
		case "EC PRIVATE KEY":
			key, err := x509.ParseECPrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("as2: parse EC private key: %w", err)
			}
			return key, nil
		case "PRIVATE KEY":
			key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("as2: parse PKCS#8 private key: %w", err)
			}
			return key, nil
		}
	}
	return nil, ErrUnsupportedKeyType
}
