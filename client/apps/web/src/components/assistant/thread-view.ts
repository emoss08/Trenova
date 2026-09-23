import type { AssistantMessage } from "@/types/assistant";
import { classifyMessage } from "./classify-message";

export type ToolCallRecord = NonNullable<AssistantMessage["toolCalls"]>[number];

/** One lookup the assistant asked for, paired with what came back. */
export type ToolExchange = {
  call: ToolCallRecord;
  result: AssistantMessage | null;
  /** The call is not in view; it was rebuilt from its result. */
  orphan?: boolean;
};

export type ThreadEntry =
  | { kind: "user"; message: AssistantMessage }
  | { kind: "decision"; message: AssistantMessage }
  | { kind: "declined"; message: AssistantMessage }
  | { kind: "refusal"; message: AssistantMessage }
  | { kind: "assistant"; message: AssistantMessage; tools: ToolExchange[] };

/**
 * Folds the flat, saved thread into what the reader sees: each lookup under the
 * turn that asked for it, paired by call id rather than by position.
 *
 * A model can ask for several tools in one turn and their results are saved in
 * the order they finished, so position alone would pair them wrongly. A result
 * whose call is not in the thread (an older turn, a trimmed history) is still
 * shown, attached to the assistant message that follows it, because a lookup
 * the reader cannot see is worse than one shown slightly out of place.
 */
export function groupThread(messages: readonly AssistantMessage[]): ThreadEntry[] {
  const entries: ThreadEntry[] = [];
  const openCalls = new Map<string, ToolExchange>();
  let orphans: ToolExchange[] = [];

  for (const message of messages) {
    switch (classifyMessage(message)) {
      case "tool": {
        const open = openCalls.get(message.toolCallId);
        if (open) {
          open.result = message;
          openCalls.delete(message.toolCallId);
        } else {
          orphans.push({
            call: { id: message.toolCallId, name: message.toolName, arguments: {} },
            result: message,
            orphan: true,
          });
        }
        break;
      }
      case "decision":
        entries.push({ kind: "decision", message });
        break;
      case "user":
        entries.push({ kind: "user", message });
        break;
      case "declined-prompt":
        entries.push({ kind: "declined", message });
        break;
      case "refusal":
        entries.push({ kind: "refusal", message });
        break;
      case "assistant": {
        const tools: ToolExchange[] = [...orphans];
        orphans = [];
        for (const call of message.toolCalls ?? []) {
          const exchange: ToolExchange = { call, result: null };
          tools.push(exchange);
          openCalls.set(call.id, exchange);
        }
        entries.push({ kind: "assistant", message, tools });
        break;
      }
    }
  }

  // Results that arrived after the last assistant message, with nothing to
  // follow them, are given a home rather than lost.
  if (orphans.length > 0) {
    const last = entries.at(-1);
    if (last && last.kind === "assistant") {
      last.tools.push(...orphans);
    }
  }

  return entries;
}

/** Where an assistant entry sits in the reply it belongs to. */
export type TurnPlacement = {
  /** A later step of the same reply: drawn without the agent's header. */
  continued: boolean;
  /** On the first step: how long the whole reply took, from the question to its last step. */
  workedSeconds: number | null;
};

/**
 * A reply that took several model steps is saved as several assistant
 * messages, one per step. Drawn as they are stored, the agent's name and
 * face repeated over every step, so one answer read as four people talking.
 * The first step carries the header and how long the reply took; the rest
 * continue under it.
 *
 * The time runs from the message that asked to the last step of the reply,
 * both stamped when they were made. A reply whose question is not in view
 * has nothing to measure from and says nothing rather than guessing.
 */
export function turnPlacements(entries: readonly ThreadEntry[]): Map<string, TurnPlacement> {
  const placements = new Map<string, TurnPlacement>();
  let askedAt = 0;
  let lead: TurnPlacement | null = null;
  let lastAt = 0;

  const closeRun = () => {
    if (lead !== null && askedAt > 0 && lastAt >= askedAt) {
      lead.workedSeconds = lastAt - askedAt;
    }
    lead = null;
  };

  for (const entry of entries) {
    if (entry.kind !== "assistant") {
      closeRun();
      askedAt = entry.kind === "refusal" ? 0 : entry.message.createdAt;
      continue;
    }
    if (lead === null) {
      lead = { continued: false, workedSeconds: null };
      placements.set(entry.message.id, lead);
    } else {
      placements.set(entry.message.id, { continued: true, workedSeconds: null });
    }
    lastAt = entry.message.createdAt;
  }
  closeRun();

  return placements;
}

const TOOL_IDENTIFIER = /\b[a-z][a-z0-9]*(?:_[a-z0-9]+)+\b/gu;

/**
 * What a decision note shows: its first line, which is the decision, and
 * nothing after it, which is the application instructing the agent. A tool
 * named on that line is written as words — "Approved create report" — since
 * a person never chose the identifier.
 */
export function decisionHeadline(content: string): string {
  const first = content.split("\n", 1)[0]?.trim() ?? "";

  return first.replace(TOOL_IDENTIFIER, (name) => name.replaceAll("_", " "));
}
