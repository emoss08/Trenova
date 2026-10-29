package billingqueuereadiness

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueuereview"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func podType(requiresSignature bool) *documenttype.DocumentType {
	return &documenttype.DocumentType{
		ID:                pulid.MustNew("dt_"),
		Code:              podCode,
		Name:              "Proof of delivery",
		RequiresSignature: requiresSignature,
	}
}

func podDocument(
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

func TestPODFindingFollowsTheSignatureStatus(t *testing.T) {
	t.Parallel()

	signedAt := int64(1_792_000_000)
	cases := []struct {
		name         string
		status       document.SignatureStatus
		requires     bool
		signed       bool
		unsigned     bool
		tileState    billingqueue.DocumentState
		code         billingqueue.IssueCode
		wantSignedAt bool
	}{
		{
			name:         "signed",
			status:       document.SignatureStatusSigned,
			requires:     true,
			signed:       true,
			tileState:    billingqueue.DocumentStateOK,
			wantSignedAt: true,
		},
		{
			name:      "unsigned",
			status:    document.SignatureStatusUnsigned,
			requires:  true,
			unsigned:  true,
			tileState: billingqueue.DocumentStateUnsigned,
			code:      billingqueue.IssuePODUnsigned,
		},
		{
			name:      "unknown",
			status:    document.SignatureStatusUnknown,
			requires:  true,
			tileState: billingqueue.DocumentStateOK,
		},
		{
			name:      "unsigned but the type needs no signature",
			status:    document.SignatureStatusUnsigned,
			requires:  false,
			tileState: billingqueue.DocumentStateOK,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			docType := podType(tc.requires)
			var at *int64
			if tc.status == document.SignatureStatusSigned {
				at = &signedAt
			}
			doc := podDocument(docType, tc.status, at)
			required := []*documenttype.DocumentType{docType}

			pod := PODState(nil, []*document.Document{doc}, required)

			assert.True(t, pod.Required)
			assert.True(t, pod.Present)
			assert.Equal(t, doc.ID, pod.DocumentID)
			assert.Equal(t, tc.signed, pod.Signed)
			assert.Equal(t, tc.unsigned, pod.Unsigned)
			if tc.wantSignedAt {
				require.NotNil(t, pod.SignedAt)
				assert.Equal(t, signedAt, *pod.SignedAt)
			}

			tiles := documentTiles([]*document.Document{doc}, required)
			require.Len(t, tiles, 1)
			assert.Equal(t, tc.tileState, tiles[0].State)
			assert.Equal(t, tc.signed, tiles[0].Signed)

			findings := billingqueuereview.Findings(&billingqueuereview.FindingInput{
				CustomerName: "Acme",
				POD:          pod,
			})
			if tc.code == "" {
				assert.Empty(t, findings)
				return
			}
			require.Len(t, findings, 1)
			assert.Equal(t, tc.code, findings[0].Code)
			assert.Equal(t, doc.ID.String(), findings[0].SubjectKey)
		})
	}
}

func TestPODFindingPrefersASignedCopyOverAnUnsignedOne(t *testing.T) {
	t.Parallel()

	docType := podType(true)
	unsigned := podDocument(docType, document.SignatureStatusUnsigned, nil)
	signed := podDocument(docType, document.SignatureStatusSigned, nil)
	required := []*documenttype.DocumentType{docType}

	for _, docs := range [][]*document.Document{
		{unsigned, signed},
		{signed, unsigned},
	} {
		pod := PODState(nil, docs, required)

		assert.True(t, pod.Signed)
		assert.False(t, pod.Unsigned)
		assert.Equal(t, signed.ID, pod.DocumentID)
		assert.Empty(t, billingqueuereview.Findings(&billingqueuereview.FindingInput{POD: pod}))

		tiles := documentTiles(docs, required)
		require.Len(t, tiles, 1)
		assert.Equal(t, signed.ID, tiles[0].DocumentID)
		assert.Equal(t, billingqueue.DocumentStateOK, tiles[0].State)
	}
}
