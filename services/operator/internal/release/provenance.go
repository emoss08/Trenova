package release

import (
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/verify"
)

const (
	chartSignerIssuer  = "https://token.actions.githubusercontent.com"
	chartSignerPattern = `^https://github\.com/emoss08/[Tt]renova/\.github/workflows/release\.yml@refs/tags/v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$`
)

var (
	ErrChartUnsigned = errors.New("the chart carries no signature")
	ErrChartIdentity = errors.New("the chart is not the release that was asked for")
)

type Provenance interface {
	Required() bool
	Verify(chart, signature []byte) error
}

type Unverified struct{}

func (Unverified) Required() bool { return false }

func (Unverified) Verify(_, _ []byte) error { return nil }

type Sigstore struct {
	material func() (root.TrustedMaterial, error)
	options  []verify.VerifierOption
}

func NewSigstore() *Sigstore {
	return &Sigstore{
		material: func() (root.TrustedMaterial, error) {
			return root.FetchTrustedRoot()
		},
		options: []verify.VerifierOption{
			verify.WithSignedCertificateTimestamps(1),
			verify.WithTransparencyLog(1),
			verify.WithObserverTimestamps(1),
		},
	}
}

func (*Sigstore) Required() bool { return true }

func (s *Sigstore) Verify(chart, signature []byte) error {
	if len(signature) == 0 {
		return ErrChartUnsigned
	}
	var b bundle.Bundle
	if err := b.UnmarshalJSON(signature); err != nil {
		return fmt.Errorf("read the chart's signature: %w", err)
	}
	return s.check(&b, chart)
}

func (s *Sigstore) check(entity verify.SignedEntity, chart []byte) error {
	material, err := s.material()
	if err != nil {
		return fmt.Errorf("read the Sigstore trust root: %w", err)
	}
	verifier, err := verify.NewVerifier(material, s.options...)
	if err != nil {
		return fmt.Errorf("build the chart verifier: %w", err)
	}
	identity, err := verify.NewShortCertificateIdentity(
		chartSignerIssuer,
		"",
		"",
		chartSignerPattern,
	)
	if err != nil {
		return fmt.Errorf("build the chart signer's identity: %w", err)
	}
	digest := sha256.Sum256(chart)
	if _, err = verifier.Verify(entity, verify.NewPolicy(
		verify.WithArtifactDigest("sha256", digest[:]),
		verify.WithCertificateIdentity(identity),
	)); err != nil {
		return fmt.Errorf("verify the chart's signature: %w", err)
	}
	return nil
}
