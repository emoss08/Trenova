import type { WatchtowerSourceKind } from "@/lib/graphql/watchtower";
import type { AssistantMessage } from "@/types/assistant";

/**
 * The hover fade a turn's actions use, for a thumb nobody has chosen yet.
 * A chosen thumb ignores it and stays in view.
 */
export const TURN_FEEDBACK_REVEAL = "opacity-0 group-hover/turn:opacity-100";

type ThreadStep = { kind: string; message: { id: string; content: string } };

/**
 * Which assistant steps are the answer a person rates.
 *
 * A reply that took several model steps is saved as several messages; only
 * the last of them that says anything is the answer, so each run of
 * consecutive assistant steps offers one rating, on that step.
 */
export function answerMessageIds(entries: readonly ThreadStep[]): Set<string> {
  const answers = new Set<string>();
  let candidate: string | null = null;

  for (const entry of entries) {
    if (entry.kind !== "assistant") {
      if (candidate !== null) {
        answers.add(candidate);
      }
      candidate = null;
      continue;
    }
    if (entry.message.content.trim() !== "") {
      candidate = entry.message.id;
    }
  }
  if (candidate !== null) {
    answers.add(candidate);
  }

  return answers;
}

/** The message that carries another agent's answer to a handed-off task. */
export function delegatedAnswerId(messages: readonly AssistantMessage[]): string | null {
  for (let index = messages.length - 1; index >= 0; index -= 1) {
    const message = messages[index];
    if (message.role === "Assistant" && message.content.trim() !== "") {
      return message.id;
    }
  }

  return null;
}

const AI_SOURCE_KINDS: ReadonlySet<WatchtowerSourceKind> = new Set<WatchtowerSourceKind>([
  "Insight",
  "AgentProposal",
  "AgentPlan",
  "AgentRunFailed",
  "AgentException",
]);

/** Whether a watchtower item was raised by AI work, and so can be rated. */
export function isRatableWatchtowerKind(kind: WatchtowerSourceKind): boolean {
  return AI_SOURCE_KINDS.has(kind);
}
