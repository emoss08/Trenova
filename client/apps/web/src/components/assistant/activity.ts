import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { AssistantMessage, ToolEffect } from "@/types/assistant";
import type { ToolExchange } from "./thread-view";
import { describeToolCall, parseToolResult } from "./tool-presentation";
import {
  DELEGATE_TOOL,
  emptyDelegateProgress,
  type DelegateProgress,
  type TurnSegment,
} from "./turn-stream";

export type ToolActivityStatus = "running" | "done" | "failed" | "proposed";

/** One call the agent made, live or saved, in the shape the transcript draws. */
export type ToolStep = {
  id: string;
  name: string;
  arguments: Record<string, unknown> | null | undefined;
  status: ToolActivityStatus;
  /** The saved or streamed result. Empty while running. */
  content: string;
  /** What the call did; derived from the name when the server did not say. */
  effect: ToolEffect;
  /** The server's one-line account of the result, in English; empty when it gave none. */
  summary: string;
  /** How long the call took, in seconds, when that is known. */
  durationSeconds: number | null;
  /**
   * On a delegate_task call: the other agent's work on the task, as the
   * stream delivered it or as the thread saved it. The hand-off is drawn
   * from it, under the call and never among the turn's own steps.
   */
  delegate?: DelegateSource;
};

export type DelegateSource =
  | { kind: "live"; progress: DelegateProgress }
  | { kind: "saved"; messages: readonly AssistantMessage[] };

/**
 * The effects of tools that predate the server naming them, or that it no
 * longer knows. Named tools first; then the verbs a read begins with; and
 * anything else is assumed to change something, because reading a write as a
 * lookup is the mistake this whole classification exists to stop.
 */
const NAMED_EFFECTS: Readonly<Record<string, ToolEffect>> = {
  open_page: "navigate",
  find_tools: "discover",
  find_in_trenova: "discover",
  run_report: "present",
  publish_artifact: "present",
  compare_report_runs: "present",
  compose_table_view: "present",
  ask_user: "ask",
  [DELEGATE_TOOL]: "delegate",
};

const LOOKUP_PREFIXES = ["list_", "get_", "search_", "describe_", "preview_"];

export function toolEffect(name: string, effect?: ToolEffect | null): ToolEffect {
  if (effect) {
    return effect;
  }
  const named = NAMED_EFFECTS[name];
  if (named) {
    return named;
  }

  return LOOKUP_PREFIXES.some((prefix) => name.startsWith(prefix)) ? "lookup" : "change";
}

/** How a saved call ended: a proposal is recorded rather than run. */
function savedStatus(result: AssistantMessage | null): ToolActivityStatus {
  if (result === null) return "done";
  if (result.toolFailed) return "failed";
  return result.content.startsWith("Recorded a proposal") ? "proposed" : "done";
}

/**
 * A saved turn's calls as steps. A step's time is from the message that asked
 * for it to the result that answered it, both stamped when they were made; a
 * result whose call is not in view has nothing to measure from.
 */
export function stepsFromExchanges(tools: readonly ToolExchange[], askedAt: number): ToolStep[] {
  return tools.map(({ call, result, orphan, delegated }) => {
    const name = call.name !== "" ? call.name : (result?.toolName ?? "");
    const timed = !orphan && result !== null && result.createdAt > 0 && askedAt > 0;
    return {
      id: call.id,
      name,
      arguments: call.arguments,
      status: savedStatus(result),
      content: result?.content ?? "",
      effect: toolEffect(name, call.effect ?? result?.effect),
      summary: result?.summary ?? "",
      durationSeconds: timed ? Math.max(0, result.createdAt - askedAt) : null,
      ...(name === DELEGATE_TOOL
        ? { delegate: { kind: "saved" as const, messages: delegated ?? [] } }
        : {}),
    };
  });
}

/** The calls of a turn still streaming, timed by the reader's own clock. */
export function stepsFromSegments(segments: readonly TurnSegment[]): ToolStep[] {
  return segments.flatMap((segment) => (segment.kind === "tool" ? [segmentStep(segment)] : []));
}

