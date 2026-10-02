import { useT } from "@trenova/shared/i18n/use-t";
import { JsonViewer } from "@/components/elements/json-viewer";
import { formatWorkDuration } from "@/lib/ai-usage-format";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { cn } from "@trenova/shared/lib/utils";
import type { ToolEffect } from "@/types/assistant";
import {
  BanIcon,
  BlocksIcon,
  BookOpenIcon,
  CheckIcon,
  ChevronRightIcon,
  CircleAlertIcon,
  CodeIcon,
  CompassIcon,
  FileTextIcon,
  ForwardIcon,
  GaugeIcon,
  GitCompareArrowsIcon,
  GlobeIcon,
  LockIcon,
  MessageCircleQuestionIcon,
  PenLineIcon,
  PresentationIcon,
  Repeat2Icon,
  ScrollTextIcon,
  SearchIcon,
  TableIcon,
  type LucideIcon,
} from "lucide-react";
import { useMemo, useState } from "react";
import {
  delegateRunning,
  describeActivity,
  groupActivity,
  isActionEffect,
  refusalLabel,
  refusalTone,
  toolRefusal,
  workSummary,
  type ActivityGroup,
  type ActivityLine,
  type ToolRefusal,
  type ToolStep,
} from "./activity";
import {
  describeToolCall,
  parseToolResult,
  readableEntries,
  readableResult,
  type ParsedToolResult,
  type ReadableEntry,
  type ReadableValue,
  WEB_READ_TOOL,
  WEB_SEARCH_TOOL,
} from "./tool-presentation";
import { DelegateStep } from "./delegate-step";
import { DisplayValue } from "./display-value";
import { WorkingDot } from "./voice/working-dot";
import { WebSourceList } from "./web-citations";
import { sourcesOfStep } from "./web-sources";

export type { ToolActivityStatus, ToolStep } from "./activity";

const EFFECT_ICONS: Record<ToolEffect, LucideIcon> = {
  lookup: SearchIcon,
  discover: BlocksIcon,
  navigate: CompassIcon,
  present: PresentationIcon,
  change: PenLineIcon,
  ask: MessageCircleQuestionIcon,
  delegate: ForwardIcon,
};

/** A few tools say more about themselves than their effect does. */
const NAMED_ICONS: Readonly<Record<string, LucideIcon>> = {
  find_in_trenova: BookOpenIcon,
  run_report: FileTextIcon,
  publish_artifact: ScrollTextIcon,
  compose_table_view: TableIcon,
  compare_report_runs: GitCompareArrowsIcon,
  [WEB_SEARCH_TOOL]: GlobeIcon,
  [WEB_READ_TOOL]: GlobeIcon,
};

const REFUSAL_ICONS: Record<ToolRefusal, LucideIcon> = {
  denied: LockIcon,
  invalid: BanIcon,
  over_budget: GaugeIcon,
  duplicate: Repeat2Icon,
};

const REFUSAL_TEXT: Record<ReturnType<typeof refusalTone>, string> = {
  warning: "text-warning",
  neutral: "text-foreground-muted",
};

function refusalText(refusal: ToolRefusal): string {
  return REFUSAL_TEXT[refusalTone(refusal)];
}

function iconFor(group: ActivityGroup): LucideIcon {
  const first = group.steps[0];

  return NAMED_ICONS[first.name] ?? EFFECT_ICONS[group.effect];
}

/** The whole group's time, when every step in it was timed. */
function groupDuration(group: ActivityGroup): number | null {
  let total = 0;
  for (const step of group.steps) {
    if (step.durationSeconds === null) {
      return null;
    }
    total += step.durationSeconds;
  }

  return total;
}

/**
 * What the agent did on its way to an answer, a line per piece of work.
 *
 * Reads fold into one line, because four lookups are one piece of looking.
 * Actions never do: opening a page, running a report or saving a change each
 * stand on a line of their own, in the ink colour, with their own mark in a
 * small well, so an action is never mistaken for a lookup. Each line says
 * what happened in the words of what happened, and opens onto the call
 * itself — what was asked, what came back, and the raw payload behind a
 * second click.
 *
 * While a reply is being written the rows land one by one. Once it is saved
 * a `folded` list is one quiet line — "Worked through 6 steps · 12s" — that
 * opens onto the rows, so the answer is what the eye lands on and the work
 * is there for whoever wants to check it.
 */
