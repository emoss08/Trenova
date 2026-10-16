import { isMentionKind, type AssistantMessage } from "@/types/assistant";
import type { MentionKind } from "@trenova/shared/types/generated/record-kinds";
import { parseToolResult } from "./tool-presentation";
import type { ThreadEntry, ToolExchange } from "./thread-view";
import type { TurnSegment } from "./turn-stream";

/** One choice the assistant offered. */
export type AskOption = {
  value: string;
  label: string;
  detail: string;
};

/** The kinds of value a person can type in answer: words, a number, a day, a moment or a record. */
export type AskValueType = "text" | "number" | "date" | "datetime" | "record";

/**
 * How a typed value is entered. A record names the kind it is searched among;
 * a kind the @ search cannot find is read as words instead, because a picker
 * that can find nothing is worse than a box.
 */
export type AskInput =
  | { type: Exclude<AskValueType, "record"> }
  | { type: "record"; recordKind: MentionKind };

/** One value of a small form the assistant asked for, which together complete one change. */
export type AskField = AskInput & {
  key: string;
  label: string;
  /** What the value should look like, said under the control. */
  hint: string;
  /** The form can be sent without it. */
  optional: boolean;
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
  /**
   * The value the agent would use, offered in the empty composer for the
   * person to accept with one key. Only a value in words with no choices has
   * one; anywhere else it is empty.
   */
  suggestion: string;
  /** How the person's own value is entered when they type one. */
  input: AskInput;
  /** The values asked for together. When there are any, there are no options and no own value. */
  fields: AskField[];
};

/** A question in the thread, and the message that answered it once one has. */
export type ThreadAsk = {
  request: ThreadAskRequest;
  answer: AssistantMessage | null;
};

/** Every question a thread holds, by the reply that asked it, and the one still waiting. */
export type ThreadAsks = {
  /** Each reply's questions in the order it asked them, keyed by the reply's message id. */
  byReply: ReadonlyMap<string, readonly ThreadAsk[]>;
  /** The newest question nothing the person has said since has answered. */
  open: ThreadAskRequest | null;
};

const ASK_TOOL = "ask_user";

/** The contract's bound on one form; a longer one is a page, not a question. */
const MAX_ASK_FIELDS = 8;

/** The contract's bound on a suggestion; a longer one was not meant to be taken at a glance. */
const MAX_ASK_SUGGESTION_CHARS = 120;

const TEXT_INPUT: AskInput = { type: "text" };

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

/**
 * The newest question a reply being written has finished asking. A call still
 * running has no result to read, and one that failed asked nothing.
 */
export function liveAskRequest(segments: readonly TurnSegment[]): ThreadAskRequest | null {
  const steps = segments.flatMap((segment) =>
    segment.kind === "tool" && segment.status === "done"
      ? [{ id: segment.callId, name: segment.name, content: segment.content }]
      : [],
  );

  return askRequestsFromSteps(steps).at(-1) ?? null;
}

/**
 * Something the person said in the conversation, which settles any question
 * asked before it — whether they clicked one of its options or typed
 * something else entirely. Comparing positions rather than tracking which
 * button was pressed is what makes a reopened thread render the same as a
 * live one.
 */
export function isPersonMessage(message: AssistantMessage): boolean {
  return message.role === "User" && message.kind === "Message";
}

/**
 * Each question in the thread with the message that answered it: the first
 * thing the person said after it was asked. The newest question with no such
 * message is the one still waiting.
 */
export function threadAsks(entries: readonly ThreadEntry[]): ThreadAsks {
  const said: AssistantMessage[] = [];
  const asked: { replyId: string; request: ThreadAskRequest }[] = [];

  for (const entry of entries) {
    if (isPersonMessage(entry.message)) {
      said.push(entry.message);
    }
    if (entry.kind === "assistant") {
      for (const request of askRequestsFrom(entry.tools)) {
        asked.push({ replyId: entry.message.id, request });
      }
    }
  }

  const byReply = new Map<string, ThreadAsk[]>();
  let open: ThreadAskRequest | null = null;
  for (const { replyId, request } of asked) {
    const answer = said.find((message) => message.sequence > request.sequence) ?? null;
    const asks = byReply.get(replyId);
    if (asks) {
      asks.push({ request, answer });
    } else {
      byReply.set(replyId, [{ request, answer }]);
    }
    if (answer === null && (open === null || request.sequence >= open.sequence)) {
      open = request;
    }
  }

  return { byReply, open };
}

