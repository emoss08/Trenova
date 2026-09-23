import type { AssistantMessage } from "@/types/assistant";
import { classifyMessage } from "./classify-message";

export type ToolCallRecord = NonNullable<AssistantMessage["toolCalls"]>[number];

/** One lookup the assistant asked for, paired with what came back. */
export type ToolExchange = {
  call: ToolCallRecord;
  result: AssistantMessage | null;
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
          });
        }
        break;
      }
      case "user":
        entries.push({
          kind: message.kind === "DecisionNote" ? "decision" : "user",
          message,
        });
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
