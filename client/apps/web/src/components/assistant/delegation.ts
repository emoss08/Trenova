import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { toSentenceFragment } from "@trenova/shared/lib/utils";
import {
  RECORD_LINKS,
  isRecordEntityType,
  recordPath,
  type RecordEntityType,
} from "@/config/record-links";
import {
  artifactKindSchema,
  assistantDelegateFinishedEventSchema,
  type ArtifactKind,
  type AssistantMessage,
  type DelegateDocument,
  type DelegateReport,
  type DelegateStatus,
  type DelegateWrite,
  type ToolExecutionResult,
} from "@/types/assistant";
import {
  delegateName,
  madeChange,
  stepsFromExchanges,
  stepsFromSegments,
  type ToolStep,
} from "./activity";
import type { ToolExchange } from "./thread-view";
import { describeToolCall, parseToolResult } from "./tool-presentation";
import type { TurnRetry, TurnSegment } from "./turn-stream";

/**
 * How a hand-off stands: under way, one of the server's endings, or unknown
 * for a saved hand-off whose account is not in view.
 */
export type DelegateOutcome = DelegateStatus | "running" | "unknown";

/** What the other agent is doing this moment, for the line under a hand-off in progress. */
export type DelegatePhase = "starting" | "thinking" | "writing" | "working" | "reading";

/**
 * A hand-off as the transcript draws it, live or saved: who was asked, what
 * they were asked, the steps they took, what they answered and what came of
 * it. Both sources become this one shape, so a past conversation reads the
 * same as the one being watched.
 */
export type DelegateView = {
  callId: string;
  agentId: string;
  agentName: string;
  /**
   * The agent's mark as the stream announced it or the thread served it; empty
   * when neither said, and the transcript falls back to the conversation
   * agent's list of delegates, then to a mark derived from the id.
   */
  icon: string;
  accent: string;
  task: string;
  /** The other agent's own calls. */
  steps: ToolStep[];
  /** Its answer: the account's, or the words it has written so far. */
  reply: string;
  phase: DelegatePhase;
  retrying: TurnRetry | null;
  report: DelegateReport | null;
  outcome: DelegateOutcome;
  /** Why it did not finish, when it did not. */
  reason: string;
};

const NOT_RUN = /^Tool "[^"]*" was not run: /u;

/**
 * The account of a task, taken back out of the delegate_task call's result
 * text. A conversation saved since the account was kept on the result reads
 * `delegateReport` instead; this is only for one saved before. The delegating
 * agent reads the same object the reader was shown, fenced as untrusted data
 * with a note after it, so an older conversation recovers it from there. A
 * result that is not the account — one cut short, or a refusal to hand the
 * task over at all — reads as none.
 */
export function parseDelegateReport(content: string): DelegateReport | null {
  if (content === "") {
    return null;
  }
  const parsed = parseToolResult(content);
  if (parsed.kind !== "json") {
    return null;
  }
  const report = assistantDelegateFinishedEventSchema.safeParse(parsed.value);

  return report.success ? report.data : null;
}

/** Why a task was never handed over, when the call was refused before it began. */
export function refusedHandOff(content: string): string | null {
  const match = NOT_RUN.exec(content);

  return match ? content.slice(match[0].length).trim() : null;
}

function stringArgument(step: ToolStep, key: string): string {
  const value = step.arguments?.[key];

  return typeof value === "string" ? value.trim() : "";
}

/** The words of the last message written, open or finished. */
function lastText(segments: readonly TurnSegment[]): string {
  for (let index = segments.length - 1; index >= 0; index -= 1) {
    const segment = segments[index];
    if (segment.kind === "text" && segment.text.trim() !== "") {
      return segment.text;
    }
  }

  return "";
}

function livePhase(segments: readonly TurnSegment[]): DelegatePhase {
  const last = segments.at(-1);
  if (!last) {
    return "starting";
  }
  if (last.kind === "reasoning") {
    return last.closed ? "reading" : "thinking";
  }
  if (last.kind === "text") {
    return last.closed ? "reading" : "writing";
  }

  return last.status === "running" ? "working" : "reading";
}

/**
 * How the hand-off ended when the account is not in hand: a refusal to hand
 * it over, a failed call, or — for a call that settled without one — an
 * ending the transcript cannot name.
 */