export function ToolActivity({
  steps,
  live = false,
  folded = false,
}: {
  steps: ToolStep[];
  live?: boolean;
  /** Draws a settled list as one summary line that opens onto the rows. */
  folded?: boolean;
}) {
  const t = useT();
  const [open, setOpen] = useState(false);
  // While a reply is being written the step under way is told by the working
  // line beneath it; a step joins this list when it lands, with its check.
  // A hand-off is the exception: it opens as soon as the task is handed
  // over, because the other agent's own work is drawn inside it as it goes.
  const groups = useMemo(
    () =>
      groupActivity(
        live
          ? steps.filter((step) => step.status !== "running" || step.effect === "delegate")
          : steps,
      ),
    [live, steps],
  );
  if (groups.length === 0) {
    return null;
  }

  const rows = (
    <ol className="flex min-w-0 flex-col gap-0.5">
      {groups.map((group) =>
        group.effect === "delegate" ? (
          <DelegateStep
            key={group.key}
            step={group.steps[0]}
            live={live}
            running={live && delegateRunning(group.steps[0])}
          />
        ) : (
          <ActivityRow key={group.key} group={group} live={live} />
        ),
      )}
    </ol>
  );

  if (!folded) {
    return rows;
  }

  const failed = steps.some((step) => step.status === "failed");

  // Live, the rows stand open with no line above them; the moment the reply
  // settles they fold up under the summary, so the answer is what remains.
  return (
    <Collapsible
      open={live || open}
      onOpenChange={setOpen}
      data-slot="work-log"
      data-state={live ? "live" : open ? "open" : "closed"}
      className="min-w-0"
    >
      {!live && (
        <CollapsibleTrigger
          className={cn(
            "group/work ui-focus-ring rounded-control -mx-1.5 flex max-w-[calc(100%+0.75rem)] min-w-0 items-center gap-1.5 px-1.5 py-0.5 text-left text-xs transition-colors",
            "text-foreground-muted hover:bg-surface-hover hover:text-foreground",
          )}
        >
          <ChevronRightIcon
            aria-hidden
            className={cn("size-3 shrink-0 transition-transform duration-200", open && "rotate-90")}
          />
          <WorkMarks groups={groups} />
          <span className="min-w-0 truncate">{workSummary(steps, t)}</span>
          {failed && <CircleAlertIcon aria-hidden className="text-danger size-3 shrink-0" />}
        </CollapsibleTrigger>
      )}
      <CollapsibleContent className="ease-settle h-(--collapsible-panel-height) overflow-hidden transition-[height] duration-200 data-ending-style:h-0 data-starting-style:h-0">
        <div className={cn(!live && "border-border-subtle mt-1 ml-1.25 border-l pl-3")}>{rows}</div>
      </CollapsibleContent>
    </Collapsible>
  );
}

/** The marks of the first few pieces of work, overlapped, so a folded line still says what kind of work it was. */
const WORK_MARK_LIMIT = 3;

function WorkMarks({ groups }: { groups: readonly ActivityGroup[] }) {
  return (
    <span aria-hidden className="flex shrink-0 items-center -space-x-1">
      {groups.slice(0, WORK_MARK_LIMIT).map((group) => (
        <WorkMark key={group.key} group={group} />
      ))}
    </span>
  );
}

function WorkMark({ group }: { group: ActivityGroup }) {
  const Icon = iconFor(group);

  return (
    <span className="bg-sunken ring-card flex size-4 items-center justify-center rounded-full ring-1">
      <Icon className="text-foreground-muted size-2.5" />
    </span>
  );
}

