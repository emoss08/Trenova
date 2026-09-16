import type { AssistantMessage } from "@/types/assistant";

/**
 * How a message should be presented.
 *
 * - `tool` — a lookup the assistant performed, shown collapsed
 * - `refusal` — a boundary the guard enforced, shown as a notice rather than as
 *   something the assistant said
 * - `declined-prompt` — the user turn that was refused, shown muted so the
 *   conversation still reads in order without implying it was answered
 * - `user` / `assistant` — ordinary turns
 */
export type MessagePresentation = "tool" | "refusal" | "declined-prompt" | "user" | "assistant";

/**
 * Decides how to render one message.
 *
 * Refusal takes precedence over role, and that ordering is the point: a refused
 * assistant turn carries the guard's explanation, not an answer, and rendering it
 * as a normal reply would tell someone the assistant answered when it declined.
 * A refused user turn is muted rather than hidden, so the thread still shows what
 * was asked.
 *
 * Tool messages are checked first because the server never marks them refused —
 * the guard runs on the turn, not on individual lookups — so role is the only
 * signal that matters for them.
 */
export function classifyMessage(message: AssistantMessage): MessagePresentation {
  if (message.role === "Tool") {
    return "tool";
  }

  if (message.refused) {
    return message.role === "User" ? "declined-prompt" : "refusal";
  }

  return message.role === "User" ? "user" : "assistant";
}
