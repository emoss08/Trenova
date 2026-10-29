package smime

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/des" //nolint:gosec // DES-EDE3-CBC is the AS2 '3des' partner setting
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
	"fmt"
	"math/big"
	"math/bits"
)

var (
	oidEnvelopedData    = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 3}
	oidRSAEncryption    = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 1}
	oidDESCBC           = asn1.ObjectIdentifier{1, 3, 14, 3, 2, 7}
	oidDESEDE3CBC       = asn1.ObjectIdentifier{1, 2, 840, 113549, 3, 7}
	oidAES128CBC        = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 1, 2}
	oidAES256CBC        = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 1, 42}
	oidAES128GCM        = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 1, 6}
	oidAES256GCM        = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 1, 46}
	gcmNonceSize        = 12
	errRecipientKeyType = errors.New("as2: encryption certificate must carry an RSA public key")
)

var ErrInsecureEncryptionAlgorithm = errors.New(
	"as2: content is encrypted with single DES, which is refused",
)

type contentCipherMode int

const (
	contentCipherCBC contentCipherMode = iota
	contentCipherGCM
)

type contentCipher struct {
	oid      asn1.ObjectIdentifier
	keySize  int
	mode     contentCipherMode
	newBlock func(key []byte) (cipher.Block, error)
}

var contentCiphers = map[string]contentCipher{
	EncryptionAlgorithmTripleDES: {
		oid:      oidDESEDE3CBC,
		keySize:  24,
		mode:     contentCipherCBC,
		newBlock: des.NewTripleDESCipher,
	},
	EncryptionAlgorithmAES128CBC: {
		oid:      oidAES128CBC,
		keySize:  16,
		mode:     contentCipherCBC,
		newBlock: aes.NewCipher,
	},
	EncryptionAlgorithmAES256CBC: {
		oid:      oidAES256CBC,
		keySize:  32,
		mode:     contentCipherCBC,
		newBlock: aes.NewCipher,
	},
	EncryptionAlgorithmAES128GCM: {
		oid:      oidAES128GCM,
		keySize:  16,
		mode:     contentCipherGCM,
		newBlock: aes.NewCipher,
	},
	EncryptionAlgorithmAES256GCM: {
		oid:      oidAES256GCM,
		keySize:  32,
		mode:     contentCipherGCM,
		newBlock: aes.NewCipher,
	},
}

type envelopeContentInfo struct {
	ContentType asn1.ObjectIdentifier
	Content     asn1.RawValue
}

type envelopedData struct {
	Version              int
	RecipientInfos       []keyTransRecipientInfo `asn1:"set"`
	EncryptedContentInfo encryptedContentInfo
}

type keyTransRecipientInfo struct {
	Version                int
	IssuerAndSerialNumber  issuerAndSerialNumber
	KeyEncryptionAlgorithm pkix.AlgorithmIdentifier
	EncryptedKey           []byte
}

type issuerAndSerialNumber struct {
	Issuer       asn1.RawValue
	SerialNumber *big.Int
}

type encryptedContentInfo struct {
	ContentType                asn1.ObjectIdentifier
	ContentEncryptionAlgorithm pkix.AlgorithmIdentifier
	EncryptedContent           asn1.RawValue `asn1:"tag:0,optional"`
}

type gcmParameters struct {
	Nonce  []byte `asn1:"tag:4"`
	ICVLen int
}

func lookupContentCipher(algorithm string) (contentCipher, error) {
	normalized := NormalizeAlgorithm(algorithm, EncryptionAlgorithmAES256CBC)
	selected, ok := contentCiphers[normalized]
	if !ok {
		return contentCipher{}, fmt.Errorf("%w: %s", ErrUnsupportedEncryptionAlgorithm, algorithm)
	}
	return selected, nil
}

func encryptEnvelope(
	content []byte,
	recipient *x509.Certificate,
	selected contentCipher,
) ([]byte, error) {
	if recipient == nil {
		return nil, errors.New("as2: encryption certificate is required")
	}
	publicKey, ok := recipient.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, errRecipientKeyType
	}

	key := make([]byte, selected.keySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("as2: generate content key: %w", err)
	}
	if selected.mode == contentCipherCBC && selected.keySize == 24 {
		setOddParity(key)
	}

	eci, err := sealContent(content, key, selected)
	if err != nil {
		return nil, err
	}

	encryptedKey, err := rsa.EncryptPKCS1v15( //nolint:staticcheck // AS2 partners expect rsaEncryption key transport
		rand.Reader,
		publicKey,
		key,
	)
	if err != nil {
		return nil, fmt.Errorf("as2: wrap content key: %w", err)
	}

	envelope := envelopedData{
		Version: 0,
		RecipientInfos: []keyTransRecipientInfo{{
			Version: 0,
			IssuerAndSerialNumber: issuerAndSerialNumber{
				Issuer:       asn1.RawValue{FullBytes: recipient.RawIssuer},
				SerialNumber: recipient.SerialNumber,
			},
			KeyEncryptionAlgorithm: pkix.AlgorithmIdentifier{
				Algorithm:  oidRSAEncryption,
				Parameters: asn1.NullRawValue,
			},
			EncryptedKey: encryptedKey,
		}},
		EncryptedContentInfo: eci,
	}
	inner, err := asn1.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("as2: encode enveloped data: %w", err)
	}
	return asn1.Marshal(envelopeContentInfo{
		ContentType: oidEnvelopedData,
		Content: asn1.RawValue{
			Class:      asn1.ClassContextSpecific,
			Tag:        0,
			IsCompound: true,
			Bytes:      inner,
		},
	})
}

