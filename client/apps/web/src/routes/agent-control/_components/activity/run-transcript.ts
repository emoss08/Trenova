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

export type TranscriptBlock =
  | {
      kind: "turn";
      key: string;
      entry: Extract<ThreadEntry, { kind: "assistant" }>;
      /** Part of the step was too large to keep whole: its words, reasoning or a result. */
      clipped: boolean;
    }
  | { kind: "note"; key: string; message: AssistantMessage }
  | { kind: "gap"; key: string; count: number };

function argumentsOf(value: unknown): Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : {};
}

/**
 * One saved message of a run in the shape the conversation draws, so the
 * run reads with the same pieces a thread does. A role this reader does not
 * know is left out rather than drawn as somebody else's words.
 */
function threadMessage(
  runId: string,
  index: number,
  message: AgentRunTranscriptMessage,
): AssistantMessage | null {
  const role = messageRoleSchema.safeParse(message.role);
  if (!role.success) {
    return null;
  }
  const verdict = toolVerdictSchema.safeParse(message.toolVerdict);

  return assistantMessageSchema.parse({
    id: `${runId}:${index}`,
    threadId: runId,
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

function blocksOf(
  runId: string,
  messages: readonly AgentRunTranscriptMessage[],
  offset: number,
): TranscriptBlock[] {
  const clipped = new Set<string>();
  const converted: AssistantMessage[] = [];
  messages.forEach((message, position) => {
    const saved = threadMessage(runId, offset + position, message);
    if (saved === null) {
      return;
    }
    if (message.omitted) {
      clipped.add(saved.id);
    }
    converted.push(saved);
  });

  return groupThread(converted).map((entry): TranscriptBlock => {
    if (entry.kind !== "assistant") {
      return { kind: "note", key: entry.message.id, message: entry.message };
    }
    const parts = [entry.message.id, ...entry.tools.map((tool) => tool.result?.id ?? "")];

    return {
      kind: "turn",
      key: entry.message.id,
      entry,
      clipped: parts.some((id) => clipped.has(id)),
    };
  });
}

/**
 * A run's transcript as the steps a reader follows: each of the model's turns
 * with the tools it called folded under it, and the stretch the record left
 * out, counted where it fell. The two sides of that gap are grouped apart, so
 * a call is never paired with a result from the other side of it.
 */
export function transcriptBlocks(runId: string, transcript: AgentRunTranscript): TranscriptBlock[] {
  const { messages, omittedMessages, omittedAt } = transcript;
  if (omittedMessages <= 0) {
    return blocksOf(runId, messages, 0);
  }

  const at = Math.min(Math.max(omittedAt, 0), messages.length);

  return [
    ...blocksOf(runId, messages.slice(0, at), 0),
    { kind: "gap", key: `${runId}:gap`, count: omittedMessages },
    ...blocksOf(runId, messages.slice(at), at + omittedMessages),
  ];
}