export function segmentStep(segment: Extract<TurnSegment, { kind: "tool" }>): ToolStep {
  const timed = segment.startedAt !== undefined && segment.finishedAt !== undefined;
  return {
    id: segment.callId,
    name: segment.name,
    arguments: segment.arguments,
    status: segment.status,
    content: segment.content,
    effect: toolEffect(segment.name, segment.effect),
    summary: segment.summary ?? "",
    durationSeconds: timed ? (segment.finishedAt! - segment.startedAt!) / 1000 : null,
    ...(segment.delegate || segment.name === DELEGATE_TOOL
      ? {
          delegate: {
            kind: "live" as const,
            progress:
              segment.delegate ?? emptyDelegateProgress(stringOf(segment.arguments.agentId)),
          },
        }
      : {}),
  };
}

function stringOf(value: unknown): string {
  return typeof value === "string" ? value : "";
}

/** The effects that did something, as opposed to reading or asking. */
export function isActionEffect(effect: ToolEffect): boolean {
  return (
    effect === "change" || effect === "navigate" || effect === "present" || effect === "delegate"
  );
}

export type SummaryCount = { count: number; more: boolean };

const COUNT_SUMMARY = /^(\d+\+?|No) /u;

/**
 * The count a list's summary carries ("3 customers", "25+ shipments", "No
 * customers"), or null for a summary that names something. The noun is the
 * server's English and is never shown; the number is.
 */
export function summaryCount(summary: string): SummaryCount | null {
  const match = COUNT_SUMMARY.exec(summary);
  if (!match) {
    return null;
  }
  if (match[1] === "No") {
    return { count: 0, more: false };
  }

  return {
    count: Number.parseInt(match[1], 10),
    more: match[1].endsWith("+"),
  };
}

const REPORT_SUMMARY = /^(.+) · (\d+) rows?$/u;

/** A report run's summary, "Late loads · 42 rows", split into its name and row count. */
export function reportSummary(summary: string): { name: string; rows: number | null } {
  const match = REPORT_SUMMARY.exec(summary);
  if (!match) {
    return { name: summary, rows: null };
  }

  return { name: match[1], rows: Number.parseInt(match[2], 10) };
}

/** A summary that names something rather than counting it, or empty. */
export function summaryName(summary: string): string {
  const trimmed = summary.trim();

  return trimmed === "" || summaryCount(trimmed) !== null ? "" : trimmed;
}

/**
 * Consecutive steps drawn as one line. Reads fold together, because four
 * lookups are one piece of looking; every action stands on its own line,
 * because "opened a page" folded into "looked up 3 records" is the report
 * that started this.
 */
export type ActivityGroup = {
  key: string;
  effect: ToolEffect;
  steps: ToolStep[];
};

function foldsWith(previous: ToolStep, next: ToolStep): boolean {
  if (previous.effect !== next.effect) {
    return false;
  }
  if (next.effect === "lookup") {
    return true;
  }

  return next.effect === "discover" && previous.name === next.name;
}

export function groupActivity(steps: readonly ToolStep[]): ActivityGroup[] {
  const groups: ActivityGroup[] = [];
  for (const step of steps) {
    const last = groups.at(-1);
    const previous = last?.steps.at(-1);
    if (last && previous && foldsWith(previous, step)) {
      last.steps.push(step);
      continue;
    }
    groups.push({ key: step.id, effect: step.effect, steps: [step] });
  }

  return groups;
}

export type ActivityLine = {
  /** What happened, as a sentence fragment: "Looked up Peak Distributing". */
  phrase: string;
  /** Quieter context after it: a filter, a row count, the names behind a count. */
  detail: string;
  /** Said in the danger tone after the detail: how many of the steps failed. */
  failure: string;
  state: "running" | "done" | "failed" | "proposed";
};

/** How far a list of names is spelled out before the rest become a count. */
const NAMED_LIMIT = 2;

function subjectOf(step: ToolStep): string {
  const named = summaryName(step.summary);
  if (named !== "") {
    return named;
  }

  return describeToolCall(step.name, step.arguments).subject;
}

function titleOf(step: ToolStep): string {
  return describeToolCall(step.name, step.arguments).title;
}

