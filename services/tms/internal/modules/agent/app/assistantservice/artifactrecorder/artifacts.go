package artifactrecorder

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/replycheck"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/artifactpayload"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/attribution"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/proposalstores"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/toolartifacts"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/analysis"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardmarkdown"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentactivityports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantartifactports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// Recorder collects what one turn produced. It saves each artifact
// as the tool that made it finishes, so the pane can open it while the reply
// is still arriving, and ties them to their messages once the turn is saved.
type Recorder struct {
	repo      assistantartifactports.AssistantArtifactRepository
	proposals proposalstores.ProposalStore
	activity  agentactivityports.AgentActivityPublisher
	logger    *zap.Logger
	ctx       context.Context
	thread    *conversation.Thread
	tenant    pagination.TenantInfo
	actor     kernelports.AuditActor
	reader    *permissionports.RequestActor
	emit      kernelports.AssistantStreamEmitter
	Recorded  []*assistantartifact.Artifact
	readBy    *attribution.AgentIdentity
	agentRead bool
	// reads is what an artifact needs beyond the tool's result: the agent
	// a document is credited to, the pages an extraction was read from.
	reads *Reads
}

type Reads struct {
	Attribution   *attribution.Attributor
	Conversations conversationports.ConversationRepository
	Runtime       *agentruntime.Service
	Boards        assistantports.BoardFiller
	Documents     documentports.DocumentRepository
	Contents      documentports.DocumentContentService
}

type Deps struct {
	Repo      assistantartifactports.AssistantArtifactRepository
	Proposals proposalstores.ProposalStore
	Activity  agentactivityports.AgentActivityPublisher
	Logger    *zap.Logger
	Reads     *Reads
}

// withAnalyses is a document's text with each analysis it names added after
// it, written from what the analysis kept: a model that retyped an analysis
// into a document copied its figures by hand, and a person who asked to
// publish the analysis got a second, unchecked copy of it.
func (r *Recorder) withAnalyses(body string, ids []pulid.ID) (string, error) {
	if len(ids) == 0 {
		return body, nil
	}
	var sb strings.Builder
	sb.WriteString(strings.TrimRight(body, "\n"))
	for _, id := range ids {
		kept, err := r.repo.GetByID(r.ctx, assistantartifactports.GetArtifactRequest{
			ID:         id,
			TenantInfo: r.tenant,
		})
		if err != nil || kept.ThreadID != r.thread.ID ||
			kept.Kind != assistantartifact.KindAnalysis {
			return "", toolartifacts.ErrUnknownAnalysis
		}
		var shown analysis.Display
		if err = jsonutils.Convert(kept.Payload, &shown); err != nil {
			return "", toolartifacts.ErrUnknownAnalysis
		}
		sb.WriteString("\n\n")
		sb.WriteString(shown.Markdown(kept.Title))
	}

	return sb.String(), nil
}

func (r *Recorder) bunch(
	observation *agentruntimeports.ToolObservation,
	card *assistantartifact.Artifact,
) (*agentruntimeports.ShownArtifact, error) {
	earlier, err := r.repo.ListByToolCalls(
		r.ctx,
		&assistantartifactports.ListArtifactsByToolCallsRequest{
			ThreadID:   r.thread.ID,
			TenantInfo: r.tenant,
			CallIDs:    observation.Earlier,
		},
	)
	if err != nil {
		r.logger.Warn("the turn's earlier cards could not be read to bunch them",
			zap.String("thread", r.thread.ID.String()),
			zap.Error(err),
		)

		return r.keep(card)
	}

	parts := toolartifacts.PartsOf(earlier, observation.Call.Name)
	if parts.Table == nil && len(parts.Cards) == 0 {
		return r.keep(card)
	}

	table := toolartifacts.BunchTable(observation.Call.Name, parts, card)
	r.noteReader(table)
	saved, err := r.save(table)
	if err != nil {
		return nil, err
	}

	r.retire(parts.Cards)

	shown := toolartifacts.ShownArtifact(saved)
	for _, card := range parts.Cards {
		shown.Replaces = append(shown.Replaces, card.ID)
	}

	return shown, nil
}

