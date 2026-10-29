package completionrouter

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/completionengine"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingFileReader struct {
	seen []*conversationports.ReadMessageFileRequest
}

func (r *recordingFileReader) ReadMessageFile(
	_ context.Context,
	req *conversationports.ReadMessageFileRequest,
) (*conversationports.MessageFileContent, error) {
	r.seen = append(r.seen, req)

	return &conversationports.MessageFileContent{
		Data:        []byte("%PDF"),
		ContentType: "application/pdf",
	}, nil
}

// A file is read within the tenant of the request that carries it, and only
// by a document id that parses.
func TestFileSource_ReadsWithinTheRequestsTenant(t *testing.T) {
	t.Parallel()

	reader := &recordingFileReader{}
	svc := &completionengine.Engine{Files: reader}
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	documentID := pulid.MustNew("doc_")

	source := svc.FileSource(tenant)
	require.NotNil(t, source)
	content, err := source(t.Context(), documentID.String())
	require.NoError(t, err)
	assert.Equal(t, "application/pdf", content.ContentType)
	require.Len(t, reader.seen, 1)
	assert.Equal(t, tenant, reader.seen[0].TenantInfo)
	assert.Equal(t, documentID, reader.seen[0].DocumentID)

	_, err = source(t.Context(), "not-a-document")
	require.Error(t, err)
	assert.Len(t, reader.seen, 1, "an id that does not parse reads nothing")
}