/** Whether a read returns many records rather than one. */
function readsMany(step: ToolStep): boolean {
  if (summaryCount(step.summary) !== null) {
    return true;
  }
  if (summaryName(step.summary) !== "") {
    return false;
  }

  return step.name.startsWith("list_") || step.name.startsWith("search_");
}

/** The failure message a call returned, for the line's detail. */
export function failureMessage(step: ToolStep): string {
  if (step.content === "") {
    return "";
  }
  const parsed = parseToolResult(step.content);

  return parsed.kind === "error" ? parsed.message : "";
}

/**
 * The name the result itself carries, for a saved step the server did not
 * summarize: where a page went, what a report is called.
 */
function resultName(step: ToolStep): string {
  if (step.content === "" || step.status === "running") {
    return "";
  }
  const parsed = parseToolResult(step.content);
  if (parsed.kind !== "json" || typeof parsed.value !== "object" || parsed.value === null) {
    return "";
  }
  const record = parsed.value as Record<string, unknown>;
  for (const key of ["name", "reportName", "title", "label"]) {
    const value = record[key];
    if (typeof value === "string" && value.trim() !== "") {
      return value.trim();
    }
  }

  return "";
}

function joinNames(names: readonly string[]): string {
  const unique = [...new Set(names.filter((name) => name !== ""))];
  const shown = unique.slice(0, NAMED_LIMIT).join(", ");
  const hidden = unique.length - NAMED_LIMIT;

  return hidden > 0 ? `${shown} +${hidden}` : shown;
}

function lookupLine(group: ActivityGroup, t: TranslateFn): ActivityLine {
  const running = group.steps.filter((step) => step.status === "running");
  const failed = group.steps.filter((step) => step.status === "failed");
  const settled = group.steps.filter(
    (step) => step.status !== "running" && step.status !== "failed",
  );
  const failure =
    failed.length > 0 && settled.length + running.length > 0 ? t("{0} failed", failed.length) : "";

  if (running.length > 0) {
    const current = running.at(-1)!;
    const subject = subjectOf(current);
    if (group.steps.length > 1 || subject === "") {
      return { phrase: t("Looking things up…"), detail: subject, failure, state: "running" };
    }
    return readsMany(current)
      ? { phrase: t("Searching records…"), detail: subject, failure, state: "running" }
      : { phrase: t("Looking up {0}…", subject), detail: "", failure, state: "running" };
  }

  if (settled.length === 0) {
    const first = failed[0];
    return {
      phrase: t("Couldn't look that up"),
      detail: failureMessage(first) || titleOf(first),
      failure: "",
      state: "failed",
    };
  }

  if (settled.length === 1) {
    const step = settled[0];
    const counted = summaryCount(step.summary);
    const subject = describeToolCall(step.name, step.arguments).subject;
    if (counted) {
      return {
        phrase:
          counted.count === 0
            ? t("Found nothing")
            : counted.more
              ? t("{0, plural, one {Found #+ record} other {Found #+ records}}", counted.count)
              : t("{0, plural, one {Found # record} other {Found # records}}", counted.count),
        detail: subject || titleOf(step),
        failure,
        state: "done",
      };
    }
    const named = summaryName(step.summary);
    if (named !== "") {
      return { phrase: t("Looked up {0}", named), detail: "", failure, state: "done" };
    }
    if (readsMany(step)) {
      return {
        phrase: t("Searched records"),
        detail: subject || titleOf(step),
        failure,
        state: "done",
      };
    }
    return subject !== ""
      ? { phrase: t("Looked up {0}", subject), detail: "", failure, state: "done" }
      : { phrase: t("Looked up a record"), detail: titleOf(step), failure, state: "done" };
  }

  const names = joinNames(settled.filter((step) => !readsMany(step)).map(subjectOf));
  const phrase = settled.some(readsMany)
    ? t("{0, plural, one {Ran # lookup} other {Ran # lookups}}", settled.length)
    : t("{0, plural, one {Looked up # record} other {Looked up # records}}", settled.length);

  return { phrase, detail: names, failure, state: "done" };
}

