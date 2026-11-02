import {
  refusalLabel,
  refusalTone,
  toolRefusal,
  type ToolStep,
} from "@/components/assistant/activity";
import { parseToolResult } from "@/components/assistant/tool-presentation";
import { DeskProse } from "@/components/desk-chat/conversation/desk-turns";
import { JsonViewer } from "@/components/elements/json-viewer";
import { VirtualRows } from "@/components/virtual-rows";
import { formatWorkDuration } from "@/lib/ai-usage-format";
import type { ToolEffect } from "@/types/assistant";
import { SearchLgIcon } from "@trenova/shared/components/icons";
import { Input } from "@trenova/shared/components/ui/input";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatCountdownClock } from "@trenova/shared/lib/date";
import { escapeRegExp } from "@trenova/shared/lib/regexp";
import { cn } from "@trenova/shared/lib/utils";
import { useDeferredValue, useState, type ReactNode } from "react";
import { Ic, type IcName } from "../kit/ic";
import {
  countProblems,
  indexEvents,
  onlyProblems,
  rowLine,
  runEvents,
  runStart,
  searchEvents,
  type RunEvent,
  type TranscriptBlock,
} from "./run-transcript";

type Tone = "d" | "w" | "b" | "n";

/** A collapsed row's height before it is measured; an opened one is measured once drawn. */
const ROW_ESTIMATE = 34;
/** The scroll area's height before it is measured, which sets the first window. */
const INITIAL_HEIGHT = 560;

const EFFECT_ICON: Record<ToolEffect, IcName> = {
  lookup: "search",
  change: "edit",
  navigate: "route",
  discover: "sparkle",
  present: "chart",
  ask: "hand",
  delegate: "users",
};

/** How a call ended, said only when it did not simply run. */
function callChip(step: ToolStep, t: TranslateFn): { label: string; tone: Tone } | null {
  const refusal = toolRefusal(step);
  if (refusal) {
    return {
      label: refusalLabel(refusal, t),
      tone: refusalTone(refusal) === "warning" ? "w" : "n",
    };
  }
  if (step.status === "failed") {
    return { label: t("Failed"), tone: "d" };
  }
  if (step.status === "proposed") {
    return { label: t("Proposed"), tone: "b" };
  }
  if (step.verdict === "simulated") {
    return { label: t("Simulated"), tone: "n" };
  }

  return null;
}

function expandableKeys(events: readonly RunEvent[]): string[] {
  return events.flatMap((event) => {
    if (event.kind === "thought") {
      return [event.key];
    }
    if (event.kind === "call") {
      return [event.key, ...expandableKeys(event.delegated)];
    }
    return [];
  });
}

/** Text with each case-blind occurrence of the search marked. */
function Marked({ text, query }: { text: string; query: string }) {
  const needle = query.trim();
  if (needle === "" || text === "") {
    return text;
  }
  const parts = text.split(new RegExp(`(${escapeRegExp(needle)})`, "giu"));

  return parts.map((part, index) =>
    index % 2 === 1 ? (
      <mark key={index} className="rt-hl">
        {part}
      </mark>
    ) : (
      part
    ),
  );
}

type RowState = {
  start: number;
  query: string;
  inside: ReadonlySet<string>;
  open: ReadonlySet<string>;
  onToggle: (key: string) => void;
};

/**
 * A finished run's steps as a timeline to check it by: the task it was handed, what it
 * thought and said, and every call it made in order, each with how it ended, how long it
 * took and when, opening onto exactly what it sent and got back.
 *
 * A long run can hold hundreds of steps, so the rows scroll in their own area and only
 * those near its viewport are drawn. A search finds rows by what they show or by what
 * they hold once opened, and the calls that failed or were turned away can be shown on
 * their own.
 */
