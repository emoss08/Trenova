import type { AssistantMessage } from "@/types/assistant";

/**
 * How a message should be presented.
 *
 * - `delegated` — a step another agent took on a task this conversation's
 *   agent handed it, shown only inside that hand-off and never as a turn
 * - `tool` — a step the assistant took, shown collapsed under its turn
 * - `handoff` — the card left where a person took the conversation to another
 *   agent, or the brief that opens the conversation they took it to; neither
 *   is something either side said
 * - `decision` — the note the application wrote to start the turn after a
 *   decision, shown as the decision it records and never as its text
 * - `refusal` — a boundary the guard enforced, shown as a notice rather than as
 *   something the assistant said
 * - `declined-prompt` — the user turn that was refused, shown muted so the
 *   conversation still reads in order without implying it was answered
 * - `schedule` — a request the person scheduled, shown in their words with
 *   the schedule's card under it; nothing answered it there and then
 * - `compaction` — the summary a compaction wrote, shown as the line where the
 *   conversation was compacted and never as something the person said
 * - `steer` — what the person said while a reply was being written, shown as
 *   theirs and marked as read by the reply under way
 * - `world` — the notice a reply was given when a record it was working with
 *   changed elsewhere; neither side said it
 * - `wait` — the note that started the turn picking up work the agent parked
 *   on a wait, shown as the wait it records and never as its text
 * - `user` / `assistant` — ordinary turns
 */
export type MessagePresentation =
  | "compaction"
  | "steer"
  | "world"
  | "wait"
  | "delegated"
  | "handoff"
  | "tool"
  | "decision"
  | "refusal"
  | "declined-prompt"
  | "schedule"
  | "user"
  | "assistant";

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
 *
 * A decision note is read next, ahead of the refusal flag. Its text is the
 * application's instruction to the agent — the decision on its first line,
 * then how to report it — and a person never typed it. Letting a refused note
 * fall through to the muted "declined" turn printed those instructions in full
 * under the person's name.
 */
export function classifyMessage(message: AssistantMessage): MessagePresentation {
  // Another agent's steps come first of all: its task is saved in the User
  // role and its answer in the Assistant role, and drawn by role either
  // would read as the person asking or the conversation's agent replying.
  if (message.kind === "Delegated") {
    return "delegated";
  }

  // A compaction summary is saved in the User role because the model reads
  // it in the person's place; the person never wrote it.
  if (message.kind === "Compaction") {
    return "compaction";
  }

  if (message.kind === "WorldChange") {
    return "world";
  }

  if (message.kind === "WaitNote") {
    return "wait";
  }

  if (message.role === "Tool") {
    return "tool";
  }

  if (message.kind === "Steer") {
    return "steer";
  }

  if (message.kind === "Schedule") {
    return "schedule";
  }

  if (message.kind === "Handoff" || message.kind === "HandoffBrief") {
    return "handoff";
  }

  if (message.kind === "DecisionNote") {
    return "decision";
  }

  if (message.refused) {
    return message.role === "User" ? "declined-prompt" : "refusal";
  }

  return message.role === "User" ? "user" : "assistant";
}
