package identity

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrCreatePersists(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	first, err := LoadOrCreate(dir, "partner.test")
	if err != nil {
		t.Fatalf("first identity: %v", err)
	}
	second, err := LoadOrCreate(dir, "partner.test")
	if err != nil {
		t.Fatalf("second identity: %v", err)
	}
	if first.CertificatePEM != second.CertificatePEM {
		t.Fatal("the certificate was not reused")
	}
	if _, err := os.Stat(filepath.Join(dir, "as2-cert.pem")); err != nil {
		t.Fatalf("the certificate was not persisted: %v", err)
	}
	other, err := LoadOrCreate("", "partner.test")
	if err != nil || other.CertificatePEM == first.CertificatePEM {
		t.Fatalf("no directory should give a fresh identity: %v", err)
	}
}
