package as2

import (
	"bytes"
	"compress/zlib"
	"encoding/asn1"
	"fmt"
	"io"
)

var (
	oidCompressedData        = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 1, 9}
	oidCompressionZlib       = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 3, 8}
	oidData                  = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 1}
	errNotCompressedData     = fmt.Errorf("as2: content is not CMS compressed data")
	errUnsupportedCompressor = fmt.Errorf("as2: unsupported compression algorithm")
)

type compressedContentInfo struct {
	ContentType asn1.ObjectIdentifier
	Content     compressedData `asn1:"explicit,tag:0"`
}

type compressedData struct {
	Version              int
	CompressionAlgorithm compressionAlgorithmIdentifier
	EncapContentInfo     encapsulatedContentInfo
}

type compressionAlgorithmIdentifier struct {
	Algorithm asn1.ObjectIdentifier
}

type encapsulatedContentInfo struct {
	EContentType asn1.ObjectIdentifier
	EContent     []byte `asn1:"explicit,optional,tag:0"`
}

func Compress(content []byte) ([]byte, error) {
	var compressed bytes.Buffer
	writer := zlib.NewWriter(&compressed)
	if _, err := writer.Write(content); err != nil {
		return nil, fmt.Errorf("as2: compress content: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("as2: finish compression: %w", err)
	}
	encoded, err := asn1.Marshal(compressedContentInfo{
		ContentType: oidCompressedData,
		Content: compressedData{
			Version:              0,
			CompressionAlgorithm: compressionAlgorithmIdentifier{Algorithm: oidCompressionZlib},
			EncapContentInfo: encapsulatedContentInfo{
				EContentType: oidData,
				EContent:     compressed.Bytes(),
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("as2: encode compressed data: %w", err)
	}
	return encoded, nil
}

func Decompress(encoded []byte) ([]byte, error) {
	var info compressedContentInfo
	if _, err := asn1.Unmarshal(encoded, &info); err != nil {
		return nil, fmt.Errorf("as2: decode compressed data: %w", err)
	}
	if !info.ContentType.Equal(oidCompressedData) {
		return nil, errNotCompressedData
	}
	if !info.Content.CompressionAlgorithm.Algorithm.Equal(oidCompressionZlib) {
		return nil, errUnsupportedCompressor
	}
	reader, err := zlib.NewReader(bytes.NewReader(info.Content.EncapContentInfo.EContent))
	if err != nil {
		return nil, fmt.Errorf("as2: open compressed content: %w", err)
	}
	defer reader.Close()
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("as2: decompress content: %w", err)
	}
	return content, nil
}

func IsCompressedData(encoded []byte) bool {
	var info compressedContentInfo
	if _, err := asn1.Unmarshal(encoded, &info); err != nil {
		return false
	}
	return info.ContentType.Equal(oidCompressedData)
}
