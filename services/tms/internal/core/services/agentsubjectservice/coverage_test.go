package agentsubjectservice

import (
	"os"
	"regexp"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// describedElsewhere are the subject types that deliberately have no arm,
// each because the run already holds what an arm would fetch. Adding to this
// map is the way to opt a subject out, and the reason is the point of it.
var describedElsewhere = map[agent.SubjectType]string{
	agent.SubjectOrganization: "the run's own tenant; its context already carries this",
	agent.SubjectAssistantThread: "the thread is the conversation asking, so describing " +
		"it back would only repeat what the turn already holds",
}

// Describe's default arm hands back the subject type as its own label, which
// is what a run sees when nobody wrote a describer: an id and a word. That is
// the right answer for a type this service has never heard of and the wrong
// one for a type the codebase declares, but the switch is not something the
// compiler checks. The arms are read out of the source here so a subject type
// added without a describer is caught before a run starts blind.
func TestDescribe_EveryDeclaredSubjectHasAnArm(t *testing.T) {
	t.Parallel()

	source, err := os.ReadFile("service.go")
	require.NoError(t, err)

	// Only a case counts. A subject named anywhere else in the file — stamped
	// onto a RuntimeSubject, say — is not a branch Describe can take.
	pattern := regexp.MustCompile(`case agent\.(Subject\w+)[,:]`)
	handled := make(map[string]struct{})
	for _, match := range pattern.FindAllStringSubmatch(string(source), -1) {
		handled[match[1]] = struct{}{}
	}

	names := regexp.MustCompile(`(Subject\w+)\s+=\s+SubjectType\("([^"]+)"\)`)
	enums, err := os.ReadFile("../../domain/agent/enums.go")
	require.NoError(t, err)

	declared := names.FindAllStringSubmatch(string(enums), -1)
	require.NotEmpty(t, declared)

	for _, match := range declared {
		constant, value := match[1], match[2]
		if _, exempt := describedElsewhere[agent.SubjectType(value)]; exempt {
			continue
		}
		_, ok := handled[constant]
		require.Truef(t, ok,
			"subject type %q has no arm in Describe; give it a describer, or list it "+
				"in describedElsewhere with the reason it needs none",
			value,
		)
	}
}

func TestDescribe_AnUnknownSubjectStillReadsAsSomething(t *testing.T) {
	t.Parallel()

	svc := &Service{logger: zap.NewNop()}
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	subjectID := pulid.MustNew("sub_")

	subject, err := svc.Describe(t.Context(), tenant, agent.SubjectType("Teleporter"), subjectID)
	require.NoError(t, err)
	require.NotNil(t, subject)
	require.Equal(t, "Teleporter", subject.Label)
	require.Equal(t, subjectID.String(), subject.ID)
}

// Every repository behind the new arms is optional, so an arm whose dependency
// was not wired has to hand back a usable subject rather than fall through to
// the default or panic.
func TestDescribe_UnwiredArmsStillNameTheirSubject(t *testing.T) {
	t.Parallel()

	svc := &Service{logger: zap.NewNop()}
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	cases := map[agent.SubjectType]string{
		agent.SubjectDetentionOccurrence: "Detention occurrence",
		agent.SubjectWorker:              "Driver",
		agent.SubjectCarrierIntelEvent:   "Carrier finding",
		agent.SubjectEDIInboundFile:      "EDI inbound file",
		agent.SubjectInboundMessage:      "Inbound message",
	}

	for subjectType, label := range cases {
		t.Run(string(subjectType), func(t *testing.T) {
			t.Parallel()

			subjectID := pulid.MustNew("sub_")
			subject, err := svc.Describe(t.Context(), tenant, subjectType, subjectID)
			require.NoError(t, err)
			require.NotNil(t, subject)
			require.Equal(t, subjectType, subject.Type)
			require.Equal(t, subjectID.String(), subject.ID)
			require.Equal(t, label, subject.Label)
		})
	}
}
