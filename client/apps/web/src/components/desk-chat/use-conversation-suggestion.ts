import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { useDeskSetting } from "@/stores/desk-settings-store";
import type { AssistantEntityRef, AssistantPageContext } from "@/types/assistant";
import { useEffect, useState } from "react";
import {
  matchContextKinds,
  noAgentFits,
  offersAgent,
  showsSuggestion,
  suggestionMode,
  type OfferedMatch,
} from "./agent-recommendation-state";
import { useAgentMatch } from "./use-agent-match";
import { useSuggestionRecord } from "./use-suggestion-record";

/** Hands the conversation to another agent and asks it the question there. */
export type ConversationHandOff = (
  agentId: string,
  question: string,
  mentions: readonly AssistantEntityRef[],
) => Promise<void>;

export type ConversationSuggestion = {
  /** Another agent holds the tools the reply being written needs. */
  offered: OfferedMatch | null;
  /** Tools match the reply being written but none of the person's agents holds them. */
  nobody: boolean;
  /** A reply is waiting a moment for its match, to be handed off if another agent fits. */
  routing: boolean;
  /** The conversation is being handed to another agent. */
  handingOff: boolean;
  handOff: () => Promise<void>;
  keep: () => void;
  dismissNobody: () => void;
  /**
   * Sends a reply to the agent that holds its tools when the person lets the
   * Desk move questions without asking: true when it was handed off, false
   * when it is the conversation's to send.
   */
  route: (content: string, mentions: readonly AssistantEntityRef[]) => Promise<boolean>;
  /** A reply was sent here: the suggestion on screen, if any, was passed over. */
  sentHere: () => void;
};

/**
 * Matches the reply being written in an open conversation to the person's
 * agents, as the front page matches a new question. Another agent that holds
 * the tools is offered above the box with Hand off, which hands the
 * conversation over with its summary and pinned facts and asks the reply
 * there; with auto switch on the reply is handed off without asking. Nothing
 * is matched while the agent is working, while files are attached (they
 * belong to this conversation), or where the conversation cannot be handed off.
 */
export function useConversationSuggestion({
  agent,
  draft,
  mentions,
  page,
  attachmentCount,
  busy,
  onHandOff,
  clearDraft,
}: {
  agent: AgentChoice | null;
  draft: string;
  mentions: readonly AssistantEntityRef[];
  page: AssistantPageContext | null;
  attachmentCount: number;
  busy: boolean;
  onHandOff?: ConversationHandOff;
  clearDraft: () => void;
}): ConversationSuggestion {
  const mode = suggestionMode({
    autoSwitchAgent: useDeskSetting("autoSwitchAgent"),
    suggestAgent: useDeskSetting("suggestAgent"),
  });
  const record = useSuggestionRecord("Conversation");
  const [keptOver, setKeptOver] = useState<readonly string[]>([]);
  const [nobodyDismissed, setNobodyDismissed] = useState(false);
  const [sending, setSending] = useState<string | null>(null);
  const [routing, setRouting] = useState(false);
  const [handingOff, setHandingOff] = useState(false);
  const agentId = agent?.id ?? null;
  const enabled =
    onHandOff !== undefined && agentId !== null && !busy && attachmentCount === 0 && mode !== "off";
  const advice = useAgentMatch({
    draft: sending ?? draft,
    agentId,
    contextKinds: matchContextKinds(mentions, page),
    enabled,
  });
  const offered =
    enabled && showsSuggestion(mode) && offersAgent(advice.match, agentId, keptOver)
      ? advice.match
      : null;
  const nobody = enabled && noAgentFits(advice.match) && !nobodyDismissed;
  const offeredId = offered?.agent.id ?? null;

  useEffect(() => {
    if (offeredId !== null) {
      record.shown(offeredId, agentId);
    }
  }, [offeredId, agentId, record]);

  // Clearing the box starts the next reply afresh: the agents turned down and
  // the note put away belong to the reply that was being written.
  const cleared = draft.trim() === "" && sending === null;
  const [wasCleared, setWasCleared] = useState(cleared);
  if (cleared !== wasCleared) {
    setWasCleared(cleared);
    if (cleared) {
      setKeptOver([]);
      setNobodyDismissed(false);
    }
  }
  useEffect(() => {
    if (cleared) {
      record.forget();
    }
  }, [cleared, record]);

  const handOffTo = async (
    targetId: string,
    question: string,
    carried: readonly AssistantEntityRef[],
  ) => {
    if (!onHandOff) {
      return false;
    }
    setHandingOff(true);
    try {
      await onHandOff(targetId, question, carried);
      clearDraft();

      return true;
    } catch {
      return false;
    } finally {
      setHandingOff(false);
    }
  };

  return {
    offered,
    nobody,
    routing,
    handingOff,
    handOff: async () => {
      if (!offered || routing || handingOff) {
        return;
      }
      record.settle("Switched");
      await handOffTo(offered.agent.id, draft.trim(), mentions);
    },
    keep: () => {
      if (offered && !routing && !handingOff) {
        setKeptOver((previous) => [...previous, offered.agent.id]);
        record.settle("Kept");
      }
    },
    dismissNobody: () => setNobodyDismissed(true),
    route: async (content, carried) => {
      if (mode !== "auto" || !enabled || agentId === null) {
        return false;
      }
      const settleShown = record.take();
      setSending(content);
      setRouting(true);
      const match = await advice.resolve(content);
      setRouting(false);
      setSending(null);
      if (!offersAgent(match, agentId, keptOver)) {
        settleShown("PassedOver");
        return false;
      }
      settleShown("PassedOver");
      record.autoSwitched(match.agent.id, agentId);

      return handOffTo(match.agent.id, content, carried);
    },
    sentHere: () => record.settle("PassedOver"),
  };
}
