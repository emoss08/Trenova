package conversationrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact/assistantartifactdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation/conversationdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
)

var deskColumns = []string{
	"kind", "id", "thread_id", "agent_id", "title", "thread_title", "artifact_kind", "status", "at",
}

type deskTerms struct {
	req      assistantports.SearchDeskRequest
	term     string
	contains string
}

type deskQuery func(terms *deskTerms) keel.Projection

func deskThreads(t *conversationdb.ThreadsTable, terms *deskTerms) keel.Pred {
	return keel.And(
		t.UserID.Eq(terms.req.UserID),
		t.Origin.NotIn(conversation.UnlistedOrigins()...),
	)
}

func latestFirst(cte *keel.CTETable, limit int) keel.Projection {
	cols := make([]keel.Expr, 0, len(deskColumns))
	for _, name := range deskColumns {
		cols = append(cols, cte.Col(name))
	}

	return cte.Query().With(cte).Pick(cols...).Order(cte.Col("at").Desc()).Limit(limit)
}

func deskChats(terms *deskTerms) keel.Projection {
	t := conversationdb.Threads
	at := t.LastMessageAt.Greatest(t.CreatedAt)

	return t.Pick(
		keel.Text("chat"),
		t.ID,
		t.ID,
		t.AgentDefinitionID,
		t.Title.Coalesce(""),
		keel.Text(""),
		keel.Text(""),
		t.Status,
		at,
	).
		Where(
			deskThreads(t, terms),
			keel.When(terms.term != "", t.Title.ILike(terms.contains)),
		).
		Order(t.Pinned.Desc(), at.Desc()).
		Limit(terms.req.LimitPerKind)
}

func deskChapters(terms *deskTerms) keel.Projection {
	c := conversationdb.Chapters
	t := conversationdb.Threads

	return c.Join(t, t.ID.EqCol(c.ThreadID)).
		Pick(
			keel.Text("chap"),
			c.ID,
			t.ID,
			t.AgentDefinitionID,
			c.Title,
			t.Title.Coalesce(""),
			keel.Text(""),
			c.TitleSource,
			c.CreatedAt,
		).
		Where(
			deskThreads(t, terms),
			keel.When(terms.term != "", c.Title.ILike(terms.contains)),
		).
		Order(c.CreatedAt.Desc()).
		Limit(terms.req.LimitPerKind)
}

func deskMessages(terms *deskTerms) keel.Projection {
	m := conversationdb.Messages
	t := conversationdb.Threads
	digest := m.Content.MD5()

	return latestFirst(keel.CTE("distinct_messages", deskColumns, m.
		Join(t, t.ID.EqCol(m.ThreadID)).
		DistinctOn(m.ThreadID, digest).
		Pick(
			keel.Text("msg"),
			m.ID,
			t.ID,
			m.AgentDefinitionID.Coalesce(t.AgentDefinitionID),
			m.Content.Left(4000),
			t.Title.Coalesce(""),
			keel.Text(""),
			m.Role,
			m.CreatedAt,
		).
		Where(
			deskThreads(t, terms),
			m.Role.In(conversation.RoleUser, conversation.RoleAssistant),
			m.Content.Ne(""),
			keel.When(terms.term != "", m.Content.ILike(terms.contains)),
		).
		Order(m.ThreadID.Asc(), digest.Asc(), m.CreatedAt.Desc())), terms.req.LimitPerKind)
}

func deskArtifacts(terms *deskTerms) keel.Projection {
	a := assistantartifactdb.Artifacts
	t := conversationdb.Threads
	lineage := a.LineageID.Coalesce(a.ID)

	return latestFirst(keel.CTE("latest", deskColumns, a.
		Join(t, t.ID.EqCol(a.ThreadID)).
		DistinctOn(lineage).
		Pick(
			keel.Text("art"),
			a.ID,
			t.ID,
			t.AgentDefinitionID,
			a.Title,
			t.Title.Coalesce(""),
			a.Kind,
			a.Status,
			a.CreatedAt,
		).
		Where(
			deskThreads(t, terms),
			a.Kind.Ne(assistantartifact.KindDecisionRequest),
			keel.When(terms.term != "", a.Title.ILike(terms.contains)),
		).
		Order(lineage.Asc(), a.LineageSeq.Desc(), a.CreatedAt.Desc())), terms.req.LimitPerKind)
}

func deskDecisions(terms *deskTerms) keel.Projection {
	a := assistantartifactdb.Artifacts
	t := conversationdb.Threads
	p := agentdb.AgentProposals

	return a.Join(t, t.ID.EqCol(a.ThreadID)).
		LeftJoin(p, p.ID.EqCol(a.ProposalID)).
		Pick(
			keel.Text("dec"),
			a.ID,
			t.ID,
			t.AgentDefinitionID,
			a.Title,
			t.Title.Coalesce(""),
			a.Kind,
			keel.CastText(p.Status).Coalesce(a.Status),
			a.CreatedAt,
		).
		Where(
			deskThreads(t, terms),
			a.Kind.Eq(assistantartifact.KindDecisionRequest),
			keel.When(terms.term != "", a.Title.ILike(terms.contains)),
		).
		Order(a.CreatedAt.Desc()).
		Limit(terms.req.LimitPerKind)
}

var deskSearchQueries = map[string]deskQuery{
	"chat": deskChats,
	"chap": deskChapters,
	"msg":  deskMessages,
	"art":  deskArtifacts,
	"dec":  deskDecisions,
}

func (r *repository) SearchDesk(
	ctx context.Context,
	req assistantports.SearchDeskRequest,
) ([]assistantports.DeskSearchRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]assistantports.DeskSearchRow, error) {
		if req.LimitPerKind <= 0 {
			return []assistantports.DeskSearchRow{}, nil
		}
		terms := &deskTerms{req: req, term: strings.TrimSpace(req.Query)}
		if terms.term != "" {
			terms.contains = "%" + likeEscape(terms.term) + "%"
		}
		results := make([][]assistantports.DeskSearchRow, 0, len(req.Kinds))
		items := make([]keel.BatchItem, 0, len(req.Kinds))
		for _, kind := range req.Kinds {
			query, ok := deskSearchQueries[kind]
			if !ok {
				continue
			}
			results = append(results, nil)
			items = append(items, keel.IntoSlice(query(terms), &results[len(results)-1]))
		}
		if len(items) == 0 {
			return []assistantports.DeskSearchRow{}, nil
		}
		if err := keel.Batch(ctx, r.db.Keel(ctx), items...); err != nil {
			return nil, fmt.Errorf("search desk: %w", err)
		}
		out := make([]assistantports.DeskSearchRow, 0, len(req.Kinds)*req.LimitPerKind)
		for _, rows := range results {
			out = append(out, rows...)
		}
		return out, nil
	})
}

func (r *repository) CountQuestionsSince(
	ctx context.Context,
	req assistantports.CountQuestionsSinceRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		m := conversationdb.Messages
		t := conversationdb.Threads
		count, err := m.Join(t, t.ID.EqCol(m.ThreadID)).
			Where(
				t.UserID.Eq(req.UserID),
				m.Role.Eq(conversation.RoleUser),
				m.Kind.Eq(conversation.MessageKindMessage),
				m.CreatedAt.Gte(req.Since),
			).
			Count(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, fmt.Errorf("count questions: %w", err)
		}
		return int(count), nil
	})
}
