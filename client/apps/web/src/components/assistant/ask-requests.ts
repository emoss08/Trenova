import { parseToolResult } from "./tool-presentation";
import type { ToolExchange } from "./thread-view";

/** One choice the assistant offered. */
export type AskOption = {
  value: string;
  label: string;
  detail: string;
};

/** A question the assistant put to the person, as the thread records it. */
export type ThreadAskRequest = {
  /** The tool message's position, used to tell an answered question from an open one. */
  sequence: number;
  callId: string;
  question: string;
  options: AskOption[];
  allowOther: boolean;
  otherHint: string;
};

const ASK_TOOL = "ask_user";

/**
 * Pulls the questions out of a turn's tool traffic.
 *
 * The runtime answers ask_user itself and stores the question as an ordinary
 * fenced tool result, so it arrives here the same way a report run does and is
 * read back the same way.
 */
export function askRequestsFrom(tools: readonly ToolExchange[]): ThreadAskRequest[] {
  const requests: ThreadAskRequest[] = [];

  for (const exchange of tools) {
    if (exchange.call.name !== ASK_TOOL) {
      continue;
    }
    const result = exchange.result;
    if (result === null || result.toolFailed) {
      continue;
    }

    const parsed = parseAskResult(result.content);
    if (parsed !== null) {
      requests.push({ ...parsed, sequence: result.sequence, callId: result.toolCallId });
    }
  }

  return requests;
}

/**
 * The same questions, read out of a turn that is still streaming.
 *
 * A live turn has no saved messages yet, so it cannot go through the thread
 * grouping — but the question is in the tool result either way, and the person
 * should not watch a turn end, see nothing, and wonder what they were asked.
 * The position is the one a saved message would not have yet, so a live prompt
 * is never marked answered.
 */
export function askRequestsFromSteps(
  steps: readonly { id: string; name: string; content: string }[],
): ThreadAskRequest[] {
  const requests: ThreadAskRequest[] = [];

  for (const step of steps) {
    if (step.name !== ASK_TOOL || step.content === "") {
      continue;
    }
    const parsed = parseAskResult(step.content);
    if (parsed !== null) {
      requests.push({ ...parsed, sequence: Number.MAX_SAFE_INTEGER, callId: step.id });
    }
  }

  return requests;
}

type ParsedAsk = Omit<ThreadAskRequest, "sequence" | "callId">;

/**
 * A question a model composed is not a typed contract. A malformed option is
 * dropped rather than failing the whole prompt, and a question with nothing
 * left to answer it is not rendered at all — an empty card is worse than the
 * sentence the assistant wrote alongside it.
 */
function parseAskResult(content: string): ParsedAsk | null {
  const result = parseToolResult(content);
  if (result.kind !== "json" || typeof result.value !== "object" || result.value === null) {
    return null;
  }

  const payload = result.value as Record<string, unknown>;
  const question = typeof payload.question === "string" ? payload.question.trim() : "";
  if (question === "") {
    return null;
  }

  const options = optionsFrom(payload.options);
  const allowOther = payload.allowOther !== false;
  if (options.length === 0 && !allowOther) {
    return null;
  }

  return {
    question,
    options,
    allowOther,
    otherHint: typeof payload.otherHint === "string" ? payload.otherHint : "",
  };
}

function optionsFrom(raw: unknown): AskOption[] {
  if (!Array.isArray(raw)) {
    return [];
  }

  const options: AskOption[] = [];
  for (const entry of raw) {
    if (typeof entry !== "object" || entry === null) {
      continue;
    }
    const { value, label, detail } = entry as Record<string, unknown>;
    const text = typeof label === "string" ? label : "";
    const answer = typeof value === "string" && value !== "" ? value : text;
    if (answer === "") {
      continue;
    }
    options.push({
      value: answer,
      label: text !== "" ? text : answer,
      detail: typeof detail === "string" ? detail : "",
    });
  }

  return options;
}
