package inboundintake

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/app/inboundqueuetest"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInboundMailLandsOnlyForAnOrganizationWithTriage(t *testing.T) {
	t.Parallel()

	mailbox := inboundqueuetest.TestMailbox()
	now := time.Unix(1784131200, 0)

	messages := &stubMessageRepo{}
	off := newService(&stubMailboxRepo{mailbox: mailbox}, messages, &stubStorage{})
	off.features = plantest.SharedServer().
		Without(mailbox.OrganizationID, platformcatalog.FeatureAITriage)
	_, err := off.intake().ReceiveWebhook(t.Context(), signed(t, resendBody(), now))
	require.ErrorIs(t, err, ErrUnknownMailbox, "answered like a mailbox that does not exist")
	assert.Nil(t, messages.created)

	messages = &stubMessageRepo{}
	on := newService(&stubMailboxRepo{mailbox: mailbox}, messages, &stubStorage{})
	on.features = plantest.SharedServer().
		Planned(mailbox.OrganizationID).
		Without(pulid.MustNew("org_"), platformcatalog.FeatureAITriage)
	_, err = on.intake().ReceiveWebhook(t.Context(), signed(t, resendBody(), now))
	require.NoError(t, err, "another organization's plan does not count")
	assert.NotNil(t, messages.created)
}