func sealContent(
	content, key []byte,
	selected contentCipher,
) (encryptedContentInfo, error) {
	block, err := selected.newBlock(key)
	if err != nil {
		return encryptedContentInfo{}, fmt.Errorf("as2: build content cipher: %w", err)
	}
	if selected.mode == contentCipherGCM {
		return sealGCM(content, block, selected.oid)
	}
	return sealCBC(content, block, selected.oid)
}

func sealCBC(
	content []byte,
	block cipher.Block,
	oid asn1.ObjectIdentifier,
) (encryptedContentInfo, error) {
	iv := make([]byte, block.BlockSize())
	if _, err := rand.Read(iv); err != nil {
		return encryptedContentInfo{}, fmt.Errorf("as2: generate IV: %w", err)
	}
	padded := padPKCS7(content, block.BlockSize())
	ciphertext := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, padded)
	ivParameter, err := asn1.Marshal(iv)
	if err != nil {
		return encryptedContentInfo{}, fmt.Errorf("as2: encode IV: %w", err)
	}
	return encryptedContentInfo{
		ContentType: oidData,
		ContentEncryptionAlgorithm: pkix.AlgorithmIdentifier{
			Algorithm:  oid,
			Parameters: asn1.RawValue{FullBytes: ivParameter},
		},
		EncryptedContent: asn1.RawValue{
			Class: asn1.ClassContextSpecific,
			Tag:   0,
			Bytes: ciphertext,
		},
	}, nil
}

func sealGCM(
	content []byte,
	block cipher.Block,
	oid asn1.ObjectIdentifier,
) (encryptedContentInfo, error) {
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return encryptedContentInfo{}, fmt.Errorf("as2: build GCM: %w", err)
	}
	nonce := make([]byte, gcmNonceSize)
	if _, err = rand.Read(nonce); err != nil {
		return encryptedContentInfo{}, fmt.Errorf("as2: generate nonce: %w", err)
	}
	ciphertext := aead.Seal(nil, nonce, content, nil)
	parameters, err := asn1.Marshal(gcmParameters{Nonce: nonce, ICVLen: aead.Overhead()})
	if err != nil {
		return encryptedContentInfo{}, fmt.Errorf("as2: encode GCM parameters: %w", err)
	}
	return encryptedContentInfo{
		ContentType: oidData,
		ContentEncryptionAlgorithm: pkix.AlgorithmIdentifier{
			Algorithm: oid,
			Parameters: asn1.RawValue{
				Class:      asn1.ClassUniversal,
				Tag:        asn1.TagSequence,
				IsCompound: true,
				Bytes:      parameters,
			},
		},
		EncryptedContent: asn1.RawValue{
			Class: asn1.ClassContextSpecific,
			Tag:   0,
			Bytes: ciphertext,
		},
	}, nil
}

func padPKCS7(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padded := make([]byte, len(data)+padding)
	copy(padded, data)
	for i := len(data); i < len(padded); i++ {
		padded[i] = byte(padding) //nolint:gosec // padding is at most one block
	}
	return padded
}

func setOddParity(key []byte) {
	for i, b := range key {
		if bits.OnesCount8(b&0xFE)%2 == 0 {
			key[i] = b&0xFE | 1
		} else {
			key[i] = b & 0xFE
		}
	}
}

func envelopeContentAlgorithm(der []byte) (asn1.ObjectIdentifier, error) {
	root, _, err := readBERElement(der, 0)
	if err != nil {
		return nil, err
	}
	if !root.is(asn1.ClassUniversal, asn1.TagSequence, true) {
		return nil, errMalformedEnvelope
	}
	rootChildren, err := root.children(2)
	if err != nil {
		return nil, err
	}
	if len(rootChildren) < 2 {
		return nil, errMalformedEnvelope
	}
	contentType, err := rootChildren[0].objectIdentifier()
	if err != nil {
		return nil, err
	}
	if !contentType.Equal(oidEnvelopedData) {
		return nil, errMalformedEnvelope
	}
	if !rootChildren[1].is(asn1.ClassContextSpecific, 0, true) {
		return nil, errMalformedEnvelope
	}
	explicit, err := rootChildren[1].children(1)
	if err != nil {
		return nil, err
	}
	if len(explicit) != 1 || !explicit[0].is(asn1.ClassUniversal, asn1.TagSequence, true) {
		return nil, errMalformedEnvelope
	}
	fields, err := explicit[0].children(maxEnvelopeFields)
	if err != nil {
		return nil, err
	}
	sawRecipients := false
	for _, field := range fields {
		switch {
		case field.is(asn1.ClassUniversal, asn1.TagSet, true):
			sawRecipients = true
		case sawRecipients && field.is(asn1.ClassUniversal, asn1.TagSequence, true):
			return encryptedContentAlgorithm(field)
		}
	}
	return nil, errMalformedEnvelope
}

func encryptedContentAlgorithm(eci berElement) (asn1.ObjectIdentifier, error) {
	fields, err := eci.children(3)
	if err != nil {
		return nil, err
	}
	if len(fields) < 2 || !fields[1].is(asn1.ClassUniversal, asn1.TagSequence, true) {
		return nil, errMalformedEnvelope
	}
	algorithm, err := fields[1].children(2)
	if err != nil {
		return nil, err
	}
	if len(algorithm) == 0 {
		return nil, errMalformedEnvelope
	}
	return algorithm[0].objectIdentifier()
}