func (r *Recorder) keep(
	artifact *assistantartifact.Artifact,
) (*agentruntimeports.ShownArtifact, error) {
	saved, err := r.save(artifact)
	if err != nil {
		return nil, err
	}

	return toolartifacts.ShownArtifact(saved), nil
}

func (r *Recorder) retire(cards []*assistantartifact.Artifact) {
	if len(cards) == 0 {
		return
	}

	ids := make([]pulid.ID, 0, len(cards))
	for _, card := range cards {
		ids = append(ids, card.ID)
	}
	if err := r.repo.Delete(r.ctx, &assistantartifactports.DeleteArtifactsRequest{
		ThreadID:   r.thread.ID,
		TenantInfo: r.tenant,
		IDs:        ids,
	}); err != nil {
		r.logger.Warn("the cards a table replaced could not be removed",
			zap.String("thread", r.thread.ID.String()),
			zap.Error(err),
		)

		return
	}

	for _, id := range ids {
		r.forget(id)
		r.emit(kernelports.StreamEvent{
			Event: assistantstreamports.AssistantEventArtifactRemoved,
			Data:  assistantstreamports.AssistantArtifactRemovedEvent{ID: id},
		})
	}
}

// assignSlug names a new lineage for its link: its title as a slug, numbered
// when the conversation already has one by that name. A later version takes
// the slug of the lineage it continues.
func (r *Recorder) assignSlug(
	artifact *assistantartifact.Artifact,
	previous *assistantartifact.Artifact,
) {
	if artifact.Slug != "" {
		return
	}
	if previous != nil {
		artifact.Slug = previous.Slug
		return
	}

	base := assistantartifact.SlugBase(artifact.Title)
	taken, err := r.repo.TakenSlugs(r.ctx, r.thread.ID, r.tenant, base)
	if err != nil {
		r.logger.Warn("artifact slugs could not be read", zap.Error(err))
		taken = map[string]bool{}
	}
	artifact.Slug = assistantartifact.UniqueSlug(base, taken)
}

func (r *Recorder) continueTable(
	observation *agentruntimeports.ToolObservation,
) (*agentruntimeports.ShownArtifact, bool) {
	if observation.Failed || observation.Data == nil || observation.Call.ID == "" {
		return nil, false
	}
	at, ok := artifactpayload.PositionOf(observation.Call.Arguments)
	if !ok {
		return nil, false
	}
	document, ok := artifactpayload.ToJSONDocument(observation.Data)
	if !ok {
		return nil, false
	}
	if _, rows := document.Fields[artifactpayload.ResultItems].([]any); !rows {
		return nil, false
	}
	table := r.tableContinuedBy(&observation.Call, at)
	if table == nil {
		return nil, false
	}

	next := artifactpayload.WithPage(table, document.Fields)
	if next == nil {
		return toolartifacts.ShownArtifact(table), true
	}
	saved, err := r.store(next)
	if err != nil {
		return toolartifacts.ShownArtifact(table), true
	}

	return toolartifacts.ShownArtifact(saved), true
}

func (r *Recorder) tableContinuedBy(
	call *llmports.ToolCall,
	at artifactpayload.PagePosition,
) *assistantartifact.Artifact {
	key := assistantartifact.LineageKeyFor(
		assistantartifact.KindTableView, call.Name, call.Arguments,
	)
	for _, recorded := range slices.Backward(r.Recorded) {
		if recorded.LineageKey == key && at.Continues(recorded, call.Name) {
			return recorded
		}
	}

	latest, err := r.repo.LatestInLineage(r.ctx, assistantartifactports.LatestInLineageRequest{
		ThreadID:       r.thread.ID,
		TenantInfo:     r.tenant,
		LineageKey:     key,
		ExceptToolCall: call.ID,
	})
	if err != nil {
		r.logger.Warn("the table a page continues could not be read", zap.Error(err))
		return nil
	}
	if !at.Continues(latest, call.Name) {
		return nil
	}

	return latest
}