function ActivityRow({ group, live }: { group: ActivityGroup; live: boolean }) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const line = describeActivity(group, t);
  const running = line.state === "running";
  const action = isActionEffect(group.effect);
  // A folded group that grows while it is watched says so: the new count
  // rises into place rather than swapping silently.
  const [firstPhrase] = useState(line.phrase);
  const grew = live && line.phrase !== firstPhrase;
  const duration = groupDuration(group);

  return (
    <li className={cn("min-w-0", live && "animate-land")}>
      <Collapsible open={open} onOpenChange={setOpen}>
        <CollapsibleTrigger
          disabled={running}
          className={cn(
            "group/activity ui-focus-ring -mx-1.5 flex w-[calc(100%+0.75rem)] min-w-0 items-center gap-2 rounded-control px-1.5 py-1 text-left text-xs transition-colors",
            "hover:bg-surface-hover disabled:cursor-default disabled:hover:bg-transparent",
          )}
        >
          <ActivityMark group={group} line={line} action={action} live={live} />
          <span className="flex min-w-0 flex-1 items-baseline gap-1.5">
            <span
              key={line.phrase}
              className={cn(
                "min-w-0 shrink truncate",
                line.refusal
                  ? refusalText(line.refusal)
                  : line.state === "failed"
                    ? "text-danger"
                    : action
                      ? "text-foreground"
                      : "text-foreground-muted",
                grew && "animate-rise",
              )}
            >
              {line.phrase}
            </span>
            {line.detail !== "" && (
              <span className="text-foreground-subtle min-w-0 flex-1 truncate">{line.detail}</span>
            )}
            {line.failure !== "" && <span className="text-danger shrink-0">{line.failure}</span>}
          </span>
          {duration !== null && !running && (
            <span className="text-foreground-subtle shrink-0 font-mono text-2xs tabular-nums">
              {formatWorkDuration(duration)}
            </span>
          )}
          {!running && (
            <ChevronRightIcon
              aria-hidden
              className={cn(
                "text-foreground-subtle size-3 shrink-0 opacity-0 transition-[opacity,rotate] group-hover/activity:opacity-100 group-focus-visible/activity:opacity-100",
                open && "rotate-90 opacity-100",
              )}
            />
          )}
        </CollapsibleTrigger>
        <CollapsibleContent className="h-(--collapsible-panel-height) overflow-hidden transition-[height] duration-200 ease-settle data-ending-style:h-0 data-starting-style:h-0">
          <div className="pt-1 pb-2 pl-6">
            {group.steps.length === 1 ? (
              <StepDetails step={group.steps[0]} />
            ) : (
              <ol className="flex flex-col gap-0.5">
                {group.steps.map((step) => (
                  <StepRow key={step.id} step={step} />
                ))}
              </ol>
            )}
          </div>
        </CollapsibleContent>
      </Collapsible>
    </li>
  );
}

/**
 * The line's mark: the kind of work it was, and an action's mark sits in a
 * small sunken well, which is what sets it apart from a lookup without
 * spending a colour on it. In a reply still being written, a step lands as a
 * check with the confirm spring, so finishing is felt as well as read.
 */
function ActivityMark({
  group,
  line,
  action,
  live,
}: {
  group: ActivityGroup;
  line: ActivityLine;
  action: boolean;
  live: boolean;
}) {
  const Icon = iconFor(group);
  const Refused = line.refusal ? REFUSAL_ICONS[line.refusal] : null;

  return (
    <span
      aria-hidden
      className={cn(
        "flex size-4.5 shrink-0 items-center justify-center rounded-sm",
        action && line.state !== "running" && "bg-sunken",
      )}
    >
      {line.state === "running" ? (
        <WorkingDot working still />
      ) : Refused && line.refusal ? (
        <Refused className={cn("size-3", refusalText(line.refusal), live && "animate-confirm")} />
      ) : line.state === "failed" ? (
        <CircleAlertIcon className={cn("text-danger size-3", live && "animate-confirm")} />
      ) : live ? (
        <CheckIcon key={line.phrase} className="text-foreground-muted animate-confirm size-3" />
      ) : (
        <Icon
          className={cn("size-3", action ? "text-foreground-muted" : "text-foreground-subtle")}
        />
      )}
    </span>
  );
}

