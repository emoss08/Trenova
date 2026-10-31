package deskbench

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/deskbench/benchlive"
	"github.com/emoss08/trenova/internal/deskbench/benchrecord"
	"github.com/emoss08/trenova/internal/deskbench/benchscenario"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/paletteports"
	"github.com/emoss08/trenova/shared/pulid"
)

// OpenPalette is a scenario asked from the command palette. A palette
// question is not a Desk conversation: it is answered on the Nova harness and
// saved nowhere, so the bench gives it a stand-in thread only to key its
// records by.
func (s *Session) OpenPalette(
	_ context.Context,
	label, title, _ string,
) (*Conversation, error) {
	return &Conversation{
		Label:   label,
		session: s,
		Thread: &conversation.Thread{
			ID:    pulid.MustNew("athr_"),
			Title: BenchTitlePrefix + strings.TrimSpace(title),
		},
		palette: true,
	}, nil
}

// askPalette asks one question through the palette and records its answer
// from the stream the palette itself reads.
func (c *Conversation) askPalette(
	ctx context.Context,
	u *benchscenario.Utterance,
) ([]*benchrecord.TurnRecord, error) {
	bench := c.session.bench
	tenant := c.session.Tenant
	tenant.UserID = c.session.User.ID

	record := &benchrecord.TurnRecord{
		ThreadID:  c.Thread.ID,
		Origin:    conversationturn.AssistantTurnOriginPerson,
		Input:     u.Content,
		StartedAt: time.Now(),
		Status:    conversation.AssistantTurnStatusRunning,
	}
	ask, err := bench.Palette.Ask(ctx, &paletteports.AskRequest{
		Question: paletteports.Question{
			Content:  u.Content,
			Page:     u.Page,
			Mentions: u.Mentions,
		},
		TenantInfo: tenant,
	}, &c.session.Actor)
	if err != nil {
		return nil, err
	}
	record.TurnID = ask.ID

	live := bench.live.Turn(c.Label)
	live.Linef("== palette %s ==", ask.ID)
	live.Linef("person: %s", u.Content)
	collector := newEventCollector(live)
	answer := &paletteAnswer{}

	ref := paletteports.AskRef{AskID: ask.ID, TenantInfo: tenant}
	runCtx, cancel := context.WithTimeout(ctx, TurnTimeout)
	relayErr := bench.Palette.Relay(runCtx, &paletteports.RelayRequest{
		Ref: ref,
		OnFrame: func(frame assistantstreamports.TurnStreamFrame) error {
			answer.read(frame)

			return collector.onFrame(frame)
		},
	})
	timedOut := errors.Is(runCtx.Err(), context.DeadlineExceeded)
	cancel()
	if timedOut {
		if stopErr := bench.Palette.Stop(ctx, ref); stopErr != nil {
			record.CaptureError = "stop: " + stopErr.Error()
		}
	}
	if relayErr != nil && !errors.Is(relayErr, context.Canceled) {
		record.CaptureError = benchrecord.JoinErrors(record.CaptureError, "relay: "+relayErr.Error())
	}

	record.FinishedAt = time.Now()
	collector.into(record)
	record.Reply = answer.reply()
	record.Status, record.Failure = answer.outcome(timedOut)

	bench.Measure(ctx, c.session, record, record.StartedAt.Add(-WindowLead))
	benchlive.AnnounceTurnEnd(live, record)
	c.turns = append(c.turns, record)

	return []*benchrecord.TurnRecord{record}, nil
}

// paletteAnswer gathers the answer as the stream writes it: text before a
// round of reads is set aside, and what streams after the last round is the
// answer.
type paletteAnswer struct {
	text    strings.Builder
	done    bool
	failure string
}

func (a *paletteAnswer) read(frame assistantstreamports.TurnStreamFrame) {
	switch frame.Event {
	case assistantstreamports.AssistantEventDelta:
		var delta assistantstreamports.AssistantDeltaEvent
		if sonic.Unmarshal(frame.Data, &delta) == nil {
			a.text.WriteString(delta.Text)
		}
	case assistantstreamports.AssistantEventMessage:
		a.text.Reset()
	case assistantstreamports.AssistantEventDone:
		a.done = true
	case assistantstreamports.AssistantEventError:
		var failed struct {
			Message string `json:"message"`
		}
		if sonic.Unmarshal(frame.Data, &failed) == nil {
			a.failure = failed.Message
		}
	}
}

func (a *paletteAnswer) reply() string {
	return strings.TrimSpace(a.text.String())
}

func (a *paletteAnswer) outcome(timedOut bool) (conversation.AssistantTurnStatus, string) {
	switch {
	case a.done:
		return conversation.AssistantTurnStatusCompleted, ""
	case timedOut:
		return conversation.AssistantTurnStatusFailed, "the answer did not arrive in time"
	case a.failure != "":
		return conversation.AssistantTurnStatusFailed, a.failure
	default:
		return conversation.AssistantTurnStatusFailed, "the answer ended without finishing"
	}
}