func Open(
	ctx context.Context,
	deps Deps,
	thread *conversation.Thread,
	tenant pagination.TenantInfo,
	actor *permissionports.RequestActor,
	emit kernelports.AssistantStreamEmitter,
) *Recorder {
	if deps.Repo == nil {
		return nil
	}
	if emit == nil {
		emit = func(kernelports.StreamEvent) {}
	}

	return &Recorder{
		repo:      deps.Repo,
		proposals: deps.Proposals,
		activity:  deps.Activity,
		logger:    deps.Logger,
		ctx:       ctx,
		thread:    thread,
		tenant:    tenant,
		actor:     actor.AuditActorOrSystem(),
		reader:    actor,
		emit:      emit,
		reads:     deps.Reads,
	}
}

// Observer is the hook the runtime calls per tool result; nil when there is
// no recorder, so the runtime skips the call altogether.
func (r *Recorder) Observer() agentruntimeports.ToolObserver {
	if r == nil {
		return nil
	}

	return r.Observe
}

//nolint:gocritic // Observe is the runtime's ToolObserver, which takes the observation by value
func (r *Recorder) Observe(
	observation agentruntimeports.ToolObservation,
) (*agentruntimeports.ShownArtifact, error) {
	if document, ok := observation.Data.(agentruntimeports.PublishedDocument); ok {
		return r.publish(observation.Call.ID, &document)
	}
	if request, ok := observation.Data.(agentruntimeports.DecisionRequest); ok {
		return r.requestDecision(observation.Call.ID, request)
	}
	if request, ok := observation.Data.(agentruntimeports.BoardRequest); ok {
		return r.showBoard(observation.Call.ID, request)
	}
	if observation.Call.Name == artifactpayload.ToolGetShipmentDraft && !observation.Failed {
		if extraction := r.extraction(observation.Call.ID, observation.Data); extraction != nil {
			return r.shownIfKept(extraction), nil
		}
	}

	if action := observation.Action; action != nil && action.Executed && !observation.Failed {
		if saved := r.Written(
			observation.Call.ID, observation.Call.Name, action.ExecutionResult,
		); saved != nil {
			return toolartifacts.ShownArtifact(saved), nil
		}
	}

	if shown, continued := r.continueTable(&observation); continued {
		return shown, nil
	}

	artifact := toolartifacts.ArtifactFromObservation(observation)
	if artifact == nil {
		return nil, nil //nolint:nilnil // a ToolObserver answers nil, nil for a result nothing shows
	}
	if artifact.Kind == assistantartifact.KindNavigation {
		r.announceNavigation(artifact)
		return nil, nil //nolint:nilnil // a ToolObserver answers nil, nil for a result nothing shows
	}
	artifact.LineageKey = assistantartifact.LineageKeyFor(
		artifact.Kind,
		observation.Call.Name,
		observation.Call.Arguments,
	)
	artifactpayload.NoteReadWith(artifact, observation.RanWith)
	r.noteReader(artifact)
	if artifact.Kind == assistantartifact.KindEntityCard && len(observation.Earlier) > 0 {
		return r.bunch(&observation, artifact)
	}

	return r.shownIfKept(artifact), nil
}

func (r *Recorder) shownIfKept(
	artifact *assistantartifact.Artifact,
) *agentruntimeports.ShownArtifact {
	shown, err := r.keep(artifact)
	if err != nil {
		return nil
	}

	return shown
}

// announceNavigation tells the app to move to the page the agent opened. It
// is a move, not something to keep: the reply names the page as a link, and
// nothing is added beside the conversation.
func (r *Recorder) announceNavigation(artifact *assistantartifact.Artifact) {
	r.emit(kernelports.StreamEvent{
		Event: assistantstreamports.AssistantEventArtifact,
		Data: assistantstreamports.AssistantArtifactEvent{
			ID:               pulid.MustNew("art_"),
			Kind:             artifact.Kind,
			Status:           artifact.Status,
			Title:            artifact.Title,
			SourceToolCallID: artifact.SourceToolCallID,
			Path:             toolartifacts.NavigationPath(artifact),
		},
	})
}

