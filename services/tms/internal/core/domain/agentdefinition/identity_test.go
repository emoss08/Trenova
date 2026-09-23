package agentdefinition_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

func TestKnownIconsAndAccents(t *testing.T) {
	t.Parallel()

	icons := agentdefinition.KnownIcons()
	require.NotEmpty(t, icons)
	require.Contains(t, icons, agentdefinition.IconBot)

	seen := make(map[string]struct{}, len(icons))
	for _, icon := range icons {
		require.NotContains(t, seen, icon, "icon names must be unique")
		seen[icon] = struct{}{}
	}

	accents := agentdefinition.KnownAccents()
	require.Len(t, accents, 8, "eight accents keep the picker one row")
	require.Contains(t, accents, agentdefinition.AccentIndigo)
}

func TestResolvedIcon_FallsBackToTheTemplate(t *testing.T) {
	t.Parallel()

	chosen := &agentdefinition.Definition{Icon: agentdefinition.IconShield}
	require.Equal(t, agentdefinition.IconShield, chosen.ResolvedIcon())

	fromTemplate := &agentdefinition.Definition{Template: agentdefinition.TemplateBillingException}
	require.Equal(t, agentdefinition.IconReceipt, fromTemplate.ResolvedIcon())

	bare := &agentdefinition.Definition{}
	require.Equal(t, agentdefinition.IconBot, bare.ResolvedIcon())
}

func TestChosenIcon_IsEmptyForAnAgentWithNoIconOfItsOwn(t *testing.T) {
	t.Parallel()

	chosen := &agentdefinition.Definition{Icon: agentdefinition.IconShield}
	require.Equal(t, agentdefinition.IconShield, chosen.ChosenIcon())

	fromTemplate := &agentdefinition.Definition{Template: agentdefinition.TemplateBillingException}
	require.Equal(t, agentdefinition.IconReceipt, fromTemplate.ChosenIcon())

	unknown := &agentdefinition.Definition{Icon: "not-an-icon"}
	require.Empty(t, unknown.ChosenIcon())

	bare := &agentdefinition.Definition{}
	require.Empty(t, bare.ChosenIcon(), "a reader draws its initials, not the generic icon")
}

func TestResolvedAccent_IsStableAndSpread(t *testing.T) {
	t.Parallel()

	chosen := &agentdefinition.Definition{Accent: agentdefinition.AccentAmber}
	require.Equal(t, agentdefinition.AccentAmber, chosen.ResolvedAccent())

	id := pulid.MustNew("agdef_")
	first := (&agentdefinition.Definition{ID: id}).ResolvedAccent()
	second := (&agentdefinition.Definition{ID: id}).ResolvedAccent()
	require.Equal(t, first, second, "the same agent must always look the same")
	require.Contains(t, agentdefinition.KnownAccents(), first)

	byName := (&agentdefinition.Definition{Name: "Dispatch desk"}).ResolvedAccent()
	require.Contains(t, agentdefinition.KnownAccents(), byName,
		"an agent that has not been saved yet still needs an accent")

	counts := make(map[string]int, 8)
	for range 400 {
		accent := (&agentdefinition.Definition{ID: pulid.MustNew("agdef_")}).ResolvedAccent()
		counts[accent]++
	}
	require.Len(t, counts, 8, "every accent should be reachable")
}

func TestValidate_RejectsUnknownIdentity(t *testing.T) {
	t.Parallel()

	definition := identityDefinition()
	definition.Icon = "skull"
	definition.Accent = "chartreuse"

	multiErr := errortypes.NewMultiError()
	definition.Validate(multiErr)

	require.True(t, multiErr.HasErrors())
	require.Contains(t, multiErr.Error(), "Icon")
	require.Contains(t, multiErr.Error(), "Accent")
}

func TestValidate_AcceptsAKnownIdentityAndAnEmptyOne(t *testing.T) {
	t.Parallel()

	chosen := identityDefinition()
	chosen.Icon = agentdefinition.IconTruck
	chosen.Accent = agentdefinition.AccentTeal

	multiErr := errortypes.NewMultiError()
	chosen.Validate(multiErr)
	require.False(t, multiErr.HasErrors(), multiErr.Error())

	unset := identityDefinition()
	multiErr = errortypes.NewMultiError()
	unset.Validate(multiErr)
	require.False(t, multiErr.HasErrors(), multiErr.Error())
}

func identityDefinition() *agentdefinition.Definition {
	definition := &agentdefinition.Definition{
		OrganizationID:  pulid.MustNew("org_"),
		BusinessUnitID:  pulid.MustNew("bu_"),
		Name:            "Dispatch desk",
		AutonomyCeiling: agent.TierPropose,
		TriggerMode:     agentdefinition.TriggerChat,
	}
	definition.ApplyDefaults()

	return definition
}
