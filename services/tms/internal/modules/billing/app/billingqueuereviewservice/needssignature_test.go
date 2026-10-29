package billingqueuereviewservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func signedPODType(requiresSignature bool) *documenttype.DocumentType {
	return &documenttype.DocumentType{
		ID:                pulid.MustNew("dt_"),
		Code:              documenttype.CodePOD,
		Name:              "Proof of delivery",
		RequiresSignature: requiresSignature,
	}
}

func signedPODDocument(
	docType *documenttype.DocumentType,
	status document.SignatureStatus,
	signedAt *int64,
) *document.Document {
	return &document.Document{
		ID:               pulid.MustNew("doc_"),
		OriginalName:     "pod.pdf",
		IsCurrentVersion: true,
		DocumentTypeID:   &docType.ID,
		DocumentType:     docType,
		SignatureStatus:  status,
		SignedAt:         signedAt,
	}
}

func TestNeedsSignatureReadsOnlyCurrentRequiredPODs(t *testing.T) {
	t.Parallel()

	required := signedPODDocument(signedPODType(true), document.SignatureStatusUnknown, nil)
	assert.True(t, needsSignature(required))

	notRequired := signedPODDocument(signedPODType(false), document.SignatureStatusUnknown, nil)
	assert.False(t, needsSignature(notRequired))

	old := signedPODDocument(signedPODType(true), document.SignatureStatusUnknown, nil)
	old.IsCurrentVersion = false
	assert.False(t, needsSignature(old))

	rejected := signedPODDocument(signedPODType(true), document.SignatureStatusUnknown, nil)
	rejected.Status = document.StatusRejected
	assert.False(t, needsSignature(rejected))

	untyped := signedPODDocument(signedPODType(true), document.SignatureStatusUnknown, nil)
	untyped.DocumentType = nil
	assert.False(t, needsSignature(untyped))

	assert.False(t, needsSignature(nil))
}
