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
	// MetadataEncodingEncrypted is the encoding type for encrypted data
	MetadataEncodingEncrypted = "binary/encrypted"

	// MetadataEncryptionKeyID identifies which key was used for encryption
	MetadataEncryptionKeyID = "encryption-key-id"

	// CompressionThreshold is the minimum size in bytes before compression is applied
	CompressionThreshold = 1024 // 1KB
)

// EncryptionCodec implements PayloadCodec using AES-GCM encryption
type EncryptionCodec struct {
	KeyID string
}

// NewEncryptionDataConverter creates a new data converter with encryption
func NewEncryptionDataConverter(options DataConverterOptions) converter.DataConverter {
	codecs := []converter.PayloadCodec{}

	// Add compression codec if enabled (must be added after encryption codec since they're applied in reverse)
	if options.EnableCompression {
		codecs = append(codecs, converter.NewZlibCodec(converter.ZlibCodecOptions{
			AlwaysEncode: options.CompressionThreshold <= 0, // Always compress if no threshold
		}))
	}

	// Add encryption codec if enabled
	if options.EnableEncryption && options.EncryptionKeyID != "" {
		codecs = append(codecs, &EncryptionCodec{
			KeyID: options.EncryptionKeyID,
		})
	}

	if len(codecs) == 0 {
		// No security features enabled, return default converter
		return converter.GetDefaultDataConverter()
	}

	// Create codec data converter with our security codecs
	return converter.NewCodecDataConverter(
		converter.GetDefaultDataConverter(),
		codecs...,
	)
}

// DataConverterOptions contains options for the data converter
type DataConverterOptions struct {
	// EnableEncryption enables payload encryption
	EnableEncryption bool

	// EncryptionKeyID identifies which key to use
	EncryptionKeyID string

	// EnableCompression enables payload compression
	EnableCompression bool

	// CompressionThreshold is the minimum size before compression
	CompressionThreshold int
}

// Encode encrypts payloads
func (e *EncryptionCodec) Encode(payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	result := make([]*commonpb.Payload, len(payloads))

	for i, p := range payloads {
		// Marshal the original payload
		origBytes, err := p.Marshal()
		if err != nil {
			return payloads, fmt.Errorf("failed to marshal payload: %w", err)
		}

		// Get encryption key
		key := e.getKey(e.KeyID)
		if key == nil {
			return payloads, fmt.Errorf("encryption key not found for ID: %s", e.KeyID)
		}

		// Encrypt the payload
		encrypted, err := e.encrypt(origBytes, key)
		if err != nil {
			return payloads, fmt.Errorf("encryption failed: %w", err)
		}

		// Create new encrypted payload
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

// Decode decrypts payloads
func (e *EncryptionCodec) Decode(payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	result := make([]*commonpb.Payload, len(payloads))

	for i, p := range payloads {
		// Check if payload is encrypted
		if string(p.Metadata[converter.MetadataEncoding]) != MetadataEncodingEncrypted {
			result[i] = p
			continue
		}

		// Get the key ID used for encryption
		keyID, ok := p.Metadata[MetadataEncryptionKeyID]
		if !ok {
			return payloads, fmt.Errorf("no encryption key ID in metadata")
		}

		// Get decryption key
		key := e.getKey(string(keyID))
		if key == nil {
			return payloads, fmt.Errorf("decryption key not found for ID: %s", string(keyID))
		}

		// Decrypt the payload
		decrypted, err := e.decrypt(p.Data, key)
		if err != nil {
			return payloads, fmt.Errorf("decryption failed: %w", err)
		}

		// Unmarshal the decrypted payload
		result[i] = &commonpb.Payload{}
		if err := result[i].Unmarshal(decrypted); err != nil {
			return payloads, fmt.Errorf("failed to unmarshal decrypted payload: %w", err)
		}
	}

	return result, nil
}

// getKey retrieves the encryption key for the given ID
func (e *EncryptionCodec) getKey(keyID string) []byte {
	envKey := fmt.Sprintf("TEMPORAL_ENCRYPTION_KEY_%s", keyID)
	key := os.Getenv(envKey)

	if key == "" {
		// Fallback to a general key
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

// encrypt encrypts data using AES-GCM
func (e *EncryptionCodec) encrypt(plainData []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Prepend nonce to ciphertext
	return gcm.Seal(nonce, nonce, plainData, nil), nil
}

// decrypt decrypts data using AES-GCM
func (e *EncryptionCodec) decrypt(encryptedData []byte, key []byte) ([]byte, error) {
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
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := encryptedData[:nonceSize], encryptedData[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// CustomCompressionCodec implements compression with size threshold
type CustomCompressionCodec struct {
	Threshold int
}

// NewCompressionCodec creates a new compression codec
func NewCompressionCodec(threshold int) converter.PayloadCodec {
	if threshold <= 0 {
		threshold = CompressionThreshold
	}
	return &CustomCompressionCodec{
		Threshold: threshold,
	}
}

// Encode compresses payloads if they exceed the threshold
func (c *CustomCompressionCodec) Encode(payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	result := make([]*commonpb.Payload, len(payloads))

	for i, p := range payloads {
		// Check if payload is large enough to compress
		if len(p.Data) < c.Threshold {
			result[i] = p
			continue
		}

		// Compress the data
		compressed, err := c.compress(p.Data)
		if err != nil {
			return payloads, fmt.Errorf("compression failed: %w", err)
		}

		// Only use compression if it reduces size
		if len(compressed) >= len(p.Data) {
			result[i] = p
			continue
		}

		// Create compressed payload
		result[i] = &commonpb.Payload{
			Metadata: map[string][]byte{
				converter.MetadataEncoding: []byte("binary/gzip"),
				"uncompressed-size":        fmt.Appendf(nil, "%d", len(p.Data)),
			},
			Data: compressed,
		}
	}

	return result, nil
}

// Decode decompresses payloads
func (c *CustomCompressionCodec) Decode(payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	result := make([]*commonpb.Payload, len(payloads))

	for i, p := range payloads {
		// Check if payload is compressed
		if string(p.Metadata[converter.MetadataEncoding]) != "binary/gzip" {
			result[i] = p
			continue
		}

		// Decompress the data
		decompressed, err := c.decompress(p.Data)
		if err != nil {
			return payloads, fmt.Errorf("decompression failed: %w", err)
		}

		// Restore original payload
		result[i] = &commonpb.Payload{
			Metadata: p.Metadata,
			Data:     decompressed,
		}

		// Remove compression encoding
		delete(result[i].Metadata, converter.MetadataEncoding)
		delete(result[i].Metadata, "uncompressed-size")
	}

	return result, nil
}

// compress compresses data using gzip
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

// decompress decompresses data using gzip
func (c *CustomCompressionCodec) decompress(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