function discoverLine(group: ActivityGroup, t: TranslateFn): ActivityLine {
  const step = group.steps.at(-1)!;
  const guide = step.name === "find_in_trenova";

  if (step.status === "running") {
    return {
      phrase: guide ? t("Checking the product guide…") : t("Finding the right tools…"),
      detail: "",
      failure: "",
      state: "running",
    };
  }
  if (step.status === "failed") {
    return {
      phrase: guide ? t("Couldn't check the product guide") : t("Couldn't find a tool for it"),
      detail: failureMessage(step),
      failure: "",
      state: "failed",
    };
  }

  return {
    phrase: guide ? t("Checked the product guide") : t("Found the right tools"),
    detail: guide ? summaryName(step.summary) : "",
    failure: "",
    state: "done",
  };
}

function navigateLine(step: ToolStep, t: TranslateFn): ActivityLine {
  if (step.status === "running") {
    return { phrase: t("Opening a page…"), detail: "", failure: "", state: "running" };
  }
  if (step.status === "failed") {
    return {
      phrase: t("Couldn't open that page"),
      detail: failureMessage(step),
      failure: "",
      state: "failed",
    };
  }
  const destination = summaryName(step.summary) || resultName(step);

  return {
    phrase: destination !== "" ? t("Opened {0}", destination) : t("Opened a page"),
    detail: "",
    failure: "",
    state: "done",
  };
}

function presentLine(step: ToolStep, t: TranslateFn): ActivityLine {
  const named = summaryName(step.summary) || resultName(step);

  if (step.status === "failed") {
    return {
      phrase: t("Couldn't put that together"),
      detail: failureMessage(step) || titleOf(step),
      failure: "",
      state: "failed",
    };
  }

  switch (step.name) {
    case "run_report": {
      const report = reportSummary(named);
      if (step.status === "running") {
        const subject = describeToolCall(step.name, step.arguments).subject;
        return {
          phrase: t("Running a report…"),
          detail: subject,
          failure: "",
          state: "running",
        };
      }
      return {
        phrase:
          report.name === ""
            ? t("Ran a report")
            : report.rows === null
              ? t("Started {0}", report.name)
              : t("Ran {0}", report.name),
        detail:
          report.rows === null ? "" : t("{0, plural, one {# row} other {# rows}}", report.rows),
        failure: "",
        state: "done",
      };
    }
    case "publish_artifact":
      return step.status === "running"
        ? { phrase: t("Writing it up…"), detail: "", failure: "", state: "running" }
        : {
            phrase: named !== "" ? t("Published {0}", named) : t("Published a document"),
            detail: "",
            failure: "",
            state: "done",
          };
    case "compose_table_view":
      return step.status === "running"
        ? { phrase: t("Building a table…"), detail: "", failure: "", state: "running" }
        : { phrase: t("Built a table"), detail: named, failure: "", state: "done" };
    case "compare_report_runs":
      return step.status === "running"
        ? { phrase: t("Comparing report runs…"), detail: "", failure: "", state: "running" }
        : { phrase: t("Compared report runs"), detail: named, failure: "", state: "done" };
    default:
      return step.status === "running"
        ? { phrase: t("Putting a result together…"), detail: "", failure: "", state: "running" }
        : {
            phrase: named !== "" ? t("Prepared {0}", named) : t("Prepared a result"),
            detail: "",
            failure: "",
            state: "done",
          };
  }
}

/**
 * What a write did, in the verb of the write. The server's summary names
 * what was written from the model's own arguments; the verb is ours, so it
 * can be translated.
 */
export function madeChange(name: string, subject: string, t: TranslateFn): string {
  if (subject === "") {
    return t("Made a change");
  }
  const verb = name.split("_", 1)[0];
  switch (verb) {
    case "create":
      return name.endsWith("_report") ? t("Saved {0}", subject) : t("Created {0}", subject);
    case "update":
      return name.endsWith("_report") ? t("Saved {0}", subject) : t("Updated {0}", subject);
    case "fork":
      return t("Copied {0}", subject);
    case "add":
      return t("Added {0}", subject);
    case "remove":
      return t("Removed {0}", subject);
    case "cancel":
      return t("Cancelled {0}", subject);
    case "send":
    case "email":
    case "notify":
      return t("Sent {0}", subject);
    case "approve":
      return t("Approved {0}", subject);
    case "reject":
      return t("Declined {0}", subject);
    case "resolve":
      return t("Resolved {0}", subject);
    case "dismiss":
      return t("Dismissed {0}", subject);
    case "record":
      return t("Recorded {0}", subject);
    case "reassign":
      return t("Reassigned {0}", subject);
    case "tender":
      return t("Tendered {0}", subject);
    case "flag":
      return t("Flagged {0}", subject);
    default:
      return t("Changed {0}", subject);
  }
}

