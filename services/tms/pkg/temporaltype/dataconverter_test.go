package temporaltype

import (
	"testing"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEncryptionDataConverter_NoCodecs(t *testing.T) {
	t.Parallel()

	dc := NewEncryptionDataConverter(DataConverterOptions{})
	assert.NotNil(t, dc)
}

func TestNewEncryptionDataConverter_WithEncryption(t *testing.T) {
	t.Parallel()

	dc := NewEncryptionDataConverter(DataConverterOptions{
		EnableEncryption: true,
		EncryptionKeyID:  "test-key",
	})
	assert.NotNil(t, dc)
}

func TestNewEncryptionDataConverter_WithCompression(t *testing.T) {
	t.Parallel()

	dc := NewEncryptionDataConverter(DataConverterOptions{
		EnableCompression:    true,
		CompressionThreshold: 512,
	})
	assert.NotNil(t, dc)
}

func TestNewEncryptionDataConverter_WithBoth(t *testing.T) {
	t.Parallel()

	dc := NewEncryptionDataConverter(DataConverterOptions{
		EnableEncryption:     true,
		EncryptionKeyID:      "test-key",
		EnableCompression:    true,
		CompressionThreshold: 512,
	})
	assert.NotNil(t, dc)
}

func TestNewEncryptionDataConverter_CompressionNoThreshold(t *testing.T) {
	t.Parallel()

	dc := NewEncryptionDataConverter(DataConverterOptions{
		EnableCompression:    true,
		CompressionThreshold: 0,
	})
	assert.NotNil(t, dc)
}

func TestEncryptionCodec_GetKey_NoEnvVar(t *testing.T) {
	t.Parallel()

	codec := &EncryptionCodec{KeyID: "nonexistent-key"}
	key := codec.getKey("nonexistent-key")
	assert.Nil(t, key)
}

func TestEncryptionCodec_GetKey_WithEnvVar(t *testing.T) {
	t.Setenv("TEMPORAL_ENCRYPTION_KEY_test", "12345678901234567890123456789012")

	codec := &EncryptionCodec{KeyID: "test"}
	key := codec.getKey("test")
	require.NotNil(t, key)
	assert.Len(t, key, 32)
}

func TestEncryptionCodec_GetKey_ShortKey(t *testing.T) {
	t.Setenv("TEMPORAL_ENCRYPTION_KEY_short", "shortkey")

	codec := &EncryptionCodec{KeyID: "short"}
	key := codec.getKey("short")
	require.NotNil(t, key)
	assert.Len(t, key, 32)
}

func TestEncryptionCodec_GetKey_FallbackKey(t *testing.T) {
	t.Setenv("TEMPORAL_ENCRYPTION_KEY", "fallback-key-that-is-at-least-32-bytes-long")

	codec := &EncryptionCodec{KeyID: "unknown"}
	key := codec.getKey("unknown")
	require.NotNil(t, key)
	assert.Len(t, key, 32)
}

func TestEncryptionCodec_EncryptDecrypt(t *testing.T) {
	t.Setenv("TEMPORAL_ENCRYPTION_KEY_roundtrip", "this-is-a-32-byte-encryption-ke")

	codec := &EncryptionCodec{KeyID: "roundtrip"}
	key := codec.getKey("roundtrip")
	require.NotNil(t, key)

	plaintext := []byte("hello world - this is secret data")
	encrypted, err := codec.encrypt(plaintext, key)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, encrypted)

	decrypted, err := codec.decrypt(encrypted, key)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptionCodec_Decrypt_CiphertextTooShort(t *testing.T) {
	t.Setenv("TEMPORAL_ENCRYPTION_KEY_short2", "12345678901234567890123456789012")

	codec := &EncryptionCodec{KeyID: "short2"}
	key := codec.getKey("short2")
	require.NotNil(t, key)

	_, err := codec.decrypt([]byte("short"), key)
	assert.ErrorIs(t, err, ErrCiphertextTooShort)
}

func TestEncryptionCodec_Encode_NoKey(t *testing.T) {
	t.Parallel()

	codec := &EncryptionCodec{KeyID: "missing-key"}
	payloads := []*commonpb.Payload{
		{Data: []byte("test data")},
	}

	_, err := codec.Encode(payloads)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "encryption key not found")
}

func TestEncryptionCodec_EncodeDecodeFull(t *testing.T) {
	t.Setenv("TEMPORAL_ENCRYPTION_KEY_full", "12345678901234567890123456789012")

	codec := &EncryptionCodec{KeyID: "full"}

	original := &commonpb.Payload{
		Metadata: map[string][]byte{
			converter.MetadataEncoding: []byte("json/plain"),
		},
		Data: []byte(`{"key":"value"}`),
	}

	encoded, err := codec.Encode([]*commonpb.Payload{original})
	require.NoError(t, err)
	require.Len(t, encoded, 1)
	assert.Equal(
		t,
		[]byte(MetadataEncodingEncrypted),
		encoded[0].Metadata[converter.MetadataEncoding],
	)

	decoded, err := codec.Decode(encoded)
	require.NoError(t, err)
	require.Len(t, decoded, 1)
	assert.Equal(t, original.Data, decoded[0].Data)
}