/**
 * The messages that answered a question. The question's record shows what
 * was said under it, so the message is not drawn again as a new exchange.
 */
export function askAnswerIds(asks: ThreadAsks): ReadonlySet<string> {
  const ids = new Set<string>();
  for (const replyAsks of asks.byReply.values()) {
    for (const { answer } of replyAsks) {
      if (answer !== null) {
        ids.add(answer.id);
      }
    }
  }

  return ids;
}

/**
 * Whether the person answers in the composer: a value of their own, in words,
 * with nothing on the question to fill in. The composer is the box then; the
 * question does not draw a second one.
 */
export function answersInComposer(request: ThreadAskRequest): boolean {
  return request.fields.length === 0 && request.allowOther && request.input.type === "text";
}

type ParsedAsk = Omit<ThreadAskRequest, "sequence" | "callId">;

/**
 * A question a model composed is not a typed contract. A malformed option or
 * field is dropped rather than failing the whole prompt, and a question with
 * nothing left to answer it is not rendered at all — an empty card is worse
 * than the sentence the assistant wrote alongside it.
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

  const otherHint = typeof payload.otherHint === "string" ? payload.otherHint : "";
  const input = inputFrom(payload.input);

  // A form is the whole answer: the values it asks for complete one change,
  // so a choice or a value of the person's own beside it has nothing to mean.
  const fields = fieldsFrom(payload.fields);
  if (fields.length > 0) {
    return {
      question,
      options: [],
      allowOther: false,
      otherHint,
      suggestion: "",
      input,
      fields,
    };
  }

  const options = optionsFrom(payload.options);
  const allowOther = payload.allowOther !== false;
  if (options.length === 0 && !allowOther) {
    return null;
  }

  const suggestion =
    options.length === 0 && allowOther && input.type === "text"
      ? suggestionFrom(payload.suggestion)
      : "";

  return { question, options, allowOther, otherHint, suggestion, input, fields: [] };
}

/**
 * A suggestion is accepted whole with one key, so it is one line, and one the
 * contract would not have sent is dropped rather than cut short.
 */
function suggestionFrom(raw: unknown): string {
  if (typeof raw !== "string") {
    return "";
  }
  const suggestion = raw.trim().replace(/\s+/g, " ");
  return Array.from(suggestion).length > MAX_ASK_SUGGESTION_CHARS ? "" : suggestion;
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

/**
 * How a value is typed. Anything this client does not know how to draw is
 * read as words, which every model can take back: an unknown type, a record
 * with no kind, or a kind of record the @ search does not offer.
 */
function inputFrom(raw: unknown): AskInput {
  if (typeof raw !== "object" || raw === null) {
    return TEXT_INPUT;
  }

  const { type, recordKind } = raw as Record<string, unknown>;
  if (type === "number" || type === "date" || type === "datetime") {
    return { type };
  }
  if (type === "record" && typeof recordKind === "string" && isMentionKind(recordKind)) {
    return { type: "record", recordKind };
  }

  return TEXT_INPUT;
}

function fieldsFrom(raw: unknown): AskField[] {
  if (!Array.isArray(raw)) {
    return [];
  }

  const fields: AskField[] = [];
  const keys = new Set<string>();
  for (const entry of raw) {
    if (fields.length === MAX_ASK_FIELDS) {
      break;
    }
    if (typeof entry !== "object" || entry === null) {
      continue;
    }
    const field = entry as Record<string, unknown>;
    const key = typeof field.key === "string" ? field.key.trim() : "";
    if (key === "" || keys.has(key)) {
      continue;
    }
    keys.add(key);
    const label = typeof field.label === "string" ? field.label.trim() : "";
    fields.push({
      ...inputFrom(field),
      key,
      label: label !== "" ? label : key,
      hint: typeof field.hint === "string" ? field.hint.trim() : "",
      optional: field.optional === true,
    });
  }

  return fields;
}
