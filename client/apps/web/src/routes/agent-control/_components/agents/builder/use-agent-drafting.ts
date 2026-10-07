import {
  draftAgentFromDescription,
  tightenAgentInstructions,
  type AgentDraft,
  type TightenedInstructions,
} from "@/lib/graphql/agent-builder";
import { queries } from "@/lib/queries";
import type { AutonomyTier, TriggerMode } from "@/types/assistant";
import { useQuery } from "@tanstack/react-query";
import { useCallback } from "react";
import type { AgentFormValues } from "../agent-form-schema";

export type DraftedAgent = Partial<AgentFormValues> & { notes: AgentDraft["notes"] };

/** A drafted agent in the builder's own fields. */
export function draftedValues(draft: AgentDraft): DraftedAgent {
  const tiers =
    draft.toolTiers && typeof draft.toolTiers === "object"
      ? (draft.toolTiers as Record<string, AutonomyTier>)
      : {};
  return {
    name: draft.name,
    description: draft.description,
    icon: draft.icon,
    accent: draft.accent,
    instructions: draft.instructions,
    guardrails: [...draft.guardrails],
    triggerMode: draft.triggerMode as TriggerMode,
    cronExpression: draft.cronExpression,
    cronTimezone: draft.cronTimezone,
    eventKinds: [...draft.eventKinds],
    intervalSeconds: draft.intervalSeconds,
    toolNames: [...draft.toolNames],
    toolTiers: { ...tiers },
    autonomyCeiling: draft.autonomyCeiling,
    dataAccessCeiling: draft.dataAccessCeiling,
    outputMode: draft.outputMode,
    enabled: draft.enabled,
    shadowMode: draft.shadowMode,
    simulationMode: false,
    decisionTimeoutSeconds: draft.decisionTimeoutSeconds,
    runTimeoutSeconds: draft.runTimeoutSeconds,
    maxToolCalls: draft.maxToolCalls,
    maxConcurrentRuns: draft.maxConcurrentRuns,
    notes: draft.notes,
  };
}

/** Whether Nova can draft and tighten, and the two calls. */
export function useAgentDrafting(): {
  available: boolean;
  draft: (description: string) => Promise<DraftedAgent>;
  tighten: (instructions: string) => Promise<TightenedInstructions>;
} {
  const availableQuery = useQuery({ ...queries.agentBuilder.drafting(), staleTime: 60_000 });
  const draft = useCallback(
    async (description: string) => draftedValues(await draftAgentFromDescription(description)),
    [],
  );
  const tighten = useCallback(
    (instructions: string) => tightenAgentInstructions(instructions),
    [],
  );
  return { available: availableQuery.data ?? false, draft, tighten };
}
