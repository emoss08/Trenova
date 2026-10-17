package document

import (
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type SignatureSource string

const (
	SignatureSourceExtraction SignatureSource = "Extraction"
	SignatureSourcePerson     SignatureSource = "Person"
)

const signingDateSkewSeconds int64 = 24 * 60 * 60

func (s SignatureStatus) IsValid() bool {
	switch s {
	case SignatureStatusUnknown, SignatureStatusSigned, SignatureStatusUnsigned:
		return true
	}
	return false
}

func (s SignatureStatus) String() string {
	return string(s)
}

func (s SignatureSource) IsValid() bool {
	return s == SignatureSourceExtraction || s == SignatureSourcePerson
}

func (d *Document) SignatureSetByPerson() bool {
	return d.SignatureSource == SignatureSourcePerson
}

func (d *Document) AcceptsDetectedSignature() bool {
	return !d.SignatureSetByPerson()
}

type SignatureMark struct {
	Status   SignatureStatus
	SignedAt *int64
	By       pulid.ID
	At       int64
}

func (d *Document) SignatureMatches(status SignatureStatus, signedAt *int64) bool {
	if d.SignatureStatus != status {
		return false
	}
	if d.SignedAt == nil || signedAt == nil {
		return d.SignedAt == nil && signedAt == nil
	}
	return *d.SignedAt == *signedAt
}

func (d *Document) MarkSignature(mark *SignatureMark) error {
	if mark.By.IsNil() {
		return errortypes.NewAuthorizationError(
			"Only a person can record whether a document is signed",
		)
	}

	multiErr := errortypes.NewMultiError()
	if !mark.Status.IsValid() {
		multiErr.Add(
			"signatureStatus",
			errortypes.ErrInvalid,
			"Signature status must be Signed, Unsigned or Unknown",
		)
	}
	if mark.SignedAt != nil {
		switch {
		case mark.Status != SignatureStatusSigned:
			multiErr.Add(
				"signedAt",
				errortypes.ErrInvalid,
				"Only a signed document has a signing date",
			)
		case *mark.SignedAt <= 0:
			multiErr.Add("signedAt", errortypes.ErrInvalid, "Signing date is invalid")
		case *mark.SignedAt > mark.At+signingDateSkewSeconds:
			multiErr.Add("signedAt", errortypes.ErrInvalid, "Signing date can't be in the future")
		}
	}
	if multiErr.HasErrors() {
		return multiErr
	}

	if !d.IsCurrentVersion {
		return errortypes.NewBusinessError(
			"Only the current version of a document can have its signature recorded. Record it on the latest version instead.",
		)
	}

	d.SignatureStatus = mark.Status
	d.SignedAt = nil
	if mark.Status == SignatureStatusSigned && mark.SignedAt != nil {
		signedAt := *mark.SignedAt
		d.SignedAt = &signedAt
	}
	d.SignatureSource = SignatureSourcePerson
	d.SignatureUpdatedByID = mark.By
	at := mark.At
	d.SignatureUpdatedAt = &at

	return nil
}
