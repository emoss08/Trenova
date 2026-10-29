package detentionnoticeservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionnotice"
	"github.com/stretchr/testify/require"
)

func TestNoticeKindFor_MatchesTheMomentOfTheStay(t *testing.T) {
	t.Parallel()

	base := int64(1_767_225_600)
	stopped := base + 3600

	occ := &detention.DetentionOccurrence{FreeTimeExpiresAt: base + 1800}
	require.Equal(t, detentionnotice.NoticeKindWarning, noticeKindFor(occ, base))

	require.Equal(t, detentionnotice.NoticeKindStarted, noticeKindFor(occ, base+3600))

	sentAt := base + 1900
	occ.NoticeSentAt = &sentAt
	require.Equal(t, detentionnotice.NoticeKindUpdate, noticeKindFor(occ, base+3600))

	occ.ClockStopAt = &stopped
	require.Equal(t, detentionnotice.NoticeKindFinal, noticeKindFor(occ, base+7200))
}
