package temporaltype

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"os"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
)

const (
	MetadataEncodingEncrypted = "binary/encrypted"
	MetadataEncryptionKeyID   = "encryption-key-id"
	CompressionThreshold      = 1024 // 1KB
)

type EncryptionCodec struct {
	KeyID string
}

func NewEncryptionDataConverter(options DataConverterOptions) converter.DataConverter {
	codecs := []converter.PayloadCodec{}

	// ! Add compression codec if enabled (must be added after encryption codec since they're applied in reverse)
	if options.EnableCompression {
		codecs = append(codecs, converter.NewZlibCodec(converter.ZlibCodecOptions{
			AlwaysEncode: options.CompressionThreshold <= 0, // Always compress if no threshold
		}))
	}

	if options.EnableEncryption && options.EncryptionKeyID != "" {
		codecs = append(codecs, &EncryptionCodec{
			KeyID: options.EncryptionKeyID,
		})
	}

	if len(codecs) == 0 {
		return converter.GetDefaultDataConverter()
	}

	return converter.NewCodecDataConverter(
		converter.GetDefaultDataConverter(),
		codecs...,
	)
}

type DataConverterOptions struct {
	EnableEncryption     bool
	EncryptionKeyID      string
	EnableCompression    bool
	CompressionThreshold int
}

func (e *EncryptionCodec) Encode(payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	result := make([]*commonpb.Payload, len(payloads))

	for i, p := range payloads {
		origBytes, err := p.Marshal()
		if err != nil {
			return payloads, fmt.Errorf("failed to marshal payload: %w", err)
		}

		key := e.getKey(e.KeyID)
		if key == nil {
			return payloads, fmt.Errorf("encryption key not found for ID: %s", e.KeyID)
		}

		encrypted, err := e.encrypt(origBytes, key)
		if err != nil {
			return payloads, fmt.Errorf("encryption failed: %w", err)
		}

		result[i] = &commonpb.Payload{
			Metadata: map[string][]byte{
				converter.MetadataEncoding: []byte(MetadataEncodingEncrypted),
				MetadataEncryptionKeyID:    []byte(e.KeyID),
			},
			Data: encrypted,
		}
	}

	return result, nil
}

func (e *EncryptionCodec) Decode(payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	result := make([]*commonpb.Payload, len(payloads))

	for i, p := range payloads {
		if string(p.GetMetadata()[converter.MetadataEncoding]) != MetadataEncodingEncrypted {
			result[i] = p
			continue
		}

		keyID, ok := p.GetMetadata()[MetadataEncryptionKeyID]
		if !ok {
			return payloads, ErrNoEncryptionKeyID
		}

		key := e.getKey(string(keyID))
		if key == nil {
			return payloads, fmt.Errorf("decryption key not found for ID: %s", string(keyID))
		}

		decrypted, err := e.decrypt(p.GetData(), key)
		if err != nil {
			return payloads, fmt.Errorf("decryption failed: %w", err)
		}

		result[i] = &commonpb.Payload{}
		if err = result[i].Unmarshal(decrypted); err != nil {
			return payloads, fmt.Errorf("failed to unmarshal decrypted payload: %w", err)
		}
	}

	return result, nil
}

func (e *EncryptionCodec) getKey(keyID string) []byte {
	envKey := fmt.Sprintf("TEMPORAL_ENCRYPTION_KEY_%s", keyID)
	key := os.Getenv(envKey)

	if key == "" {
		key = os.Getenv("TEMPORAL_ENCRYPTION_KEY")
	}

	if key == "" {
		return nil
	}

	keyBytes := []byte(key)
	if len(keyBytes) < 32 {
		padded := make([]byte, 32)
		copy(padded, keyBytes)
		return padded
	}

	return keyBytes[:32]
}

func (e *EncryptionCodec) encrypt(plainData, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plainData, nil), nil
}

func (e *EncryptionCodec) decrypt(encryptedData, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(encryptedData) < nonceSize {
		return nil, ErrCiphertextTooShort
	}

	nonce, ciphertext := encryptedData[:nonceSize], encryptedData[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

type CustomCompressionCodec struct {
	Threshold int
}

func NewCompressionCodec(threshold int) converter.PayloadCodec {
	if threshold <= 0 {
		threshold = CompressionThreshold
	}
	return &CustomCompressionCodec{
		Threshold: threshold,
	}
}

func (c *CustomCompressionCodec) Encode(payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	result := make([]*commonpb.Payload, len(payloads))

	for i, p := range payloads {
		data := p.GetData()
		if len(data) < c.Threshold {
			result[i] = p
			continue
		}

		compressed, err := c.compress(data)
		if err != nil {
			return payloads, fmt.Errorf("compression failed: %w", err)
		}

		if len(compressed) >= len(data) {
			result[i] = p
			continue
		}

		result[i] = &commonpb.Payload{
			Metadata: map[string][]byte{
				converter.MetadataEncoding: []byte("binary/gzip"),
				"uncompressed-size":        fmt.Appendf(nil, "%d", len(data)),
			},
			Data: compressed,
		}
	}

	return result, nil
}

func (c *CustomCompressionCodec) Decode(payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	result := make([]*commonpb.Payload, len(payloads))

	for i, p := range payloads {
		metadata := p.GetMetadata()
		if string(metadata[converter.MetadataEncoding]) != "binary/gzip" {
			result[i] = p
			continue
		}

		decompressed, err := c.decompress(p.GetData())
		if err != nil {
			return payloads, fmt.Errorf("decompression failed: %w", err)
		}

		result[i] = &commonpb.Payload{
			Metadata: metadata,
			Data:     decompressed,
		}

		delete(result[i].GetMetadata(), converter.MetadataEncoding)
		delete(result[i].GetMetadata(), "uncompressed-size")
	}

	return result, nil
}

func (c *CustomCompressionCodec) compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)

	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (c *CustomCompressionCodec) decompress(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var buf bytes.Buffer
	const maxDecompressedSize = 100 * 1024 * 1024 // 100MB limit
	if _, err = io.Copy(&buf, io.LimitReader(reader, maxDecompressedSize)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