function outcomeWithoutReport(step: ToolStep): DelegateOutcome {
  if (step.status === "running") {
    return "running";
  }
  if (refusedHandOff(step.content) !== null) {
    return "declined";
  }
  if (step.status === "failed") {
    return "failed";
  }

  return "unknown";
}

function reasonOf(step: ToolStep, report: DelegateReport | null): string {
  if (report && report.reason !== "") {
    return report.reason;
  }
  const refused = refusedHandOff(step.content);
  if (refused !== null) {
    return refused;
  }
  if (step.status === "failed" && step.content !== "") {
    const parsed = parseToolResult(step.content);
    return parsed.kind === "error" ? parsed.message : "";
  }

  return "";
}

/**
 * The saved steps of the other agent as calls. Each call is timed from the
 * message that asked for it; a result whose call is out of view is still
 * shown, untimed.
 */
function savedSteps(messages: readonly AssistantMessage[]): ToolStep[] {
  const groups: { askedAt: number; exchanges: ToolExchange[] }[] = [];
  const open = new Map<string, ToolExchange>();
  for (const message of messages) {
    if (message.role === "Assistant") {
      const exchanges = (message.toolCalls ?? []).map((call) => {
        const exchange: ToolExchange = { call, result: null };
        open.set(call.id, exchange);
        return exchange;
      });
      if (exchanges.length > 0) {
        groups.push({ askedAt: message.createdAt, exchanges });
      }
      continue;
    }
    if (message.role !== "Tool") {
      continue;
    }
    const exchange = open.get(message.toolCallId);
    if (exchange) {
      exchange.result = message;
      open.delete(message.toolCallId);
      continue;
    }
    groups.push({
      askedAt: 0,
      exchanges: [
        {
          call: { id: message.toolCallId, name: message.toolName, arguments: {} },
          result: message,
          orphan: true,
        },
      ],
    });
  }

  return groups.flatMap((group) => stepsFromExchanges(group.exchanges, group.askedAt));
}

/** The last thing the other agent said, from a saved thread. */
function savedReply(messages: readonly AssistantMessage[]): string {
  for (let index = messages.length - 1; index >= 0; index -= 1) {
    const message = messages[index];
    if (message.role === "Assistant" && message.content.trim() !== "") {
      return message.content;
    }
  }

  return "";
}

/**
 * The other agent's answer from a saved thread: the account's, unless the
 * account kept it cut short and its own saved message holds more of it.
 */
function savedAnswer(report: DelegateReport | null, messages: readonly AssistantMessage[]): string {
  const kept = report?.reply ?? "";
  const written = savedReply(messages);
  if (kept === "") {
    return written;
  }

  return kept.endsWith("…") && written.length > kept.length ? written : kept;
}

/**
 * The other agent's mark from a saved thread: the one its steps were served
 * with, else the one its account carries. Empty when neither says.
 */
function savedMark(
  messages: readonly AssistantMessage[],
  report: DelegateReport | null,
): { icon: string; accent: string } {
  const step = messages.find(
    (message) => (message.agentIcon ?? "") !== "" || (message.agentAccent ?? "") !== "",
  );

  return {
    icon: step?.agentIcon || report?.icon || "",
    accent: step?.agentAccent || report?.accent || "",
  };
}

/**
 * A hand-off as the transcript draws it, from the delegate_task call it
 * hangs under. Null for any other call.
 */
export function delegateView(step: ToolStep): DelegateView | null {
  const source = step.delegate;
  if (!source) {
    return null;
  }

  if (source.kind === "live") {
    const { progress } = source;
    const report = progress.report ?? parseDelegateReport(step.content);
    return {
      callId: step.id,
      agentId: progress.agentId || stringArgument(step, "agentId"),
      agentName: delegateName(step),
      icon: progress.icon || report?.icon || "",
      accent: progress.accent || report?.accent || "",
      task: progress.task || stringArgument(step, "task"),
      steps: stepsFromSegments(progress.segments),
      reply: report?.reply || lastText(progress.segments),
      phase: livePhase(progress.segments),
      retrying: report ? null : progress.retrying,
      report,
      outcome: report?.status ?? outcomeWithoutReport(step),
      reason: reasonOf(step, report),
    };
  }

  const { messages } = source;
  const report = source.report ?? parseDelegateReport(step.content);
  const task = messages.find((message) => message.role === "User")?.content.trim() ?? "";
  const mark = savedMark(messages, report);

  return {
    callId: step.id,
    agentId:
      messages.find((message) => (message.agentId ?? "") !== "")?.agentId ||
      report?.agentId ||
      stringArgument(step, "agentId"),
    agentName: delegateName(step),
    icon: mark.icon,
    accent: mark.accent,
    task: task || stringArgument(step, "task"),
    steps: savedSteps(messages),
    reply: savedAnswer(report, messages),
    phase: "reading",
    retrying: null,
    report,
    outcome: report?.status ?? outcomeWithoutReport(step),
    reason: reasonOf(step, report),
  };
}

