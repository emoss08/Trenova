import {
  recordAgentSuggestion,
  settleAgentSuggestion,
  type AgentSuggestionSurface,
} from "@/lib/graphql/agent-suggestion";
import { useState } from "react";

type SettledOutcome = Parameters<typeof settleAgentSuggestion>[1];

type OpenSuggestion = {
  agentId: string;
  /** The record's id once the server has it; null when recording failed. */
  id: Promise<string | null>;
};

export type SuggestionRecord = {
  /** A suggestion appeared: recorded once while the same agent stays named. */
  shown: (suggestedAgentId: string, currentAgentId: string | null) => void;
  /** What the person did with the suggestion on screen, if there is one. */
  settle: (outcome: SettledOutcome) => void;
  /**
   * Takes the suggestion on screen for a question being sent, to settle once
   * the send knows where the question went; clearing the box meanwhile does
   * not lose it.
   */
  take: () => (outcome: SettledOutcome) => void;
  /** The question went to another agent without a suggestion being shown. */
  autoSwitched: (suggestedAgentId: string, currentAgentId: string | null) => void;
  /** The box was cleared: the suggestion on screen is left as it was. */
  forget: () => void;
};

/**
 * Records, for AI analytics, each agent the Desk names for a question and what
 * the person does with it. Recording never holds anything up and a failure is
 * not the person's concern: a suggestion that could not be recorded is simply
 * not counted. One left unsettled is counted as passed over by the server.
 */
export function createSuggestionRecord(surface: AgentSuggestionSurface): SuggestionRecord {
  let open: OpenSuggestion | null = null;

  const take = () => {
    const current = open;
    open = null;

    return (outcome: SettledOutcome) => {
      if (!current) {
        return;
      }
      void current.id
        .then((id) => (id ? settleAgentSuggestion(id, outcome) : undefined))
        .catch(() => undefined);
    };
  };
  const settle = (outcome: SettledOutcome) => take()(outcome);

  return {
    shown: (suggestedAgentId, currentAgentId) => {
      if (open?.agentId === suggestedAgentId) {
        return;
      }
      settle("PassedOver");
      open = {
        agentId: suggestedAgentId,
        id: recordAgentSuggestion({
          suggestedAgentId,
          currentAgentId,
          surface,
          outcome: "Shown",
        }).catch(() => null),
      };
    },
    settle,
    take,
    autoSwitched: (suggestedAgentId, currentAgentId) => {
      void recordAgentSuggestion({
        suggestedAgentId,
        currentAgentId,
        surface,
        outcome: "AutoSwitched",
      }).catch(() => undefined);
    },
    forget: () => {
      open = null;
    },
  };
}

/** One record of suggestions for the life of the composer it belongs to. */
export function useSuggestionRecord(surface: AgentSuggestionSurface): SuggestionRecord {
  const [record] = useState(() => createSuggestionRecord(surface));

  return record;
}
