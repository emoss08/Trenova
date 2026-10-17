package document

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const signatureTestNow = int64(1_792_000_000)

func TestMarkSignatureRecordsWhoSaidSo(t *testing.T) {
	t.Parallel()

	doc := &Document{IsCurrentVersion: true, SignatureStatus: SignatureStatusUnknown}
	person := pulid.MustNew("usr_")
	signedAt := signatureTestNow - 3600

	require.NoError(t, doc.MarkSignature(&SignatureMark{
		Status:   SignatureStatusSigned,
		SignedAt: &signedAt,
		By:       person,
		At:       signatureTestNow,
	}))

	assert.Equal(t, SignatureStatusSigned, doc.SignatureStatus)
	require.NotNil(t, doc.SignedAt)
	assert.Equal(t, signedAt, *doc.SignedAt)
	assert.Equal(t, SignatureSourcePerson, doc.SignatureSource)
	assert.Equal(t, person, doc.SignatureUpdatedByID)
	require.NotNil(t, doc.SignatureUpdatedAt)
	assert.Equal(t, signatureTestNow, *doc.SignatureUpdatedAt)
	assert.True(t, doc.SignatureSetByPerson())
	assert.False(t, doc.AcceptsDetectedSignature())
}

func TestMarkSignatureClearsTheDateWhenNoLongerSigned(t *testing.T) {
	t.Parallel()

	signedAt := signatureTestNow - 3600
	doc := &Document{
		IsCurrentVersion: true,
		SignatureStatus:  SignatureStatusSigned,
		SignedAt:         &signedAt,
		SignatureSource:  SignatureSourceExtraction,
	}

	require.NoError(t, doc.MarkSignature(&SignatureMark{
		Status: SignatureStatusUnknown,
		By:     pulid.MustNew("usr_"),
		At:     signatureTestNow,
	}))

	assert.Equal(t, SignatureStatusUnknown, doc.SignatureStatus)
	assert.Nil(t, doc.SignedAt)
	assert.True(t, doc.SignatureSetByPerson())
}

func TestMarkSignatureRefusesWhatItCannotRecord(t *testing.T) {
	t.Parallel()

	future := signatureTestNow + 3*24*3600
	past := signatureTestNow - 3600
	cases := map[string]*SignatureMark{
		"unknown status": {
			Status: "Smudged",
			By:     pulid.MustNew("usr_"),
			At:     signatureTestNow,
		},
		"date on unsigned": {
			Status:   SignatureStatusUnsigned,
			SignedAt: &past,
			By:       pulid.MustNew("usr_"),
			At:       signatureTestNow,
		},
		"date in the future": {
			Status:   SignatureStatusSigned,
			SignedAt: &future,
			By:       pulid.MustNew("usr_"),
			At:       signatureTestNow,
		},
		"no person":               {Status: SignatureStatusSigned, At: signatureTestNow},
		"not the current version": nil,
	}
	for name, mark := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			doc := &Document{IsCurrentVersion: true, SignatureStatus: SignatureStatusUnknown}
			if mark == nil {
				doc.IsCurrentVersion = false
				mark = &SignatureMark{
					Status: SignatureStatusSigned,
					By:     pulid.MustNew("usr_"),
					At:     signatureTestNow,
				}
			}

			require.Error(t, doc.MarkSignature(mark))
			assert.Equal(t, SignatureStatusUnknown, doc.SignatureStatus)
			assert.Empty(t, doc.SignatureSource)
		})
	}
}

func TestMarkSignatureRefusesAnAgentAsAuthorization(t *testing.T) {
	t.Parallel()

	err := (&Document{IsCurrentVersion: true}).MarkSignature(&SignatureMark{
		Status: SignatureStatusSigned,
		At:     signatureTestNow,
	})

	assert.True(t, errortypes.IsAuthorizationError(err))
}

func TestSignatureMatches(t *testing.T) {
	t.Parallel()

	at := signatureTestNow
	other := signatureTestNow + 1
	doc := &Document{SignatureStatus: SignatureStatusSigned, SignedAt: &at}

	assert.True(t, doc.SignatureMatches(SignatureStatusSigned, &at))
	assert.False(t, doc.SignatureMatches(SignatureStatusSigned, &other))
	assert.False(t, doc.SignatureMatches(SignatureStatusSigned, nil))
	assert.False(t, doc.SignatureMatches(SignatureStatusUnsigned, nil))
	assert.True(t, (&Document{SignatureStatus: SignatureStatusUnknown}).SignatureMatches(
		SignatureStatusUnknown,
		nil,
	))
}