export type HandOffTone = "muted" | "warning" | "danger";

/** Who was asked, in the words of how it went. */
export function handOffHeadline(view: DelegateView, t: TranslateFn): string {
  const name = view.agentName;
  switch (view.outcome) {
    case "running":
      return name !== "" ? t("Asking {0}…", name) : t("Asking another agent…");
    case "declined":
      return name !== "" ? t("Couldn't ask {0}", name) : t("Couldn't ask another agent");
    default:
      return name !== "" ? t("Asked {0}", name) : t("Asked another agent");
  }
}

/**
 * How the task ended, said once and toned by what the person should make of
 * it: finished is quiet, an answer cut short is a warning, one that never ran
 * or broke off is the danger tone. A running or unknown hand-off says nothing.
 */
export function handOffStatus(
  view: DelegateView,
  t: TranslateFn,
): { text: string; tone: HandOffTone } | null {
  const reason = view.reason;
  switch (view.outcome) {
    case "completed":
      return { text: t("Finished"), tone: "muted" };
    case "exhausted":
      return {
        text: t("Used every step its settings allow before it finished"),
        tone: "warning",
      };
    case "refused":
      return { text: t("Its answer was withheld"), tone: "warning" };
    case "declined":
      return {
        text: reason !== "" ? t("Could not be asked: {0}", reason) : t("Could not be asked"),
        tone: "danger",
      };
    case "failed":
      return {
        text: reason !== "" ? t("Stopped partway: {0}", reason) : t("Stopped partway"),
        tone: "danger",
      };
    case "stopped":
      return { text: t("You stopped it before it finished"), tone: "warning" };
    default:
      return null;
  }
}

/** What the other agent is doing now, for the line under its steps. */
export function handOffWorking(view: DelegateView, t: TranslateFn): string {
  const retrying = view.retrying;
  if (retrying) {
    return retrying.kind === "busy" ? t("Waiting for the model…") : t("Starting over…");
  }
  switch (view.phase) {
    case "starting":
      return t("Reading the task…");
    case "thinking":
      return t("Thinking…");
    case "writing":
      return t("Writing its answer…");
    case "working":
      return t("Working…");
    default:
      return t("Reading what came back…");
  }
}

function recordEntityOf(kind: string): RecordEntityType | null {
  const key = kind
    .trim()
    .toLowerCase()
    .replaceAll(/[\s-]+/gu, "_");

  return key !== "" && isRecordEntityType(key) ? key : null;
}

function camelKey(entity: RecordEntityType): string {
  return `${entity.replaceAll(/_([a-z])/gu, (_match, letter: string) => letter.toUpperCase())}Id`;
}

/** The registry entity a result names outright, when it names one the app opens. */
function namedRecord(result: ToolExecutionResult): { entity: RecordEntityType; id: string } | null {
  const record = result.record;
  if (!record) {
    return null;
  }
  const id = record.id.trim();

  return id !== "" && isRecordEntityType(record.entityType)
    ? { entity: record.entityType, id }
    : null;
}

/**
 * Where the record a write made opens, from the registry. A result that
 * names its record (`record`) is linked by it. One from a tool that does not
 * is read the old way: its kind must be a record the app has a page for, and
 * its id is taken by the name the record's own tools use (`reportId`), then a
 * plain `id`, then the only id there is; several unnamed ids name nothing
 * certain.
 */
export function madeRecordPath(result: ToolExecutionResult | null | undefined): string | null {
  if (!result) {
    return null;
  }
  const named = namedRecord(result);
  if (named !== null) {
    return recordPath(named.entity, named.id);
  }
  const entity = recordEntityOf(result.kind);
  if (entity === null) {
    return null;
  }
  const ids = result.ids;
  const values = Object.values(ids).filter((value) => value.trim() !== "");
  const id = ids[camelKey(entity)] || ids.id || (values.length === 1 ? values[0] : "") || "";

  return id === "" ? null : recordPath(entity, id);
}

