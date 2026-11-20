package sim

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/journal"
	"github.com/emoss08/trenova/edi-partner-sim/internal/ladder"
	"github.com/emoss08/trenova/edi-partner-sim/internal/oracle"
	"github.com/emoss08/trenova/edi-partner-sim/internal/runlog"
	"github.com/emoss08/trenova/edi-partner-sim/internal/transcript"
)

func (s *Server) Runs() *runlog.Store {
	return s.runs
}

func (s *Server) notifyRun(n runlog.Notice) {
	s.journal.Events().PublishRun(n.Kind, n.Partner, &journal.RunNotice{
		RunID:    n.RunID,
		Scenario: n.Scenario,
		Keys:     n.Keys,
	})
	s.changed()
}

func (s *Server) runsHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /control/runs", s.handleRuns)
	mux.HandleFunc("POST /control/runs", s.handleStartRun)
	mux.HandleFunc("GET /control/runs/{id}", s.handleRun)
	mux.HandleFunc("POST /control/runs/{id}", s.handleFinishRun)
	mux.HandleFunc("POST /control/runs/{id}/scenarios", s.handlePutScenario)
	mux.HandleFunc("POST /control/runs/{id}/steps", s.handlePutStep)
	return mux
}

func (s *Server) handleRuns(w http.ResponseWriter, _ *http.Request) {
	payload := map[string]any{"runs": s.runs.Summaries(), "current": nil}
	if current, ok := s.runs.Current(); ok {
		payload["current"] = current
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) handleStartRun(w http.ResponseWriter, r *http.Request) {
	start := new(runlog.Start)
	if !decode(w, r, start) {
		return
	}
	summary, err := s.runs.Start(*start)
	if err != nil {
		writeRunFailure(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, summary)
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	run, ok := s.runs.Get(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "no such run")
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (s *Server) handleFinishRun(w http.ResponseWriter, r *http.Request) {
	finish := new(runlog.Finish)
	if !decode(w, r, finish) {
		return
	}
	summary, err := s.runs.Finish(r.PathValue("id"), *finish)
	if err != nil {
		writeRunFailure(w, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) handlePutScenario(w http.ResponseWriter, r *http.Request) {
	scenario := new(runlog.Scenario)
	if !decode(w, r, scenario) {
		return
	}
	if _, ok := s.peer(scenario.Partner); !ok {
		writeError(w, http.StatusNotFound, "no such partner: "+scenario.Partner)
		return
	}
	if err := s.runs.PutScenario(r.PathValue("id"), scenario); err != nil {
		writeRunFailure(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handlePutStep(w http.ResponseWriter, r *http.Request) {
	post := new(runlog.StepPost)
	if !decode(w, r, post) {
		return
	}
	if err := s.runs.PutStep(r.PathValue("id"), post); err != nil {
		writeRunFailure(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func writeRunFailure(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, runlog.ErrNoRun), errors.Is(err, runlog.ErrNoScenario):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, runlog.ErrInvalid):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

type conversationView struct {
	journal.Conversation
	Scenario *scenarioRef `json:"scenario,omitempty"`
}

type scenarioRef struct {
	RunID   string `json:"runId"`
	Slug    string `json:"slug"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	Waiting bool   `json:"waiting,omitempty"`
}

func (s *Server) conversations(name string) []conversationView {
	listed := s.journal.Conversations(name)
	out := make([]conversationView, 0, len(listed))
	for index := range listed {
		view := conversationView{Conversation: listed[index]}
		if attached, ok := s.runs.For(name, listed[index].Key); ok {
			view.Scenario = &scenarioRef{
				RunID:   attached.Run.ID,
				Slug:    attached.Scenario.Slug,
				Name:    attached.Scenario.Name,
				Status:  attached.Scenario.Status,
				Waiting: attached.Scenario.Waiting,
			}
		}
		out = append(out, view)
	}
	return out
}

type ConversationDetail struct {
	Partner  string            `json:"partner"`
	Key      string            `json:"key"`
	Rows     []ladder.Row      `json:"rows"`
	Messages []journal.Message `json:"messages"`
	Run      *runlog.Summary   `json:"run"`
	Scenario *runlog.Scenario  `json:"scenario"`
}

func (s *Server) Conversation(name, key string) (ConversationDetail, error) {
	if _, ok := s.peer(name); !ok {
		return ConversationDetail{}, fmt.Errorf("%w: %s", ErrUnknownPartner, name)
	}
	detail := ConversationDetail{
		Partner:  name,
		Key:      key,
		Messages: s.journal.Messages(name, journal.Filter{Key: key}),
	}
	if attached, ok := s.runs.For(name, key); ok {
		detail.Run, detail.Scenario = &attached.Run, &attached.Scenario
	}
	detail.Rows = ladder.Compose(key, detail.Messages, detail.Scenario)
	if detail.Rows == nil {
		detail.Rows = []ladder.Row{}
	}
	return detail, nil
}

func (s *Server) handleConversation(w http.ResponseWriter, r *http.Request) {
	detail, err := s.Conversation(r.PathValue("partner"), r.PathValue("key"))
	if err != nil {
		writeFailure(w, err)
		return
	}
	if len(detail.Messages) == 0 && detail.Scenario == nil {
		writeError(w, http.StatusNotFound, "no conversation about "+detail.Key)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

type Imported struct {
	Direction journal.Direction
	Purpose   journal.Purpose
	State     journal.State
	At        time.Time
	Raw       string
	Variation string
	Error     string
}

func (s *Server) Import(name string, in *Imported) (journal.Message, error) {
	p, ok := s.peer(name)
	if !ok {
		return journal.Message{}, fmt.Errorf("%w: %s", ErrUnknownPartner, name)
	}
	at := in.At.UTC()
	m := &journal.Message{
		Partner:     name,
		Direction:   in.Direction,
		Purpose:     in.Purpose,
		State:       in.State,
		At:          at,
		DeliveredAt: &at,
		Transport:   p.transport(),
		Raw:         in.Raw,
		Variation:   in.Variation,
		Error:       in.Error,
		Sets:        []string{},
	}
	judged := in.Direction == journal.FromTrenova
	var (
		judgement *oracle.Judgement
		err       error
	)
	if judged {
		judgement, err = p.guides.Judge([]byte(in.Raw))
	} else {
		judgement, err = describe([]byte(in.Raw))
	}
	if err != nil {
		m.State, m.Error = journal.StateUnreadable, err.Error()
		return s.journal.Add("message.loaded", m), nil
	}
	if judged {
		m.Judgement = judgement
		accepted := judgement.Accepted()
		m.Accepted = &accepted
	}
	fill(m, judgement)
	m.Documents = transcript.Build(judgement, transcript.Options{
		Judged:    judged,
		Variation: in.Variation,
		Earlier:   s.earlierCharge(name),
	})
	targets := s.importTargets(m)
	recorded := s.journal.Add("message.loaded", m)
	s.journal.LinkAck(recorded.ID, targets...)
	s.changed()
	final, _ := s.journal.Get(recorded.ID)
	return final, nil
}

func (s *Server) importTargets(m *journal.Message) []string {
	var targets []string
	for index := range m.Documents {
		for _, acked := range m.Documents[index].Acknowledges {
			target, found := s.journal.Find(m.Partner, func(c *journal.Message) bool {
				if c.Direction == m.Direction {
					return false
				}
				return slices.ContainsFunc(c.Documents, func(d journal.Document) bool {
					if acked.Set == journal.SetTA1 {
						return journal.SameControl(d.Interchange, acked.Interchange)
					}
					return journal.SameControl(d.GroupControl, acked.GroupControl)
				})
			})
			if !found || slices.Contains(targets, target.ID) {
				continue
			}
			targets = append(targets, target.ID)
			for _, key := range target.Keys {
				if !slices.Contains(m.Keys, key) {
					m.Keys = append(m.Keys, key)
				}
			}
		}
	}
	return targets
}
