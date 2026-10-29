package agentaccess

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagedraft"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func (s *Access) AssertPageTurn(
	ctx context.Context,
	thread *conversation.Thread,
	page *agentpagecontext.PageContext,
	actor *permissionports.RequestActor,
) error {
	draft := pagedraft.Surface("")
	if page != nil && page.Draft != nil {
		draft = page.Draft.Surface
	}

	if !thread.Origin.PageBound() {
		if draft != "" {
			return errortypes.NewValidationError(
				"context.draft", errortypes.ErrInvalid,
				"Only an import or formula conversation carries a page's draft",
			)
		}

		return nil
	}

	surface := PageSurfaces[thread.Origin]
	subject, _ := surface.subject(thread.SubjectType)
	if thread.Status != conversation.ThreadStatusActive {
		return errortypes.NewBusinessError(
			"This conversation was closed when its {0} changed. Reopen the page to start a new one.",
			subject.SubjectType.Noun(),
		)
	}
	if draft != "" && !draftFits(&surface, subject, thread, draft) {
		return errortypes.NewValidationError(
			"context.draft", errortypes.ErrInvalid,
			"This conversation takes the page's {0} draft", string(subject.draft),
		)
	}
	if thread.HasSubject() {
		allowed, err := s.MayReadSubject(ctx, actor, thread.SubjectType)
		if err != nil {
			return err
		}
		if !allowed {
			return errortypes.NewAuthorizationError(
				"You no longer have access to the {0} this conversation is about.",
				thread.SubjectType.Noun(),
			)
		}
	}

	return nil
}

type PageSubject struct {
	SubjectType agent.SubjectType
	draft       pagedraft.Surface
	Title       string
}

type pageSurface struct {
	SystemKey       string
	subjectRequired bool
	subjects        []PageSubject
}

func (p *pageSurface) subject(subjectType agent.SubjectType) (PageSubject, bool) {
	if subjectType == "" {
		return p.subjects[0], true
	}
	for _, subject := range p.subjects {
		if subject.SubjectType == subjectType {
			return subject, true
		}
	}

	return PageSubject{}, false
}

func (p *pageSurface) takesDraft(draft pagedraft.Surface) bool {
	for _, subject := range p.subjects {
		if subject.draft == draft {
			return true
		}
	}

	return false
}

func (p *pageSurface) nouns() string {
	nouns := make([]string, 0, len(p.subjects))
	for _, subject := range p.subjects {
		nouns = append(nouns, subject.SubjectType.NounWithArticle())
	}

	return strings.Join(nouns, " or ")
}

var PageSurfaces = map[conversation.ThreadOrigin]pageSurface{
	conversation.ThreadOriginImport: {
		SystemKey:       agentdefinition.SystemKeyImportAssistant,
		subjectRequired: true,
		subjects: []PageSubject{{
			SubjectType: agent.SubjectDocument,
			draft:       pagedraft.SurfaceShipmentImport,
			Title:       "Shipment import",
		}},
	},
	conversation.ThreadOriginFormula: {
		SystemKey: agentdefinition.SystemKeyFormulaAssistant,
		subjects: []PageSubject{
			{
				SubjectType: agent.SubjectFormulaTemplate,
				draft:       pagedraft.SurfaceFormula,
				Title:       "Formula",
			},
			{
				SubjectType: agent.SubjectFormulaLibrary,
				draft:       pagedraft.SurfaceFormulaLibrary,
				Title:       "Formula library",
			},
		},
	},
}

func CheckPageSubject(
	surface *pageSurface,
	req *assistantports.OpenPageThreadRequest,
) (PageSubject, error) {
	subject, ok := surface.subject(req.SubjectType)
	if !ok {
		return PageSubject{}, errortypes.NewValidationError(
			"subjectType", errortypes.ErrInvalid,
			"This conversation is about {0}", surface.nouns(),
		)
	}
	if surface.subjectRequired && req.SubjectID.IsNil() {
		return PageSubject{}, errortypes.NewValidationError(
			"subjectId", errortypes.ErrRequired,
			"This conversation needs the {0} it is about", subject.SubjectType.Noun(),
		)
	}
	if req.SubjectID.IsNotNil() {
		if err := subject.SubjectType.CheckID(req.SubjectID); err != nil {
			return PageSubject{}, errortypes.NewValidationError(
				"subjectId", errortypes.ErrInvalid, err.Error(),
			)
		}
	}

	return subject, nil
}

func draftFits(
	surface *pageSurface,
	subject PageSubject,
	thread *conversation.Thread,
	draft pagedraft.Surface,
) bool {
	if thread.HasSubject() {
		return draft == subject.draft
	}

	return surface.takesDraft(draft)
}