/** One call inside a folded group, opening onto its own details. */
function StepRow({ step }: { step: ToolStep }) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const description = describeToolCall(step.name, step.arguments);
  const failed = step.status === "failed";
  const running = step.status === "running";
  const refusal = toolRefusal(step);
  const Refused = refusal ? REFUSAL_ICONS[refusal] : null;

  return (
    <li className="min-w-0">
      <Collapsible open={open} onOpenChange={setOpen}>
        <CollapsibleTrigger
          disabled={running}
          aria-label={t("Show details for {0}", description.title)}
          className="group/step ui-focus-ring hover:bg-surface-hover -mx-1.5 flex w-[calc(100%+0.75rem)] min-w-0 items-center gap-2 rounded-control px-1.5 py-0.5 text-left text-xs transition-colors disabled:cursor-default"
        >
          {running ? (
            <WorkingDot working still className="mx-0.75" />
          ) : Refused && refusal ? (
            <Refused aria-hidden className={cn("size-3 shrink-0", refusalText(refusal))} />
          ) : failed ? (
            <CircleAlertIcon aria-hidden className="text-danger size-3 shrink-0" />
          ) : (
            <span aria-hidden className="bg-border-strong mx-1.25 size-1 shrink-0 rounded-full" />
          )}
          <span className="flex min-w-0 flex-1 items-baseline gap-1.5">
            <span
              className={cn(
                "shrink-0",
                refusal ? refusalText(refusal) : failed ? "text-danger" : "text-foreground-muted",
              )}
            >
              {description.title}
            </span>
            {refusal && (
              <span className={cn("shrink-0", refusalText(refusal))}>
                {refusalLabel(refusal, t)}
              </span>
            )}
            {description.subject !== "" && (
              <span className="text-foreground-subtle min-w-0 truncate">{description.subject}</span>
            )}
          </span>
          {step.durationSeconds !== null && !running && (
            <span className="text-foreground-subtle shrink-0 font-mono text-2xs tabular-nums">
              {formatWorkDuration(step.durationSeconds)}
            </span>
          )}
          {!running && (
            <ChevronRightIcon
              aria-hidden
              className={cn(
                "text-foreground-subtle size-3 shrink-0 opacity-0 transition-[opacity,rotate] group-hover/step:opacity-100",
                open && "rotate-90 opacity-100",
              )}
            />
          )}
        </CollapsibleTrigger>
        <CollapsibleContent className="h-(--collapsible-panel-height) overflow-hidden transition-[height] duration-200 ease-settle data-ending-style:h-0 data-starting-style:h-0">
          <div className="pt-1 pb-2 pl-5">
            <StepDetails step={step} />
          </div>
        </CollapsibleContent>
      </Collapsible>
    </li>
  );
}

/**
 * What one call was asked and what it gave back, as labelled values. The
 * literal payload is behind Details for whoever needs to check the machine's
 * exact words; nobody should have to read JSON to learn what was looked up.
 */
function StepDetails({ step }: { step: ToolStep }) {
  const t = useT();
  const [raw, setRaw] = useState(false);
  const asked = useMemo(() => readableEntries(step.arguments ?? null), [step.arguments]);
  const result = useMemo(
    () => (step.status === "running" || step.content === "" ? null : parseToolResult(step.content)),
    [step.status, step.content],
  );
  const pages = useMemo(() => sourcesOfStep(step), [step]);

  return (
    <div className="flex min-w-0 flex-col gap-2.5 text-xs">
      {asked.entries.length > 0 && (
        <section className="flex min-w-0 flex-col gap-1">
          <h4 className="text-foreground-subtle font-medium">{t("Asked for")}</h4>
          <ReadableList entries={asked.entries} hidden={asked.hidden} />
        </section>
      )}

      <section className="flex min-w-0 flex-col gap-1">
        <h4 className="text-foreground-subtle font-medium">
          {step.status === "proposed" ? t("Outcome") : t("Got back")}
        </h4>
        {pages.length > 0 ? (
          <WebSourceList sources={pages} />
        ) : (
          <ResultBody status={step.status} result={result} refusal={toolRefusal(step)} />
        )}
      </section>

      <div className="flex flex-col gap-1.5">
        <Button
          size="xs"
          variant="ghost"
          className="text-foreground-subtle hover:text-foreground -ml-1.5 h-5 w-fit px-1.5 text-2xs"
          aria-expanded={raw}
          onClick={() => setRaw((value) => !value)}
        >
          <CodeIcon className="size-3" />
          {raw ? t("Hide details") : t("Details")}
        </Button>
        {raw && (
          <div className="bg-sunken scrollbar-overlay animate-rise max-h-72 overflow-auto rounded-md p-2">
            <JsonViewer
              data={
                {
                  tool: step.name,
                  arguments: step.arguments ?? {},
                  result:
                    result === null
                      ? null
                      : result.kind === "json"
                        ? result.value
                        : result.kind === "error"
                          ? result.message
                          : result.text,
                } as never
              }
              collapsed={2}
            />
          </div>
        )}
      </div>
    </div>
  );
}