// publish keeps a document the model wrote. A revision is the next version
// of a document this conversation already holds: the pane keeps one brief
// with its versions, and the text it replaced can be read and restored.
func (r *Recorder) publish(
	callID string,
	document *agentruntimeports.PublishedDocument,
) (*agentruntimeports.ShownArtifact, error) {
	body, err := r.withAnalyses(document.Body, document.Analyses)
	if err != nil {
		return nil, err
	}
	document.Body = body
	artifact := toolartifacts.DocumentArtifact(callID, document, r.author())

	if !document.ArtifactID.IsNil() {
		existing, getErr := r.repo.GetByID(r.ctx, assistantartifactports.GetArtifactRequest{
			ID:         document.ArtifactID,
			TenantInfo: r.tenant,
		})
		if getErr != nil || existing.ThreadID != r.thread.ID ||
			existing.Kind != assistantartifact.KindDocument {
			return nil, toolartifacts.ErrUnknownDocument
		}
		latest := r.latestVersion(existing)
		artifact.FollowLineage(latest)
		artifact.Slug = latest.Slug
		artifact.Pinned = latest.Pinned
		artifact.Payload[assistantartifact.DocumentVersionNote] = "Revised"
	}

	saved, err := r.save(artifact)
	if err != nil {
		return nil, toolartifacts.ErrDocumentNotKept
	}

	return toolartifacts.ShownArtifact(saved), nil
}

// latestVersion is the newest version of the lineage an artifact belongs to,
// or the artifact itself when its lineage cannot be read.
func (r *Recorder) latestVersion(
	artifact *assistantartifact.Artifact,
) *assistantartifact.Artifact {
	versions, err := r.repo.ListLineage(r.ctx, assistantartifactports.LineageRequest{
		ThreadID:   r.thread.ID,
		TenantInfo: r.tenant,
		ID:         artifact.ID,
	})
	if err != nil || len(versions) == 0 {
		return artifact
	}

	latest := versions[0]
	for _, version := range versions[1:] {
		if version.LineageSeq > latest.LineageSeq {
			latest = version
		}
	}

	return latest
}

// author is the name a document is credited to: the conversation's agent.
func (r *Recorder) author() string {
	if r.reads == nil {
		return ""
	}

	return r.reads.Attribution.AgentName(r.ctx, conversationports.GetThreadRequest{
		ID:         r.thread.ID,
		TenantInfo: r.tenant,
	}, r.agentThread())
}

func (r *Recorder) agentThread() *conversation.Thread {
	if r.thread.AgentDefinitionID.IsNotNil() || r.agentRead || r.reads == nil ||
		r.reads.Conversations == nil || r.reader == nil {
		return r.thread
	}
	r.agentRead = true
	thread, err := r.reads.Conversations.GetThread(r.ctx, conversationports.GetThreadRequest{
		ID:         r.thread.ID,
		UserID:     r.reader.UserID,
		TenantInfo: r.tenant,
	})
	if err != nil || thread == nil {
		return r.thread
	}
	r.thread.AgentDefinitionID = thread.AgentDefinitionID

	return r.thread
}

// Adopt takes on artifacts kept earlier in the turn, where the tools that made
// them ran, so they are tied to their messages and reported with the rest.
func (r *Recorder) Adopt(artifacts []*assistantartifact.Artifact) {
	if r == nil {
		return
	}

	retired := toolartifacts.BunchedCalls(artifacts)
	for _, artifact := range artifacts {
		if _, gone := retired[artifact.SourceToolCallID]; gone &&
			artifact.Kind == assistantartifact.KindEntityCard {
			continue
		}
		r.remember(artifact)
	}
}