func TestEncryptionCodec_Decode_NotEncrypted(t *testing.T) {
	t.Parallel()

	codec := &EncryptionCodec{KeyID: "test"}
	payload := &commonpb.Payload{
		Metadata: map[string][]byte{
			converter.MetadataEncoding: []byte("json/plain"),
		},
		Data: []byte("not encrypted"),
	}

	decoded, err := codec.Decode([]*commonpb.Payload{payload})
	require.NoError(t, err)
	assert.Equal(t, payload, decoded[0])
}

func TestEncryptionCodec_Decode_NoKeyID(t *testing.T) {
	t.Parallel()

	codec := &EncryptionCodec{KeyID: "test"}
	payload := &commonpb.Payload{
		Metadata: map[string][]byte{
			converter.MetadataEncoding: []byte(MetadataEncodingEncrypted),
		},
		Data: []byte("encrypted data"),
	}

	_, err := codec.Decode([]*commonpb.Payload{payload})
	assert.ErrorIs(t, err, ErrNoEncryptionKeyID)
}

func TestNewCompressionCodec(t *testing.T) {
	t.Parallel()

	t.Run("custom threshold", func(t *testing.T) {
		t.Parallel()
		codec := NewCompressionCodec(2048)
		assert.NotNil(t, codec)
		cc := codec.(*CustomCompressionCodec)
		assert.Equal(t, 2048, cc.Threshold)
	})

	t.Run("zero threshold uses default", func(t *testing.T) {
		t.Parallel()
		codec := NewCompressionCodec(0)
		assert.NotNil(t, codec)
		cc := codec.(*CustomCompressionCodec)
		assert.Equal(t, CompressionThreshold, cc.Threshold)
	})

	t.Run("negative threshold uses default", func(t *testing.T) {
		t.Parallel()
		codec := NewCompressionCodec(-1)
		cc := codec.(*CustomCompressionCodec)
		assert.Equal(t, CompressionThreshold, cc.Threshold)
	})
}

func TestCustomCompressionCodec_CompressDecompress(t *testing.T) {
	t.Parallel()

	codec := &CustomCompressionCodec{Threshold: 10}

	data := []byte(
		"this is a test string that should be long enough to compress well when repeated this is a test string that should be long enough",
	)

	compressed, err := codec.compress(data)
	require.NoError(t, err)
	assert.Less(t, len(compressed), len(data))

	decompressed, err := codec.decompress(compressed)
	require.NoError(t, err)
	assert.Equal(t, data, decompressed)
}

func TestCustomCompressionCodec_Encode_BelowThreshold(t *testing.T) {
	t.Parallel()

	codec := &CustomCompressionCodec{Threshold: 1024}
	payload := &commonpb.Payload{
		Metadata: map[string][]byte{
			converter.MetadataEncoding: []byte("json/plain"),
		},
		Data: []byte("small"),
	}

	encoded, err := codec.Encode([]*commonpb.Payload{payload})
	require.NoError(t, err)
	assert.Equal(t, payload, encoded[0])
}

func TestCustomCompressionCodec_Encode_AboveThreshold(t *testing.T) {
	t.Parallel()

	codec := &CustomCompressionCodec{Threshold: 10}

	largeData := make([]byte, 100)
	for i := range largeData {
		largeData[i] = 'A'
	}

	payload := &commonpb.Payload{
		Metadata: map[string][]byte{
			converter.MetadataEncoding: []byte("json/plain"),
		},
		Data: largeData,
	}

	encoded, err := codec.Encode([]*commonpb.Payload{payload})
	require.NoError(t, err)
	assert.Equal(t, []byte("binary/gzip"), encoded[0].Metadata[converter.MetadataEncoding])
	assert.Less(t, len(encoded[0].Data), len(largeData))
}

func TestCustomCompressionCodec_EncodeDecode_RoundTrip(t *testing.T) {
	t.Parallel()

	codec := &CustomCompressionCodec{Threshold: 10}

	largeData := make([]byte, 200)
	for i := range largeData {
		largeData[i] = byte(i%26) + 'a'
	}

	payload := &commonpb.Payload{
		Metadata: map[string][]byte{
			converter.MetadataEncoding: []byte("json/plain"),
		},
		Data: largeData,
	}

	encoded, err := codec.Encode([]*commonpb.Payload{payload})
	require.NoError(t, err)

	decoded, err := codec.Decode(encoded)
	require.NoError(t, err)
	assert.Equal(t, largeData, decoded[0].Data)
}

func TestCustomCompressionCodec_Decode_NotCompressed(t *testing.T) {
	t.Parallel()

	codec := &CustomCompressionCodec{Threshold: 1024}
	payload := &commonpb.Payload{
		Metadata: map[string][]byte{
			converter.MetadataEncoding: []byte("json/plain"),
		},
		Data: []byte("not compressed"),
	}

	decoded, err := codec.Decode([]*commonpb.Payload{payload})
	require.NoError(t, err)
	assert.Equal(t, payload, decoded[0])
}

func TestCustomCompressionCodec_Decode_InvalidGzip(t *testing.T) {
	t.Parallel()

	codec := &CustomCompressionCodec{Threshold: 10}
	payload := &commonpb.Payload{
		Metadata: map[string][]byte{
			converter.MetadataEncoding: []byte("binary/gzip"),
		},
		Data: []byte("not valid gzip data"),
	}

	_, err := codec.Decode([]*commonpb.Payload{payload})
	assert.Error(t, err)
}