function ReadableList({ entries, hidden }: { entries: ReadableEntry[]; hidden: number }) {
  const t = useT();

  return (
    <>
      <DescriptionList
        layout="inline"
        className="grid-cols-[minmax(4.5rem,auto)_minmax(0,1fr)] gap-x-3 gap-y-1 text-xs"
      >
        {entries.map((entry) => (
          <DescriptionItem key={entry.key} label={entry.label} valueClassName="text-xs">
            <ReadableValueText label={entry.label} value={entry.value} />
          </DescriptionItem>
        ))}
      </DescriptionList>
      {hidden > 0 && <p className="text-foreground-subtle">{t("{0} more in Details", hidden)}</p>}
    </>
  );
}

function ReadableValueText({ label, value }: { label: string; value: ReadableValue }) {
  const t = useT();

  switch (value.kind) {
    case "value":
      return <DisplayValue type={value.type} value={value.value} label={label} />;
    case "items":
      return (
        <span className="text-foreground-muted">
          {t("{0, plural, one {# item} other {# items}}", value.count)}
        </span>
      );
    case "fields":
      return (
        <span className="text-foreground-muted">
          {t("{0, plural, one {# field} other {# fields}}", value.count)}
        </span>
      );
    default:
      return <span className="line-clamp-3 break-words">{value.text}</span>;
  }
}

function ResultBody({
  status,
  result,
  refusal,
}: {
  status: ToolStep["status"];
  result: ParsedToolResult | null;
  refusal: ToolRefusal | null;
}) {
  const t = useT();

  if (status === "running") {
    return <p className="text-foreground-muted">{t("Waiting for the result…")}</p>;
  }
  if (result === null) {
    return <p className="text-foreground-muted">{t("Nothing was returned.")}</p>;
  }

  switch (result.kind) {
    case "error": {
      const Mark = refusal ? REFUSAL_ICONS[refusal] : CircleAlertIcon;
      return (
        <p
          className={cn("flex items-start gap-1.5", refusal ? refusalText(refusal) : "text-danger")}
        >
          <Mark aria-hidden className="mt-0.5 size-3 shrink-0" />
          <span className="min-w-0 break-words">{result.message}</span>
        </p>
      );
    }
    case "json":
      return <ReadableResultBody value={result.value} />;
    default:
      return (
        <div className="flex flex-col gap-1">
          <p className="text-foreground-muted line-clamp-6 break-words whitespace-pre-wrap">
            {result.text}
          </p>
          {result.truncated && (
            <p className="text-foreground-subtle">
              {t("The result was cut short for the model; the record itself is complete.")}
            </p>
          )}
        </div>
      );
  }
}

function ReadableResultBody({ value }: { value: unknown }) {
  const t = useT();
  const shaped = useMemo(() => readableResult(value), [value]);

  switch (shaped.kind) {
    case "list":
      return (
        <div className="flex min-w-0 flex-col gap-1">
          <p className="text-foreground-muted">
            {shaped.count === 0
              ? t("No records")
              : shaped.more
                ? t("{0, plural, one {#+ record} other {#+ records}}", shaped.count)
                : t("{0, plural, one {# record} other {# records}}", shaped.count)}
          </p>
          {shaped.labels.length > 0 && (
            <ul className="flex flex-wrap gap-1">
              {shaped.labels.map((label, index) => (
                <li
                  key={`${label}-${index}`}
                  className="bg-sunken text-foreground max-w-full truncate rounded-full px-2 py-px"
                >
                  {label}
                </li>
              ))}
            </ul>
          )}
        </div>
      );
    case "record":
      return shaped.entries.length === 0 ? (
        <p className="text-foreground-muted">
          {t("A structured record; open Details to read it.")}
        </p>
      ) : (
        <ReadableList entries={shaped.entries} hidden={shaped.hidden} />
      );
    default:
      return <p className="break-words">{shaped.text}</p>;
  }
}