// FromProposals views the turn's recorded proposals as drafts and its plan
// as a checklist, so a person decides them where they can read them.
func (r *Recorder) FromProposals(
	proposals []*agent.AgentProposal,
	plan *agentplan.AgentPlan,
) {
	if r == nil {
		return
	}

	for _, proposal := range proposals {
		if proposal != nil && proposal.Status == agent.ProposalStatusSuperseded {
			continue
		}
		if artifact := toolartifacts.DraftArtifact(proposal); artifact != nil {
			r.shownIfKept(artifact)
		}
	}

	if plan != nil {
		r.shownIfKept(toolartifacts.PlanArtifact(plan, proposals))
	}
}

// AttachMessages ties each artifact made by a tool call to the saved
// assistant message that asked for it, once the turn is on disk.
func (r *Recorder) AttachMessages(index map[string]pulid.ID) {
	if r == nil {
		return
	}

	for _, artifact := range r.Recorded {
		if !artifact.MessageID.IsNil() || artifact.SourceToolCallID == "" {
			continue
		}
		messageID, ok := index[artifact.SourceToolCallID]
		if !ok {
			continue
		}
		artifact.MessageID = messageID
		if _, err := r.repo.Upsert(r.ctx, artifact); err != nil {
			r.logger.Warn("could not tie an artifact to its message",
				zap.String("artifact", artifact.ID.String()),
				zap.Error(err),
			)
		}
	}
}

// KeepLinked drops the views of lookups the reply did not point to. A lookup
// is how the agent finds things out; its result is worth a place beside the
// conversation only when the reply uses it to answer. Documents, drafts,
// plans and decision requests were made on purpose and always stay, and so
// does a table the reply was told to point to rather than reprint: a long
// list, a ranking, a queue to work from or a live view. A model that
// summarized sixty shipments and forgot the link used to lose the table
// with the summary; the reader draws a badge for a kept table the reply
// did not name, so keeping it costs nothing.
func (r *Recorder) KeepLinked(linked map[string]bool) {
	if r == nil {
		return
	}

	var unused []pulid.ID
	for _, artifact := range r.Recorded {
		if !assistantartifact.IsLookup(artifact.Kind) || linked[artifact.ID.String()] {
			continue
		}
		if toolartifacts.MustBePointedTo(artifact) {
			continue
		}
		unused = append(unused, artifact.ID)
	}
	if len(unused) == 0 {
		return
	}
	if err := r.repo.Delete(r.ctx, &assistantartifactports.DeleteArtifactsRequest{
		ThreadID:   r.thread.ID,
		TenantInfo: r.tenant,
		IDs:        unused,
	}); err != nil {
		r.logger.Warn("could not drop the artifacts the reply did not use", zap.Error(err))
		return
	}
	for _, id := range unused {
		r.forget(id)
	}
}

func (r *Recorder) NameLinks(messages []conversation.Message, reply *string) {
	if r == nil {
		return
	}

	titles := make(map[string]string, len(r.Recorded))
	for _, artifact := range r.Recorded {
		titles[artifact.ID.String()] = artifact.Title
	}
	lookedUp := true
	if earlier := r.unrecordedLinks(messages, titles); len(earlier) > 0 {
		found, err := r.repo.ListTitles(r.ctx, &assistantartifactports.ArtifactTitlesRequest{
			ThreadID:   r.thread.ID,
			TenantInfo: r.tenant,
			IDs:        earlier,
		})
		if err != nil {
			r.logger.Warn("could not read the titles of the artifacts a reply names",
				zap.Error(err))
			lookedUp = false
		}
		for id, title := range found {
			titles[id.String()] = title
		}
	}

	titleOf := func(id, words string) (string, bool) {
		if title, ok := titles[id]; ok {
			return title, true
		}

		return words, !lookedUp
	}
	for idx := range messages {
		if messages[idx].Role == conversation.RoleAssistant {
			messages[idx].Content = replycheck.NameArtifactLinks(messages[idx].Content, titleOf)
		}
	}
	if reply != nil {
		*reply = replycheck.NameArtifactLinks(*reply, titleOf)
	}
}