/** A record kind in the words of the registry, as a fragment: "report", "rate matrix". */
function kindWord(result: ToolExecutionResult | null | undefined, t: TranslateFn): string {
  const entity = result ? (namedRecord(result)?.entity ?? recordEntityOf(result.kind)) : null;

  return entity === null ? "" : toSentenceFragment(t(RECORD_LINKS[entity].label));
}

export type WriteLine = {
  /** The whole line: "Created report Shipments for Peak Distributing". */
  text: string;
  /** The record's name within it, linked when `path` is set. */
  subject: string;
  path: string | null;
  state: "made" | "failed" | "simulated" | "awaiting";
  /** Why a write that ran did not go through. */
  error: string;
};

function writeSubject(write: DelegateWrite): string {
  const named = write.result?.name.trim() ?? "";
  if (named !== "") {
    return named;
  }

  return write.summary.trim();
}

/**
 * The sentence for a write whose result names its action, in our verb and
 * translated: the server's action is a key, never shown. Empty for an
 * action this client does not know, which is then worded from the tool.
 */
function actionSentence(action: string, kind: string, subject: string, t: TranslateFn): string {
  const what = [kind, subject].filter((part) => part !== "").join(" ");
  if (what === "") {
    return "";
  }
  switch (action.trim().toLowerCase()) {
    case "created":
      return t("Created {0}", what);
    case "updated":
      return t("Updated {0}", what);
    case "saved":
      return t("Saved {0}", what);
    case "deleted":
    case "removed":
      return t("Removed {0}", what);
    case "sent":
      return t("Sent {0}", what);
    case "added":
      return t("Added {0}", what);
    default:
      return "";
  }
}

/**
 * One write the other agent made, in the verb of the write. The result's
 * action and kind give the verb and the noun; a write whose tool says
 * neither is worded from its tool, the same way the turn's own changes are.
 * A write that failed or was only previewed says so, and is never worded as
 * made.
 */
export function madeLine(write: DelegateWrite, t: TranslateFn): WriteLine {
  const subject = writeSubject(write);

  if (write.error !== "") {
    return {
      text: subject !== "" ? t("{0} didn't go through", subject) : t("A change didn't go through"),
      subject: "",
      path: null,
      state: "failed",
      error: write.error,
    };
  }
  if (write.simulated) {
    return {
      text: subject !== "" ? t("Previewed a change to {0}", subject) : t("Previewed a change"),
      subject: "",
      path: null,
      state: "simulated",
      error: "",
    };
  }

  const path = madeRecordPath(write.result);
  const sentence = actionSentence(
    write.result?.action ?? "",
    kindWord(write.result, t),
    subject,
    t,
  );

  return {
    text: sentence !== "" ? sentence : madeChange(write.toolName, subject, t),
    subject,
    path,
    state: "made",
    error: "",
  };
}

/** One write that waits on the person, named by its tool and what it is about. */
export function awaitingLine(write: DelegateWrite, t: TranslateFn): WriteLine {
  const title = describeToolCall(write.toolName, {}).title;
  const subject = writeSubject(write);

  return {
    text: subject !== "" ? t("{0}: {1}", title, subject) : title,
    subject: "",
    path: null,
    state: "awaiting",
    error: "",
  };
}

/**
 * The line split around its linked name, so the name alone is the link. A
 * translation that does not repeat the name verbatim is drawn unlinked.
 */
export function splitOnSubject(
  line: WriteLine,
): { before: string; subject: string; after: string } | null {
  if (line.path === null || line.subject === "") {
    return null;
  }
  const at = line.text.lastIndexOf(line.subject);
  if (at === -1) {
    return null;
  }

  return {
    before: line.text.slice(0, at),
    subject: line.subject,
    after: line.text.slice(at + line.subject.length),
  };
}

/** The artifact kind a published document is, when it is one the pane can open. */
export function publishedKind(document: DelegateDocument): ArtifactKind | null {
  const parsed = artifactKindSchema.safeParse(document.kind);

  return parsed.success ? parsed.data : null;
}

/**
 * Whether the hand-off has anything to show beyond its headline: a task, a
 * step, an answer or an outcome. One that was refused before it began has
 * only its reason.
 */
export function handOffHasDetail(view: DelegateView): boolean {
  return view.task !== "" || view.steps.length > 0 || view.reply.trim() !== "";
}
