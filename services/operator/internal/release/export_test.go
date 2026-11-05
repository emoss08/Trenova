package release

import (
	"helm.sh/helm/v3/pkg/chart"

	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/verify"
)

func NewSigstoreFor(material root.TrustedMaterial, options ...verify.VerifierOption) *Sigstore {
	return &Sigstore{
		material: func() (root.TrustedMaterial, error) { return material, nil },
		options:  options,
	}
}

func (s *Sigstore) Check(entity verify.SignedEntity, chart []byte) error {
	return s.check(entity, chart)
}

func NamesRelease(ch *chart.Chart, name, version string) error {
	return namesRelease(ch, name, version)
}