export function RunTimeline({ blocks }: { blocks: readonly TranscriptBlock[] }) {
  const t = useT();
  const events = runEvents(blocks);
  const index = indexEvents(events);
  const start = runStart(events);
  const problems = countProblems(events);
  const keys = expandableKeys(events);
  const [open, setOpen] = useState<ReadonlySet<string>>(() => new Set());
  const [problemsOnly, setProblemsOnly] = useState(false);
  const [query, setQuery] = useState("");
  const deferredQuery = useDeferredValue(query);
  const searched = searchEvents(index, deferredQuery);
  const found = problemsOnly ? onlyProblems(searched) : searched;
  const steps = events.filter((event) => event.kind !== "gap").length;

  if (events.length === 0) {
    return <p className="ad-h">{t("The run kept no messages.")}</p>;
  }

  const allOpen = keys.length > 0 && keys.every((key) => open.has(key));
  const searching = deferredQuery.trim() !== "";
  const filtered = searching || problemsOnly;
  const toggle = (key: string) =>
    setOpen((current) => {
      const next = new Set(current);
      if (!next.delete(key)) {
        next.add(key);
      }
      return next;
    });

  return (
    <div className="rt">
      <div className="rt-bar">
        <Input
          type="search"
          value={query}
          placeholder={t("Search steps")}
          aria-label={t("Search steps")}
          inputContainerClassName="min-w-0 flex-1"
          leftElement={<SearchLgIcon className="text-muted-foreground size-3.5" />}
          onChange={(event) => setQuery(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === "Escape" && query !== "") {
              event.stopPropagation();
              setQuery("");
            }
          }}
        />
        {problems > 0 && (
          <button
            type="button"
            className={cn("rt-f", problemsOnly && "on")}
            aria-pressed={problemsOnly}
            onClick={() => setProblemsOnly((value) => !value)}
          >
            <Ic n="warn" s={12} />
            {t("{0, plural, one {# problem} other {# problems}}", problems)}
          </button>
        )}
        {keys.length > 0 && (
          <button
            type="button"
            className="xa"
            onClick={() => setOpen(allOpen ? new Set() : new Set(keys))}
          >
            <Ic n={allOpen ? "up" : "down"} s={12} />
            {allOpen ? t("Collapse all") : t("Expand all")}
          </button>
        )}
      </div>
      {filtered && (
        <p className="rt-count" aria-live="polite">
          {t("{0} of {1, plural, one {# step} other {# steps}}", found.matches, steps)}
        </p>
      )}
      {found.matches === 0 ? (
        <NothingFound query={searching ? deferredQuery.trim() : null} omitted={found.omitted} />
      ) : (
        <TimelineRows
          events={found.events}
          state={{
            start,
            query: deferredQuery,
            inside: found.inside,
            open,
            onToggle: toggle,
          }}
        />
      )}
    </div>
  );
}

/**
 * What a filter that found nothing says. A run that left messages out could have done it
 * there, where no search can look, so it says so rather than reading as "never happened".
 */
function NothingFound({ query, omitted }: { query: string | null; omitted: number }) {
  const t = useT();
  let text: string;
  if (query !== null) {
    text =
      omitted > 0
        ? t(
            "No step matches “{0}” in what the record kept. {1, plural, one {# message from the middle of the run was left out and could not be searched.} other {# messages from the middle of the run were left out and could not be searched.}}",
            query,
            omitted,
          )
        : t("No step matches “{0}”.", query);
  } else {
    text =
      omitted > 0
        ? t(
            "No step failed or was turned away in what the record kept. {0, plural, one {# message from the middle of the run was left out.} other {# messages from the middle of the run were left out.}}",
            omitted,
          )
        : t("No step failed or was turned away.");
  }

  return <p className="ad-h">{text}</p>;
}

/**
 * The rows in their own scroll area, windowed. Each is measured once drawn, so a row opened
 * onto its payload takes the room it needs and the rows below move down.
 */
function TimelineRows({ events, state }: { events: readonly RunEvent[]; state: RowState }) {
  const t = useT();

  return (
    <VirtualRows
      rows={events.map((event, index) => ({
        key: event.key,
        render: () => <EventItem event={event} state={state} edge={edgeOf(index, events.length)} />,
      }))}
      estimateSize={ROW_ESTIMATE}
      initialHeight={INITIAL_HEIGHT}
      role="list"
      aria-label={t("Steps")}
      mask="raised"
      className="rt-sc"
    />
  );
}

type Edge = "first" | "last" | "only" | undefined;

function edgeOf(index: number, count: number): Edge {
  if (count === 1) {
    return "only";
  }
  if (index === 0) {
    return "first";
  }
  return index === count - 1 ? "last" : undefined;
}

/** A delegate's work under the call that handed it over: short, so drawn whole. */
function NestedRows({ events, state }: { events: readonly RunEvent[]; state: RowState }) {
  return (
    <div role="list" className="rt-l nest">
      {events.map((event, index) => (
        <EventItem
          key={event.key}
          event={event}
          state={{ ...state, inside: new Set() }}
          edge={edgeOf(index, events.length)}
        />
      ))}
    </div>
  );
}

type ItemProps = {
  event: RunEvent;
  state: RowState;
  edge: Edge;
};

function Offset({ at, start }: { at: number; start: number }) {
  const t = useT();
  if (at <= 0 || start <= 0) {
    return null;
  }

  return <span className="rt-n">{t("+{0}", formatCountdownClock(at - start))}</span>;
}

function InsideMark() {
  const t = useT();
  return <span className="rt-c n">{t("Match inside")}</span>;
}

