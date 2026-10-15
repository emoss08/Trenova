import {
  RecordMyAgentSuggestionDocument,
  SettleMyAgentSuggestionDocument,
  type AgentSuggestionOutcome,
  type AgentSuggestionSurface,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type { AgentSuggestionOutcome, AgentSuggestionSurface };

/**
 * Records an agent the Desk named, or moved a question to without asking,
 * and returns the record's id for settling what became of it.
 */
export async function recordAgentSuggestion(input: {
  suggestedAgentId: string;
  currentAgentId: string | null;
  surface: AgentSuggestionSurface;
  outcome: Extract<AgentSuggestionOutcome, "Shown" | "AutoSwitched">;
}): Promise<string> {
  const data = await requestGraphQL({
    document: RecordMyAgentSuggestionDocument,
    operationName: "RecordMyAgentSuggestion",
    variables: {
      input: {
        suggestedAgentId: input.suggestedAgentId,
        ...(input.currentAgentId ? { currentAgentId: input.currentAgentId } : {}),
        surface: input.surface,
        outcome: input.outcome,
      },
    },
  });

  return data.recordMyAgentSuggestion.id;
}

/** Records what the person did with a suggestion; one already settled keeps its outcome. */
export async function settleAgentSuggestion(
  id: string,
  outcome: Extract<AgentSuggestionOutcome, "Switched" | "SentThrough" | "Kept" | "PassedOver">,
): Promise<void> {
  await requestGraphQL({
    document: SettleMyAgentSuggestionDocument,
    operationName: "SettleMyAgentSuggestion",
    variables: { input: { id, outcome } },
  });
}
