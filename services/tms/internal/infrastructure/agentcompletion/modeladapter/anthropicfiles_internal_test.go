package modeladapter

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmessage"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	pdfID   = "doc_01RATECON"
	imageID = "doc_01PHOTO"
	docxID  = "doc_01NOTES"
	pdfText = "Rate confirmation: linehaul $2,400, fuel $310."
)

var (
	pdfBytes   = []byte("%PDF-1.7 rate confirmation")
	imageBytes = []byte("\x89PNG photo of the seal")
)

// filesServer answers the Messages API and keeps every request body it was
// sent. With refuse set it rejects any request carrying a file, as the API
// rejects a document it cannot read.
func filesServer(t *testing.T, refuse bool) (*httptest.Server, func() []map[string]any) {
	t.Helper()

	var (
		mu     sync.Mutex
		bodies []map[string]any
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := map[string]any{}
		_ = sonic.ConfigDefault.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		bodies = append(bodies, body)
		mu.Unlock()

		encoded, _ := sonic.MarshalString(body["messages"])
		if refuse &&
			(strings.Contains(encoded, `"document"`) || strings.Contains(encoded, `"image"`)) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"type":"error","error":{"type":"invalid_request_error",` +
				`"message":"The PDF specified could not be read"}}`))
			return
		}
		if body["stream"] == true {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte(wireStreamReply))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(wireBlockingReply))
	}))
	t.Cleanup(server.Close)

	return server, func() []map[string]any {
		mu.Lock()
		defer mu.Unlock()
		return append([]map[string]any(nil), bodies...)
	}
}

// fileStore stands in for the conversation's files and counts its reads.
type fileStore struct {
	mu    sync.Mutex
	files map[string]*MessageFileContent
	reads map[string]int
}

func newFileStore() *fileStore {
	return &fileStore{
		files: map[string]*MessageFileContent{
			pdfID:   {Data: pdfBytes, ContentType: "application/pdf", PageCount: 3},
			imageID: {Data: imageBytes, ContentType: "image/png"},
			docxID: {
				Data:        []byte("PK docx"),
				ContentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			},
		},
		reads: map[string]int{},
	}
}

func (s *fileStore) source(_ context.Context, documentID string) (*MessageFileContent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reads[documentID]++
	content, ok := s.files[documentID]
	if !ok {
		return nil, errors.New("not found")
	}

	return content, nil
}

func (s *fileStore) readsOf(documentID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.reads[documentID]
}

// attachedMessage is a user message as a turn opens it: the attachments block
// rendered ahead of what the person wrote, and each file placed in it.
func attachedMessage(attachments ...agentturn.RuntimeAttachment) Message {
	rendered := agentmessage.RenderAttachments(attachments, agentmessage.AttachmentTextBudget)
	content := rendered.Block + "What does the rate confirmation pay?"
	files := make([]MessageFile, 0, len(attachments))
	for idx := range attachments {
		start := strings.Index(content, rendered.Entries[idx])
		files = append(files, MessageFile{
			DocumentID:  attachments[idx].DocumentID,
			FileName:    attachments[idx].FileName,
			ContentType: attachments[idx].ContentType,
			Size:        attachments[idx].FileSize,
			PageCount:   attachments[idx].PageCount,
			EntryStart:  start,
			EntryEnd:    start + len(rendered.Entries[idx]),
			NativeEntry: rendered.NativeEntries[idx],
		})
	}

	return Message{Role: RoleUser, Content: content, Files: files}
}

func pdfAttachment() agentturn.RuntimeAttachment {
	return agentturn.RuntimeAttachment{
		DocumentID:  pdfID,
		FileName:    "ratecon.pdf",
		ContentType: "application/pdf",
		FileSize:    int64(len(pdfBytes)),
		PageCount:   3,
		Reading:     agentturn.AttachmentReadingDone,
		Text:        pdfText,
	}
}

func imageAttachment() agentturn.RuntimeAttachment {
	return agentturn.RuntimeAttachment{
		DocumentID:  imageID,
		FileName:    "seal.png",
		ContentType: "image/png",
		FileSize:    int64(len(imageBytes)),
		Reading:     agentturn.AttachmentReadingFailed,
	}
}

func filesCall(url, model string, store *fileStore, messages ...Message) *Call {
	call := callFor(
		aiprovider.KindAnthropicMessages,
		url,
		&Request{Messages: messages, MaxTokens: 1024},
	)
	call.Provider.Model = model
	call.APIKey = "sk-provider"
	if store != nil {
		call.Files = store.source
	}

	return call
}

// userBlocks is the content of the first user message the provider was sent.
func userBlocks(t *testing.T, body map[string]any) []map[string]any {
	t.Helper()

	messages, ok := body["messages"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, messages)
	first, ok := messages[0].(map[string]any)
	require.True(t, ok)
	content, ok := first["content"].([]any)
	require.True(t, ok)
	blocks := make([]map[string]any, 0, len(content))
	for _, block := range content {
		typed, ok := block.(map[string]any)
		require.True(t, ok)
		blocks = append(blocks, typed)
	}

	return blocks
}

func blockTypes(blocks []map[string]any) []string {
	types := make([]string, 0, len(blocks))
	for _, block := range blocks {
		types = append(types, block["type"].(string))
	}

	return types
}

/*
Claude reads a PDF the person attached as the file itself, its layout, tables
and signatures included, rather than as the text read from it. The text is not
sent as well, and the entry no longer says what reading the file came to.
*/
func TestAnthropicAdapter_SendsAnAttachedPDFAsADocument(t *testing.T) {
	t.Parallel()

	server, bodies := filesServer(t, false)
	store := newFileStore()
	call := filesCall(server.URL, "claude-opus-5-5", store, attachedMessage(pdfAttachment()))

	resp, err := NewAnthropicAdapter().Complete(t.Context(), call)
	require.NoError(t, err)

	blocks := userBlocks(t, bodies()[0])
	require.Equal(t, []string{"document", "text"}, blockTypes(blocks))
	document := blocks[0]
	assert.Equal(t, "ratecon.pdf", document["title"])
	assert.Equal(t, "Attached file "+pdfID, document["context"])
	source := document["source"].(map[string]any)
	assert.Equal(t, "base64", source["type"])
	assert.Equal(t, "application/pdf", source["media_type"])
	assert.Equal(t, base64.StdEncoding.EncodeToString(pdfBytes), source["data"])

	text := blocks[1]["text"].(string)
	assert.NotContains(t, text, pdfText, "the file's text is not sent beside the file")
	assert.Contains(t, text, "- id: "+pdfID)
	assert.Contains(t, text, "the file itself is attached to this message")
	assert.Contains(t, text, "What does the rate confirmation pay?")
	assert.Equal(t, []string{pdfID}, resp.Files.Sent)
	assert.Empty(t, resp.Files.Fallbacks)
}

// An image has no title, so a line naming it goes before it; a file that
// could not be read for text is still sent, and is not described as failed.
func TestAnthropicAdapter_SendsAnAttachedImageAfterALineNamingIt(t *testing.T) {
	t.Parallel()

	server, bodies := filesServer(t, false)
	call := filesCall(
		server.URL,
		"claude-sonnet-5-5",
		newFileStore(),
		attachedMessage(imageAttachment()),
	)

	_, err := NewAnthropicAdapter().Complete(t.Context(), call)
	require.NoError(t, err)

	blocks := userBlocks(t, bodies()[0])
	require.Equal(t, []string{"text", "image", "text"}, blockTypes(blocks))
	assert.Equal(t, "Attached file "+imageID+", seal.png:", blocks[0]["text"])
	source := blocks[1]["source"].(map[string]any)
	assert.Equal(t, "image/png", source["media_type"])
	assert.Equal(t, base64.StdEncoding.EncodeToString(imageBytes), source["data"])
	assert.NotContains(t, blocks[2]["text"], "could not be read")
}

/*
The Anthropic protocol also fronts other vendors' models, behind gateways and
compatible endpoints, and one that refuses a document block fails the whole
request. A model this cannot read as Claude, and a router with no file reader,
are sent the message exactly as before, and nothing is read.
*/
func TestAnthropicAdapter_SendsFilesAsTextWhereTheModelMayNotReadThem(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		model string
		store *fileStore
	}{
		{name: "another vendor behind the protocol", model: "kimi-k2", store: newFileStore()},
		{name: "a Claude id this cannot read", model: "claude-latest", store: newFileStore()},
		{name: "no file reader", model: "claude-opus-5-5"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server, bodies := filesServer(t, false)
			message := attachedMessage(pdfAttachment())
			call := filesCall(server.URL, tc.model, tc.store, message)

			resp, err := NewAnthropicAdapter().Complete(t.Context(), call)
			require.NoError(t, err)

			blocks := userBlocks(t, bodies()[0])
			require.Equal(t, []string{"text"}, blockTypes(blocks))
			assert.Equal(t, message.Content, blocks[0]["text"])
			assert.Empty(t, resp.Files.Sent)
			if tc.store != nil {
				assert.Zero(t, tc.store.readsOf(pdfID))
			}
		})
	}
}

// Each file that cannot go as itself goes as its text, and the report says
// why; one that can still goes. A file plainly over a limit is never read.
func TestAnthropicAdapter_SendsAFileItCannotTakeAsItsText(t *testing.T) {
	t.Parallel()

	long := pdfAttachment()
	long.DocumentID = "doc_01LONG"
	long.PageCount = 120
	notes := agentturn.RuntimeAttachment{
		DocumentID:  docxID,
		FileName:    "notes.docx",
		ContentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		Text:        "Driver notes.",
	}
	big := imageAttachment()
	big.DocumentID = "doc_01BIGPHOTO"
	big.FileSize = 4_000_000
	locked := pdfAttachment()
	locked.DocumentID = "doc_01LOCKED"
	missing := pdfAttachment()
	missing.DocumentID = "doc_01GONE"

	store := newFileStore()
	store.files[locked.DocumentID] = &MessageFileContent{
		Data: pdfBytes, ContentType: "application/pdf", Unreadable: true,
	}
	server, bodies := filesServer(t, false)
	call := filesCall(server.URL, "claude-haiku-4-5", store,
		attachedMessage(long, notes, big, locked, missing, pdfAttachment()))

	resp, err := NewAnthropicAdapter().Complete(t.Context(), call)
	require.NoError(t, err)

	assert.Equal(t, []string{pdfID}, resp.Files.Sent)
	assert.Equal(t, []FileFallback{
		{DocumentID: "doc_01LONG", Reason: FileFallbackTooManyPage},
		{DocumentID: docxID, Reason: FileFallbackType},
		{DocumentID: "doc_01BIGPHOTO", Reason: FileFallbackTooLarge},
		{DocumentID: "doc_01LOCKED", Reason: FileFallbackUnreadable},
		{DocumentID: "doc_01GONE", Reason: FileFallbackUnavailable},
	}, resp.Files.Fallbacks)
	assert.Zero(
		t,
		store.readsOf("doc_01LONG"),
		"120 pages is past what a 200,000-token model takes",
	)
	assert.Zero(t, store.readsOf(docxID))
	assert.Zero(t, store.readsOf("doc_01BIGPHOTO"))

	blocks := userBlocks(t, bodies()[0])
	require.Equal(t, []string{"document", "text"}, blockTypes(blocks))
	text := blocks[1]["text"].(string)
	assert.Contains(t, text, "Driver notes.", "a file sent as its text keeps its text")
	assert.Equal(t, 3, strings.Count(text, pdfText),
		"the three PDFs sent as text keep theirs; only the sent one's is swapped out")
}

func TestAnthropicAdapter_KeepsEveryFileOfARequestWithinItsBudget(t *testing.T) {
	t.Parallel()

	first := pdfAttachment()
	second := pdfAttachment()
	second.DocumentID = "doc_01SECOND"
	store := newFileStore()
	store.files[second.DocumentID] = &MessageFileContent{
		Data:        make([]byte, 19_999_990),
		ContentType: "application/pdf",
	}
	server, _ := filesServer(t, false)
	call := filesCall(server.URL, "claude-opus-5-5", store, attachedMessage(first, second))

	resp, err := NewAnthropicAdapter().Complete(t.Context(), call)
	require.NoError(t, err)

	assert.Equal(t, []string{pdfID}, resp.Files.Sent)
	assert.Equal(t,
		[]FileFallback{{DocumentID: "doc_01SECOND", Reason: FileFallbackTooLarge}},
		resp.Files.Fallbacks,
		"with the first file sent, the second is past what is left of 20 MB",
	)
}

/*
A file the API refuses (damaged, or more than the model takes in a way no
check here sees) would otherwise fail the turn. The request goes again once,
as text, before anything reached the person.
*/
func TestAnthropicAdapter_SendsTheConversationAsTextWhenItsFilesAreRefused(t *testing.T) {
	t.Parallel()

	for _, stream := range []bool{false, true} {
		server, bodies := filesServer(t, true)
		message := attachedMessage(pdfAttachment(), imageAttachment())
		call := filesCall(server.URL, "claude-opus-5-5", newFileStore(), message)

		var (
			resp *Response
			err  error
		)
		if stream {
			resp, err = NewAnthropicAdapter().(Streamer).Stream(t.Context(), call, func(string) {})
		} else {
			resp, err = NewAnthropicAdapter().Complete(t.Context(), call)
		}
		require.NoError(t, err, "stream=%v", stream)

		sent := bodies()
		require.Len(t, sent, 2, "stream=%v", stream)
		retried := userBlocks(t, sent[1])
		require.Equal(t, []string{"text"}, blockTypes(retried), "stream=%v", stream)
		assert.Equal(t, message.Content, retried[0]["text"], "stream=%v", stream)
		assert.Empty(t, resp.Files.Sent, "stream=%v", stream)
		assert.Equal(t, []FileFallback{
			{DocumentID: pdfID, Reason: FileFallbackRefused},
			{DocumentID: imageID, Reason: FileFallbackRefused},
		}, resp.Files.Fallbacks, "stream=%v", stream)
	}
}

// A request with no file is not sent twice over a 400 of its own.
func TestAnthropicAdapter_DoesNotRetryARefusalWithoutFiles(t *testing.T) {
	t.Parallel()

	server, seen := recordingServer(t, http.StatusBadRequest, "application/json",
		`{"type":"error","error":{"type":"invalid_request_error","message":"bad"}}`)

	_, err := NewAnthropicAdapter().Complete(t.Context(), anthropicCall(server.URL))

	require.Error(t, err)
	assert.Len(t, seen(), 1)
}

/*
Entries are swapped by their place in the message, last first, so a file sent
after another still lands where its entry was. An entry whose place no longer
names it (the content changed after the turn opened) is left as it was: the
model then reads the file and its text, never the wrong entry replaced.
*/
func TestSwapEntries_ReplacesEachEntryWhereItStands(t *testing.T) {
	t.Parallel()

	message := attachedMessage(pdfAttachment(), imageAttachment())
	files := []*MessageFile{&message.Files[0], &message.Files[1]}

	swapped := swapEntries(message.Content, files)

	assert.NotContains(t, swapped, pdfText)
	assert.Equal(t, 2, strings.Count(swapped, "the file itself is attached to this message"))
	assert.Less(t, strings.Index(swapped, "- id: "+pdfID), strings.Index(swapped, "- id: "+imageID))

	shifted := "Earlier note. " + message.Content
	assert.Equal(t, shifted, swapEntries(shifted, files))
}
