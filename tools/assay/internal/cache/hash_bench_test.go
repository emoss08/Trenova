package cache

import (
	"crypto/sha256"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"testing"

	"github.com/zeebo/blake3"
)

func benchPayload(size int) []byte {
	buf := make([]byte, size)
	for i := range buf {
		buf[i] = byte(i * 7)
	}

	return buf
}

func BenchmarkFingerprint(b *testing.B) {
	const (
		fileCount = 3169
		fileSize  = 4760
	)

	root := b.TempDir()
	payload := benchPayload(fileSize)
	for i := range fileCount {
		dir := filepath.Join(root, fmt.Sprintf("group%02d", i%64), fmt.Sprintf("pkg%05d", i/8))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			b.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("f%d.go", i)), payload, 0o644); err != nil {
			b.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/b\n"), 0o644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Compute(b.Context(), Inputs{Root: root, Env: []string{"GOOS=linux"}}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHashBulk(b *testing.B) {
	const corpus = 15 << 20

	payload := benchPayload(corpus)

	b.Run("sha256", func(b *testing.B) {
		b.SetBytes(corpus)
		for b.Loop() {
			h := sha256.New()
			h.Write(payload)
			h.Sum(nil)
		}
	})

	b.Run("blake3", func(b *testing.B) {
		b.SetBytes(corpus)
		for b.Loop() {
			h := blake3.New()
			h.Write(payload)
			h.Sum(nil)
		}
	})

	b.Run("fnv128", func(b *testing.B) {
		b.SetBytes(corpus)
		for b.Loop() {
			h := fnv.New128a()
			h.Write(payload)
			h.Sum(nil)
		}
	})
}
