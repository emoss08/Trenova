package ediinboundpoll

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editransportports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestStageFetched_AnUnreadableFileDoesNotStopTheRest(t *testing.T) {
	t.Parallel()

	svc := &Service{l: zap.NewNop()}
	profile := &edi.EDICommunicationProfile{ID: pulid.MustNew("edicp_")}
	result := &PollMailboxResult{ProfileID: profile.ID}

	svc.stageFetched(
		t.Context(),
		profile,
		&editransportports.EDIInboundFetchRequest{Profile: profile},
		[]*editransportports.EDIInboundRemoteFile{
			{Path: "/in/too-large.x12", ReadError: errors.New("over the limit")},
			nil,
			{Path: "/in/empty.x12", Contents: "  "},
			{Path: "/in/also-bad.x12", ReadError: errors.New("connection lost")},
		},
		result,
	)

	assert.Equal(t, 2, result.FailedFiles)
	assert.Equal(t, 2, result.SkippedFiles)
	assert.Empty(t, result.StagedFileIDs)
}
