import { groupThread, type ThreadEntry } from "@/components/assistant/thread-view";
import type {
  AgentRunTranscript,
  AgentRunTranscriptMessage,
} from "@/lib/graphql/agent-activity-tables";
import {
  assistantMessageSchema,
  messageKindSchema,
  messageRoleSchema,
  toolVerdictSchema,
  type AssistantMessage,
} from "@/types/assistant";

export type ConversationReply = Extract<ThreadEntry, { kind: "assistant" }>;

/**
 * One row of a saved conversation read back the way the Desk draws it.
 *
 * - `asked`: what the person said. `tag` says it was sent while the agent was
 *   working (`steer`) or never reached the agent (`declined`).
 * - `turn`: one run of the agent's steps between two things the person said.
 *   `reply` is the last step that said something, read as the answer; `work`
 *   is every other step, in order: the notes it wrote on the way, the tools it
 *   called and what it thought. `askedAt` is when the question it answers was
 *   asked, which times its steps; `at` is when the turn answered.
 * - `event`: something that happened in the conversation that neither side
 *   said: a compaction, a wait that ended, a change elsewhere, a hand-over, a
 *   decision or a refusal.
 * - `gap`: messages the record left out of the middle, counted where they fell.
 */
export type ConversationItem =
  | {
      kind: "asked";
      key: string;
      message: AssistantMessage;
      tag: "steer" | "declined" | null;
    }
  | {
      kind: "turn";
      key: string;
      work: ConversationReply[];
      reply: ConversationReply | null;
      askedAt: number;
      at: number;
      /** Part of a step was too large to keep whole: its words, reasoning or a result. */
      clipped: boolean;
    }
  | { kind: "event"; key: string; message: AssistantMessage }
  | { kind: "gap"; key: string; count: number };

function argumentsOf(value: unknown): Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : {};
}

/**
 * One saved message in the shape the Desk draws. A role this reader does not
 * know is left out rather than drawn as somebody else's words.
 */
function savedMessage(
  conversationId: string,
  index: number,
  message: AgentRunTranscriptMessage,
): AssistantMessage | null {
  const role = messageRoleSchema.safeParse(message.role);
  if (!role.success) {
    return null;
  }
  const verdict = toolVerdictSchema.safeParse(message.toolVerdict);

  return assistantMessageSchema.parse({
    id: `${conversationId}:${index}`,
    threadId: conversationId,
    sequence: index,
    role: role.data,
    kind: messageKindSchema.safeParse(message.kind).data ?? "Message",
    agentId: message.agentDefinitionId || null,
    delegateCallId: message.delegateCallId || null,
    content: message.content,
    toolCalls: message.toolCalls.map((call) => ({
      id: call.id,
      name: call.name,
      arguments: argumentsOf(call.arguments),
    })),
    toolCallId: message.toolCallId,
    toolName: message.toolName,
    toolFailed: message.toolFailed,
    toolVerdict: verdict.success ? verdict.data : undefined,
    summary: message.toolSummary || undefined,
    reasoning: message.reasoning !== "" ? { text: message.reasoning } : null,
    createdAt: message.createdAt,
  });
}

function said(entry: ConversationReply): boolean {
  return entry.message.content.trim() !== "";
}

function turnOf(steps: ConversationReply[], askedAt: number, clipped: Set<string>) {
  let replyAt = -1;
  for (let index = steps.length - 1; index >= 0; index -= 1) {
    if (said(steps[index])) {
      replyAt = index;
      break;
    }
  }
  const reply = replyAt >= 0 ? steps[replyAt] : null;
  const last = steps[steps.length - 1];
  const parts = steps.flatMap((step) => [
    step.message.id,
    ...step.tools.map((tool) => tool.result?.id ?? ""),
  ]);

  return {
    kind: "turn" as const,
    key: steps[0].message.id,
    work: steps.filter((_, index) => index !== replyAt),
    reply,
    askedAt,
    at: (reply ?? last).message.createdAt,
    clipped: parts.some((id) => clipped.has(id)),
  };
}

function itemsOf(
  conversationId: string,
  messages: readonly AgentRunTranscriptMessage[],
  offset: number,
): ConversationItem[] {
  const clipped = new Set<string>();
  const converted: AssistantMessage[] = [];
  messages.forEach((message, position) => {
    const saved = savedMessage(conversationId, offset + position, message);
    if (saved === null) {
      return;
    }
    if (message.omitted) {
      clipped.add(saved.id);
    }
    converted.push(saved);
  });

  const items: ConversationItem[] = [];
  let steps: ConversationReply[] = [];
  let askedAt = 0;
  const closeTurn = () => {
    if (steps.length > 0) {
      items.push(turnOf(steps, askedAt, clipped));
    }
    steps = [];
  };

  for (const entry of groupThread(converted)) {
    if (entry.kind === "assistant") {
      steps.push(entry);
      continue;
    }
    closeTurn();
    switch (entry.kind) {
      case "user":
      case "schedule":
        askedAt = entry.message.createdAt;
        items.push({ kind: "asked", key: entry.message.id, message: entry.message, tag: null });
        break;
      case "steer":
        items.push({ kind: "asked", key: entry.message.id, message: entry.message, tag: "steer" });
        break;
      case "declined":
        items.push({
          kind: "asked",
          key: entry.message.id,
          message: entry.message,
          tag: "declined",
        });
        break;
      default:
        items.push({ kind: "event", key: entry.message.id, message: entry.message });
    }
  }
  closeTurn();

  return items;
}

/**
 * A saved conversation as the Desk reads it: each thing the person said, and
 * under it the agent's turn with its working set apart from its answer. The
 * stretch the record left out is counted where it fell, and the two sides of
 * it are grouped apart, so a call is never paired with a result from the other
 * side and a turn never spans the gap.
 */
export function conversationItems(
  conversationId: string,
  transcript: AgentRunTranscript,
): ConversationItem[] {
  const { messages, omittedMessages, omittedAt } = transcript;
  if (omittedMessages <= 0) {
    return itemsOf(conversationId, messages, 0);
  }

  const at = Math.min(Math.max(omittedAt, 0), messages.length);

  return [
    ...itemsOf(conversationId, messages.slice(0, at), 0),
    { kind: "gap", key: `${conversationId}:gap`, count: omittedMessages },
    ...itemsOf(conversationId, messages.slice(at), at + omittedMessages),
  ];
}