func (r *Recorder) unrecordedLinks(
	messages []conversation.Message,
	titles map[string]string,
) []pulid.ID {
	var ids []pulid.ID
	seen := make(map[string]bool, 4)
	for idx := range messages {
		if messages[idx].Role != conversation.RoleAssistant {
			continue
		}
		for _, raw := range replycheck.ArtifactRefIDs(messages[idx].Content) {
			if _, recorded := titles[raw]; recorded || seen[raw] {
				continue
			}
			seen[raw] = true
			if id, err := pulid.Parse(raw); err == nil {
				ids = append(ids, id)
			}
		}
	}

	return ids
}

// Artifacts is what the turn produced, for the turn's result.
func (r *Recorder) Artifacts() []assistantartifactports.AssistantArtifact {
	if r == nil || len(r.Recorded) == 0 {
		return nil
	}

	out := make([]assistantartifactports.AssistantArtifact, 0, len(r.Recorded))
	for _, artifact := range r.Recorded {
		out = append(out, artifactpayload.ToAssistantArtifact(artifact))
	}

	return out
}

// followLineage makes an artifact the next version of the one the same
// lookup made before, and returns that earlier one when there is one.
func (r *Recorder) followLineage(
	artifact *assistantartifact.Artifact,
) *assistantartifact.Artifact {
	if artifact.LineageKey == "" {
		return nil
	}
	previous, err := r.repo.LatestInLineage(r.ctx, assistantartifactports.LatestInLineageRequest{
		ThreadID:       r.thread.ID,
		TenantInfo:     r.tenant,
		LineageKey:     artifact.LineageKey,
		ExceptToolCall: artifact.SourceToolCallID,
	})
	if err != nil {
		r.logger.Warn("artifact lineage could not be read", zap.Error(err))
		return nil
	}
	artifact.FollowLineage(previous)

	return previous
}

func (r *Recorder) save(
	artifact *assistantartifact.Artifact,
) (*assistantartifact.Artifact, error) {
	artifact.ThreadID = r.thread.ID
	artifact.OrganizationID = r.tenant.OrgID
	artifact.BusinessUnitID = r.tenant.BuID
	artifact.Title = assistantartifact.NormalizeTitle(artifact.Title)

	multiErr := errortypes.NewMultiError()
	artifact.Validate(multiErr)
	if multiErr.HasErrors() {
		r.logger.Error("artifact skipped: invalid",
			zap.String("kind", string(artifact.Kind)),
			zap.Error(multiErr),
		)

		return nil, multiErr
	}

	// The same lookup run again over data that has not changed shows what the
	// last version shows; the reply points to that one instead of a copy.
	previous := r.followLineage(artifact)
	if toolartifacts.SameView(previous, artifact) {
		return previous, nil
	}
	r.assignSlug(artifact, previous)

	return r.store(artifact)
}

func (r *Recorder) store(
	artifact *assistantartifact.Artifact,
) (*assistantartifact.Artifact, error) {
	saved, err := r.repo.Upsert(r.ctx, artifact)
	if err != nil {
		r.logger.Error("artifact could not be kept",
			zap.String("thread", r.thread.ID.String()),
			zap.String("kind", string(artifact.Kind)),
			zap.Error(err),
		)

		return nil, err
	}

	r.remember(saved)
	if r.activity != nil {
		r.activity.ArtifactChanged(r.ctx, saved, r.actor, agentactivityports.ActivityUpdated)
	}
	r.emit(kernelports.StreamEvent{
		Event: assistantstreamports.AssistantEventArtifact,
		Data: assistantstreamports.AssistantArtifactEvent{
			ID:               saved.ID,
			Kind:             saved.Kind,
			Status:           saved.Status,
			Title:            saved.Title,
			SourceToolCallID: saved.SourceToolCallID,
			Path:             toolartifacts.NavigationPath(saved),
			Draft:            toolartifacts.DraftEditOf(saved),
		},
	})

	return saved, nil
}

// remember adds an artifact to what the turn produced, replacing an earlier
// entry for the same artifact: a document revised twice in one turn is one
// artifact, not two.
func (r *Recorder) remember(saved *assistantartifact.Artifact) {
	for idx, recorded := range r.Recorded {
		if recorded.ID == saved.ID {
			r.Recorded[idx] = saved
			return
		}
	}
	r.Recorded = append(r.Recorded, saved)
}

