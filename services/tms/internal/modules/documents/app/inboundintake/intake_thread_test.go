package inboundintake

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/app/inboundqueuetest"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubReplies struct {
	inboundmessageports.InboundReplyReader

	keys  map[string]string
	asked [][]string
}

func (s *stubReplies) ThreadKeyForMessageKeys(
	_ context.Context,
	_ pagination.TenantInfo,
	messageKeys []string,
) (string, bool, error) {
	s.asked = append(s.asked, messageKeys)
	for _, key := range messageKeys {
		if thread, ok := s.keys[key]; ok {
			return thread, true, nil
		}
	}

	return "", false, nil
}

func TestThreadKeyFor_FollowsTheFirstKnownMessageNamed(t *testing.T) {
	t.Parallel()

	svc, fakes := withQueue(&testService{})
	fakes.Queue.ThreadKeys["older@acme.example"] = "thread-older"
	fakes.Queue.ThreadKeys["parent@acme.example"] = "thread-parent"

	key, err := svc.intake().threadKeyFor(t.Context(), &threadLookup{
		tenant:     inboundqueuetest.QueueTenant,
		ownKey:     "child@acme.example",
		inReplyTo:  "<Parent@Acme.example>",
		references: []string{"<older@acme.example>", "<parent@acme.example>"},
	})
	require.NoError(t, err)
	assert.Equal(t, "thread-parent", key, "In-Reply-To is the closest relative")

	key, err = svc.intake().threadKeyFor(t.Context(), &threadLookup{
		tenant:     inboundqueuetest.QueueTenant,
		references: []string{"<older@acme.example>", "<unknown@acme.example>"},
	})
	require.NoError(t, err)
	assert.Equal(t, "thread-older", key, "references are read newest first")

	key, err = svc.intake().
		threadKeyFor(t.Context(), &threadLookup{tenant: inboundqueuetest.QueueTenant})
	require.NoError(t, err)
	assert.Empty(t, key, "a message naming nothing starts its own thread")
}

func TestThreadKeyFor_PrefersOurReplyWhenItIsCloserThanAKnownMessage(t *testing.T) {
	t.Parallel()

	replies := &stubReplies{keys: map[string]string{"reply@trenova.example": "thread-from-reply"}}
	svc, fakes := withQueue(&testService{replies: replies})
	fakes.Queue.ThreadKeys["original@acme.example"] = "thread-original"

	key, err := svc.intake().threadKeyFor(t.Context(), &threadLookup{
		tenant:     inboundqueuetest.QueueTenant,
		inReplyTo:  "<reply@trenova.example>",
		references: []string{"<original@acme.example>", "<reply@trenova.example>"},
	})
	require.NoError(t, err)
	assert.Equal(t, "thread-from-reply", key)
	require.Len(t, replies.asked, 1)
	assert.Equal(t, []string{"reply@trenova.example"}, replies.asked[0],
		"only the names closer than the known inbound message are asked")
}
