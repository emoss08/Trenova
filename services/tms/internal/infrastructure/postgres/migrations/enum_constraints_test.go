package migrations

import (
	"io/fs"
	"regexp"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/agentextension"
	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/internal/core/domain/aifeedback"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/stretchr/testify/require"
)

type enumConstraint struct {
	name   string
	values []string
}

func stringsOf[T ~string](values []T) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, string(value))
	}

	return out
}

/*
Every value a Go enum declares is one the column's CHECK constraint accepts.

A template added to the domain and forgotten here is a row the database
refuses. The seed for the intake desk hit exactly that: the check still listed
the original eight templates, so a fresh install could not seed at all, and an
administrator could not create any of the desk agents.

The constraint is read from the newest up migration that adds it, which is the
one a migrated database is left with.
*/
func TestEnumCheckConstraintsAcceptEveryDeclaredValue(t *testing.T) {
	t.Parallel()

	constraints := []enumConstraint{
		{
			name:   "ck_accounting_connections_status",
			values: stringsOf(accountingsync.AllConnectionStatuses()),
		},
		{
			name:   "ck_accounting_connections_last_error_category",
			values: stringsOf(accountingsync.AllErrorCategories()),
		},
		{
			name:   "ck_agent_definitions_template",
			values: stringsOf(agentdefinition.AllTemplates()),
		},
		{
			name:   "ck_agent_extensions_type",
			values: stringsOf(agentextension.AllTypes()),
		},
		{
			name:   "ck_agent_extensions_availability",
			values: stringsOf(agentextension.AllAvailabilities()),
		},
		{
			name:   "ck_agent_definitions_access_mode",
			values: stringsOf(agentdefinition.AllAccessModes()),
		},
		{
			name:   "ck_assistant_artifacts_kind",
			values: stringsOf(assistantartifact.AllKinds()),
		},
		{
			name:   "ck_assistant_messages_kind",
			values: stringsOf(conversation.AllMessageKinds()),
		},
		{
			name:   "ck_agent_eval_cases_source",
			values: stringsOf(agentquality.AllCaseSources()),
		},
		{
			name:   "ck_agent_eval_cases_status",
			values: stringsOf(agentquality.AllCaseStatuses()),
		},
		{
			name:   "chk_agent_evaluations_status",
			values: stringsOf(agent.AllEvaluationStatuses()),
		},
		{
			name:   "chk_agent_memories_kind",
			values: stringsOf(agent.AllMemoryKinds()),
		},
		{
			name:   "chk_agent_memories_source",
			values: stringsOf(agent.AllMemorySources()),
		},
		{
			name:   "chk_agent_memories_status",
			values: stringsOf(agent.AllMemoryStatuses()),
		},
		{
			name:   "chk_agent_memories_scope",
			values: stringsOf(agent.AllMemoryScopes()),
		},
		{
			name:   "chk_agent_proposals_egress_class",
			values: stringsOf(agent.EgressClasses()),
		},
		{
			name:   "ck_ai_feedback_target_type",
			values: stringsOf(aifeedback.AllTargetTypes()),
		},
		{
			name:   "ck_ai_feedback_fingerprint_source",
			values: stringsOf(aifeedback.AllFingerprintSources()),
		},
		{
			name:   "ck_agent_suite_runs_status",
			values: stringsOf(agentquality.AllSuiteRunStatuses()),
		},
		{
			name:   "ck_agent_suite_runs_trigger",
			values: stringsOf(agentquality.AllSuiteRunTriggers()),
		},
		{
			name:   "ck_ai_providers_embedding_input_style",
			values: stringsOf(aiprovider.AllEmbeddingInputStyles()),
		},
		{
			name:   "ck_ai_index_entries_source_type",
			values: stringsOf(airetrieval.AllSourceTypes()),
		},
		{
			name:   "ck_ai_index_entries_status",
			values: stringsOf(airetrieval.AllIndexStatuses()),
		},
		{
			name:   "ck_ai_retrieval_settings_paused_reason",
			values: stringsOf(airetrieval.AllPauseReasons()),
		},
		{
			name:   "ck_ai_embeddings_source_type",
			values: stringsOf(airetrieval.AllSourceTypes()),
		},
		{
			name:   "ck_ai_catalog_embeddings_corpus",
			values: stringsOf(airetrieval.AllCatalogCorpora()),
		},
	}

	files := embeddedMigrationFiles(t)
	slices.SortFunc(files, func(a, b migrationFile) int {
		if a.version < b.version {
			return -1
		}
		if a.version > b.version {
			return 1
		}
		return 0
	})

	for _, constraint := range constraints {
		adds := regexp.MustCompile(
			`(?s)ADD CONSTRAINT "` + constraint.name + `"(.*?);|CONSTRAINT "` +
				constraint.name + `" CHECK(.*?)\)\s*,?\s*\n`,
		)

		var latest string
		for _, file := range files {
			if file.direction != "up" {
				continue
			}
			body, err := fs.ReadFile(sqlMigrations, file.name)
			require.NoError(t, err)
			if match := adds.FindSubmatch(body); match != nil {
				latest = string(match[0])
			}
		}
		require.NotEmpty(t, latest, "no migration adds %s", constraint.name)

		for _, value := range constraint.values {
			require.Contains(t, latest, "'"+value+"'",
				"%s does not accept %q; add a migration that widens it", constraint.name, value)
		}
	}
}