// forget drops an artifact from what the turn produced, once a table has
// taken its place: the turn reports the table, and the message the card
// came from is tied to nothing that no longer exists.
func (r *Recorder) forget(id pulid.ID) {
	r.Recorded = slices.DeleteFunc(r.Recorded, func(recorded *assistantartifact.Artifact) bool {
		return recorded.ID == id
	})
}

func (r *Recorder) requestDecision(
	callID string,
	request agentruntimeports.DecisionRequest,
) (*agentruntimeports.ShownArtifact, error) {
	if r.proposals == nil {
		return nil, toolartifacts.ErrDecisionUnavailable
	}

	stored, err := r.proposals.ListByThread(
		r.ctx,
		agentproposalports.ListAgentProposalsByThreadRequest{
			ThreadID:   r.thread.ID,
			TenantInfo: r.tenant,
		},
	)
	if err != nil {
		r.logger.Warn("the proposal a decision was asked for could not be read",
			zap.String("thread", r.thread.ID.String()),
			zap.String("proposal", request.ProposalID.String()),
			zap.Error(err),
		)

		return nil, toolartifacts.ErrDecisionUnavailable
	}

	artifact, err := toolartifacts.RequestedArtifact(callID, request, stored)
	if err != nil {
		return nil, err
	}

	saved, err := r.save(artifact)
	if err != nil {
		return nil, toolartifacts.ErrDecisionNotShown
	}

	return toolartifacts.ShownArtifact(saved), nil
}

func (r *Recorder) showBoard(
	callID string,
	request agentruntimeports.BoardRequest,
) (*agentruntimeports.ShownArtifact, error) {
	boards := r.boards()
	if boards == nil || r.reader == nil {
		return nil, ErrNoBoards
	}
	fill := &assistantports.BoardFillRequest{
		Actor:    r.reader,
		ThreadID: r.thread.ID,
		Board:    &request.Board,
	}
	if err := boards.Check(r.ctx, fill); err != nil {
		return nil, err
	}

	var latest *assistantartifact.Artifact
	if request.ArtifactID.IsNotNil() {
		existing, err := r.repo.GetByID(r.ctx, assistantartifactports.GetArtifactRequest{
			ID:         request.ArtifactID,
			TenantInfo: r.tenant,
		})
		if err != nil || existing.ThreadID != r.thread.ID ||
			existing.Kind != assistantartifact.KindBoard {
			return nil, ErrUnknownBoard
		}
		latest = r.latestVersion(existing)
	}

	filled, err := boards.Fill(r.ctx, fill)
	if err != nil {
		r.logger.Error("board could not be filled", zap.Error(err))
		return nil, errBoardStore
	}
	author := r.author()
	payload := &board.Payload{Board: request.Board, Snapshot: filled.Snapshot(), Author: author}
	fields, err := payload.Map()
	if err != nil {
		return nil, errBoardStore
	}

	artifact := &assistantartifact.Artifact{
		Kind:             assistantartifact.KindBoard,
		Status:           assistantartifact.StatusReady,
		Title:            assistantartifact.NormalizeTitle(request.Board.Title),
		Payload:          fields,
		SourceToolCallID: callID,
	}
	if latest != nil {
		artifact.FollowLineage(latest)
		artifact.Slug = latest.Slug
		artifact.Pinned = latest.Pinned
	}
	saved, err := r.save(artifact)
	if err != nil {
		return nil, errBoardStore
	}

	shown := toolartifacts.ShownArtifact(saved)
	shown.Brief = boardmarkdown.Markdown(filled, boardmarkdown.MarkdownMeta{
		Author:     author,
		AnsweredAt: saved.CreatedAt,
		Location:   boards.Location(r.ctx, r.tenant),
	})

	return shown, nil
}

func (r *Recorder) boards() assistantports.BoardFiller {
	if r.reads == nil {
		return nil
	}

	return r.reads.Boards
}