function changeLine(step: ToolStep, t: TranslateFn): ActivityLine {
  const title = titleOf(step);

  switch (step.status) {
    case "running":
      return { phrase: t("Making a change…"), detail: title, failure: "", state: "running" };
    case "failed":
      return {
        phrase: t("The change didn't go through"),
        detail: failureMessage(step) || title,
        failure: "",
        state: "failed",
      };
    case "proposed":
      return {
        phrase: t("Proposed a change"),
        detail: summaryName(step.summary) || title,
        failure: "",
        state: "proposed",
      };
    default: {
      const subject = subjectOf(step);
      return {
        phrase: madeChange(step.name, subject, t),
        detail: subject === "" ? title : "",
        failure: "",
        state: "done",
      };
    }
  }
}

function askLine(step: ToolStep, t: TranslateFn): ActivityLine {
  if (step.status === "failed") {
    return {
      phrase: t("Couldn't ask you"),
      detail: failureMessage(step),
      failure: "",
      state: "failed",
    };
  }

  return {
    phrase: step.status === "running" ? t("Asking you…") : t("Asked you to choose"),
    detail: "",
    failure: "",
    state: step.status === "running" ? "running" : "done",
  };
}

/**
 * Who a hand-off went to. The stream names the agent when the task starts
 * and the thread keeps the name on each of its steps; the call's own summary
 * is the name too, for a hand-off whose steps are out of view.
 */
export function delegateName(step: ToolStep): string {
  const source = step.delegate;
  if (source?.kind === "live" && source.progress.agentName !== "") {
    return source.progress.agentName;
  }
  if (source?.kind === "saved") {
    const named = source.messages.find((message) => (message.agentName ?? "") !== "");
    if (named?.agentName) {
      return named.agentName;
    }
  }

  return summaryName(step.summary);
}

/** Whether a hand-off is still under way: its account has not arrived and its call has not settled. */
export function delegateRunning(step: ToolStep): boolean {
  const report = step.delegate?.kind === "live" ? step.delegate.progress.report : null;

  return step.status === "running" && report === null;
}

function delegateLine(step: ToolStep, t: TranslateFn): ActivityLine {
  const name = delegateName(step);

  if (delegateRunning(step)) {
    return {
      phrase: name !== "" ? t("Asking {0}…", name) : t("Asking another agent…"),
      detail: "",
      failure: "",
      state: "running",
    };
  }

  return {
    phrase: name !== "" ? t("Asked {0}", name) : t("Asked another agent"),
    detail: "",
    failure: "",
    state: step.status === "failed" ? "failed" : "done",
  };
}

/** One line for a group: what happened, by what kind of thing happened. */
export function describeActivity(group: ActivityGroup, t: TranslateFn): ActivityLine {
  const step = group.steps[0];
  switch (group.effect) {
    case "lookup":
      return lookupLine(group, t);
    case "discover":
      return discoverLine(group, t);
    case "navigate":
      return navigateLine(step, t);
    case "present":
      return presentLine(step, t);
    case "ask":
      return askLine(step, t);
    case "delegate":
      return delegateLine(step, t);
    default:
      return changeLine(step, t);
  }
}

/**
 * What the agent is doing this moment, for the line under a turn in
 * progress: the step still running, or failing that the last one taken.
 */
export function currentActivity(steps: readonly ToolStep[], t: TranslateFn): ActivityLine | null {
  const groups = groupActivity(steps);
  const live =
    [...groups].reverse().find((group) => group.steps.some((step) => step.status === "running")) ??
    groups.at(-1);

  return live ? describeActivity(live, t) : null;
}