function EventItem({ event, state, edge }: ItemProps) {
  const t = useT();
  const { start, query, inside, open, onToggle } = state;
  const item = (marker: IcName, tone: Tone | null, children: ReactNode) => (
    <div role="listitem" data-edge={edge} className="rt-e">
      <span className={cn("rt-mk", tone)}>
        <Ic n={marker} s={12} />
      </span>
      {children}
    </div>
  );

  switch (event.kind) {
    case "gap":
      return item(
        "more",
        null,
        <p role="note" className="rt-gap">
          {t(
            "{0, plural, one {# message from the middle of the run was left out to keep the record bounded} other {# messages from the middle of the run were left out to keep the record bounded}}",
            event.count,
          )}
        </p>,
      );
    case "task":
      return item(
        "inbox",
        null,
        <>
          <div className="rt-r">
            <span className="rt-t">{t("Task")}</span>
            <span className="rt-s" />
            <Offset at={event.at} start={start} />
          </div>
          <p className="rt-task">
            <Marked text={event.text} query={query} />
          </p>
        </>,
      );
    case "said":
      return item(
        "chat",
        null,
        <>
          <div className="rt-r">
            <span className="rt-t">{t("Said")}</span>
            <span className="rt-s" />
            <Offset at={event.at} start={start} />
          </div>
          <div className="rt-said">
            <DeskProse content={event.text} />
          </div>
        </>,
      );
    case "thought": {
      const expanded = open.has(event.key);
      return item(
        "brain",
        null,
        <>
          <button
            type="button"
            className="rt-r"
            aria-expanded={expanded}
            onClick={() => onToggle(event.key)}
          >
            <span className="rt-t">{t("Thought")}</span>
            <span className="rt-s">
              {!expanded && <Marked text={rowLine(event).detail} query={query} />}
            </span>
            {!expanded && inside.has(event.key) && <InsideMark />}
            <Offset at={event.at} start={start} />
            <span className="rt-cv">
              <Ic n="down" s={12} />
            </span>
          </button>
          {expanded && (
            <p className="rt-x rt-thought">
              <Marked text={event.text} query={query} />
            </p>
          )}
        </>,
      );
    }
    default: {
      const { step } = event;
      const expanded = open.has(event.key);
      const line = rowLine(event);
      const chip = callChip(step, t);
      return item(
        EFFECT_ICON[step.effect],
        chip?.tone === "n" ? null : (chip?.tone ?? null),
        <>
          <button
            type="button"
            className="rt-r"
            aria-expanded={expanded}
            onClick={() => onToggle(event.key)}
          >
            <span className="rt-t">
              <Marked text={line.title} query={query} />
            </span>
            <span className="rt-s">
              <Marked text={line.detail} query={query} />
            </span>
            {!expanded && inside.has(event.key) && <InsideMark />}
            {chip && <span className={cn("rt-c", chip.tone)}>{chip.label}</span>}
            {step.durationSeconds !== null && (
              <span className="rt-n">{formatWorkDuration(step.durationSeconds)}</span>
            )}
            <Offset at={event.at} start={start} />
            <span className="rt-cv">
              <Ic n="down" s={12} />
            </span>
          </button>
          {expanded && <CallDetails event={event} state={state} />}
        </>,
      );
    }
  }
}

function CallDetails({
  event,
  state,
}: {
  event: Extract<RunEvent, { kind: "call" }>;
  state: RowState;
}) {
  const t = useT();
  const { step } = event;

  return (
    <div className="rt-x">
      <p className="rt-tool">
        <span>{t("Tool")}</span>
        <code>
          <Marked text={step.name} query={state.query} />
        </code>
      </p>
      <section>
        <h5>{t("Sent")}</h5>
        {step.arguments && Object.keys(step.arguments).length > 0 ? (
          <div className="rt-code">
            <JsonViewer data={step.arguments as never} collapsed={2} />
          </div>
        ) : (
          <p className="rt-q">{t("Nothing")}</p>
        )}
      </section>
      <section>
        <h5>{step.status === "proposed" ? t("Outcome") : t("Got back")}</h5>
        <CallResult step={step} clipped={event.clipped} query={state.query} />
      </section>
      {event.delegated.length > 0 && (
        <section>
          <h5>{t("What the agent it handed this to did")}</h5>
          <NestedRows events={event.delegated} state={state} />
        </section>
      )}
    </div>
  );
}

function CallResult({ step, clipped, query }: { step: ToolStep; clipped: boolean; query: string }) {
  const t = useT();

  if (step.content === "") {
    return (
      <p className="rt-q">
        {clipped
          ? t("The result was too large to keep; only the call was kept.")
          : t("Nothing was returned.")}
      </p>
    );
  }

  const result = parseToolResult(step.content);
  switch (result.kind) {
    case "error": {
      const refusal = toolRefusal(step);
      const tone = refusal && refusalTone(refusal) === "neutral" ? "rt-q" : refusal ? "t-w" : "t-d";
      return (
        <p className={cn("rt-err", tone)}>
          <Marked text={result.message} query={query} />
        </p>
      );
    }
    case "json":
      return (
        <div className="rt-code">
          <JsonViewer data={result.value as never} collapsed={2} />
        </div>
      );
    default:
      return (
        <div className="rt-code">
          <pre className="rt-pre">
            <Marked text={result.text} query={query} />
          </pre>
          {result.truncated && (
            <p className="rt-q">
              {t("The result was cut short for the model; the record itself is complete.")}
            </p>
          )}
        </div>
      );
  }
}
