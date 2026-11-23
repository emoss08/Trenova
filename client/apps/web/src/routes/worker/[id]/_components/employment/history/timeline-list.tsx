import { HoverGlideGroup } from "@/components/hover-glide";
import type { WorkerEmploymentEventRow } from "@/lib/graphql/worker-employment";
import {
  ArrowRightIcon,
  Edit03Icon,
  PaperclipIcon,
  Pencil01Icon,
} from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { translate } from "@trenova/shared/i18n/runtime";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import {
  EMPLOYMENT_VALUE_HIDDEN_KEYS,
  EMPLOYMENT_VALUE_LABELS,
} from "@trenova/shared/types/worker-employment";
import { useEffect, useRef } from "react";
import {
  EMPLOYMENT_EVENT_TONE_CLASS,
  employmentEventMeta,
  employmentEventName,
} from "./employment-event-meta";

export type ValuePair = { key: string; label: string; from?: string; to?: string };

/** The enum values an event carries, in words; free values (a fleet code, a rate) read as typed. */
const EMPLOYMENT_VALUE_WORDS: Record<string, string> = defineLabels({
  Active: "Active",
  Inactive: "Inactive",
  Onboarding: "Onboarding",
  OnLeave: "On leave",
  Suspended: "Suspended",
  Terminated: "Terminated",
  Local: "Local",
  Regional: "Regional",
  OTR: "OTR",
  Team: "Team",
  Employee: "Employee",
  Contractor: "Contractor",
  FMLA: "FMLA",
  Medical: "Medical",
  Military: "Military",
  Parental: "Parental",
  Personal: "Personal",
  Other: "Other",
});

const ENUM_VALUE_KEYS: ReadonlySet<string> = new Set([
  "status",
  "driverType",
  "workerType",
  "leaveType",
]);

function formatValue(key: string, value: string): string {
  if (key === "hireDate" || key === "terminationDate") {
    const unix = Number(value);
    return Number.isFinite(unix) && unix > 0 ? formatUnixDateMedium(unix) : value;
  }
  if (key === "canBeAssigned") return value === "true" ? translate("Yes") : translate("No");
  if (ENUM_VALUE_KEYS.has(key)) return EMPLOYMENT_VALUE_WORDS[value] ?? value;
  return value;
}

/**
 * Pairs from/to values by key. A raw id is hidden when its labelled twin is present, and a key
 * the client has no name for is left out rather than shown by its code name.
 */
export function pairValues(
  event: Pick<WorkerEmploymentEventRow, "fromValues" | "toValues">,
): ValuePair[] {
  const keys = new Set<string>();
  const from = new Map(event.fromValues.map((entry) => [entry.key, entry.value]));
  const to = new Map(event.toValues.map((entry) => [entry.key, entry.value]));
  for (const key of [...from.keys(), ...to.keys()]) {
    if (EMPLOYMENT_VALUE_HIDDEN_KEYS.has(key)) continue;
    if (!(key in EMPLOYMENT_VALUE_LABELS)) continue;
    keys.add(key);
  }
  const pairs: ValuePair[] = [];
  for (const key of keys) {
    const fromValue = from.get(key);
    const toValue = to.get(key);
    if ((fromValue == null || fromValue === "") && (toValue == null || toValue === "")) continue;
    pairs.push({
      key,
      label: EMPLOYMENT_VALUE_LABELS[key],
      from: fromValue != null && fromValue !== "" ? formatValue(key, fromValue) : undefined,
      to: toValue != null && toValue !== "" ? formatValue(key, toValue) : undefined,
    });
  }
  return pairs;
}

export type YearGroup = { year: number; events: WorkerEmploymentEventRow[] };

export function groupByYear(events: readonly WorkerEmploymentEventRow[]): YearGroup[] {
  const sorted = [...events].sort(
    (a, b) => b.effectiveAt - a.effectiveAt || b.createdAt - a.createdAt,
  );
  const groups: YearGroup[] = [];
  for (const event of sorted) {
    const year = new Date(event.effectiveAt * 1000).getUTCFullYear();
    const last = groups[groups.length - 1];
    if (last && last.year === year) {
      last.events.push(event);
    } else {
      groups.push({ year, events: [event] });
    }
  }
  return groups;
}

/** A from → to change as a chip: the label, the old value struck through, the new one. */
export function ValueChips({ pairs }: { pairs: readonly ValuePair[] }) {
  if (pairs.length === 0) return null;
  return (
    <ul className="flex flex-wrap gap-1">
      {pairs.map((pair) => (
        <li
          key={pair.key}
          className="flex items-center gap-1 rounded-sm bg-sunken px-1.5 text-xs leading-5"
        >
          <span className="text-muted-foreground">{pair.label}</span>
          {pair.from ? <del className="text-muted-foreground">{pair.from}</del> : null}
          {pair.from && pair.to ? (
            <ArrowRightIcon aria-hidden className="size-3 text-muted-foreground" />
          ) : null}
          {pair.to ? <ins className="no-underline">{pair.to}</ins> : null}
        </li>
      ))}
    </ul>
  );
}

type TimelineListProps = {
  events: readonly WorkerEmploymentEventRow[];
  canAmend: boolean;
  onAmend: (event: WorkerEmploymentEventRow) => void;
  /** An event a link sent the reader to: scrolled into view and marked for a moment. */
  highlightId?: string | null;
};

/**
 * The employment history, newest first: the year in a sticky gutter, each event on one rail
 * with its kind's mark, its changes as chips, its reason and who recorded it.
 */
export function TimelineList({ events, canAmend, onAmend, highlightId }: TimelineListProps) {
  const groups = groupByYear(events);
  return (
    <HoverGlideGroup className="px-1.5 pt-1 pb-2">
      {groups.map((group) => (
        <section
          key={group.year}
          aria-label={String(group.year)}
          className="grid grid-cols-[3.25rem_minmax(0,1fr)]"
        >
          <div className="relative z-10">
            <span
              data-testid={`timeline-year-${group.year}`}
              className="sticky top-0 block bg-card pt-3 pl-2 font-mono text-xs text-foreground-subtle tabular-nums"
            >
              {group.year}
            </span>
          </div>
          <ol className="relative flex flex-col gap-px">
            <span aria-hidden className="absolute top-0 bottom-0 left-4 w-px bg-border" />
            {group.events.map((event) => (
              <TimelineItem
                key={event.id}
                event={event}
                canAmend={canAmend}
                onAmend={onAmend}
                highlighted={highlightId === event.id}
              />
            ))}
          </ol>
        </section>
      ))}
    </HoverGlideGroup>
  );
}

function TimelineItem({
  event,
  canAmend,
  onAmend,
  highlighted,
}: {
  event: WorkerEmploymentEventRow;
  canAmend: boolean;
  onAmend: (event: WorkerEmploymentEventRow) => void;
  highlighted: boolean;
}) {
  const t = useT();
  const ref = useRef<HTMLLIElement>(null);
  const meta = employmentEventMeta(event.kind);
  const Icon = meta.icon;
  const name = employmentEventName(event.kind);
  const pairs = pairValues(event);
  const when = formatUnixDateMedium(event.effectiveAt);

  useEffect(() => {
    if (highlighted) ref.current?.scrollIntoView({ block: "nearest" });
  }, [highlighted]);

  return (
    <li
      ref={ref}
      data-glide-target
      data-testid={`timeline-event-${event.id}`}
      data-kind={event.kind}
      className={cn(
        "group relative grid grid-cols-[1rem_minmax(0,1fr)] gap-3 rounded-md px-2 py-2.5",
        highlighted && "animate-attention-a",
      )}
    >
      <span
        aria-hidden
        className={cn(
          "relative flex h-4.5 w-4 items-center justify-center bg-card",
          EMPLOYMENT_EVENT_TONE_CLASS[meta.tone],
        )}
      >
        <Icon className="size-3.5" />
      </span>
      <div className="flex min-w-0 flex-col gap-1.5">
        <div className="flex min-h-4.5 min-w-0 items-center gap-2">
          <span className="truncate text-sm font-medium">{name}</span>
          {event.amendedAt ? (
            <Badge variant="warning" dot>
              {t("Amended")}
            </Badge>
          ) : null}
          <span className="ml-auto font-mono text-xs whitespace-nowrap text-muted-foreground tabular-nums">
            {when}
          </span>
          {canAmend ? (
            <Button
              variant="ghost"
              size="icon-xs"
              className="-my-1 opacity-0 transition-opacity group-hover:opacity-100 focus-visible:opacity-100"
              aria-label={t("Amend {0}, {1}", name, when)}
              title={t("Amend")}
              onClick={() => onAmend(event)}
            >
              <Pencil01Icon className="size-3.5" />
            </Button>
          ) : null}
        </div>
        <ValueChips pairs={pairs} />
        {event.reason ? <p className="text-sm">{event.reason}</p> : null}
        {event.amendedAt ? (
          <p className="flex items-start gap-1.5 text-xs text-warning-foreground">
            <Edit03Icon aria-hidden className="mt-0.5 size-3 shrink-0" />
            <span>{amendedLine(event)}</span>
          </p>
        ) : null}
        {event.notes ? (
          <p className="text-xs whitespace-pre-wrap text-muted-foreground">{event.notes}</p>
        ) : null}
        <p className="flex flex-wrap items-center gap-x-2 text-xs text-foreground-subtle">
          <span>
            {event.recordedBy?.name
              ? t("Recorded by {0}", event.recordedBy.name)
              : t("Recorded automatically")}
          </span>
          {event.document ? (
            <span className="flex items-center gap-1" title={event.document.originalName}>
              <PaperclipIcon aria-hidden className="size-3" />
              {event.document.originalName}
            </span>
          ) : null}
        </p>
      </div>
    </li>
  );
}

function amendedLine(event: WorkerEmploymentEventRow): string {
  const who = event.amendedBy?.name;
  const on = event.amendedAt ? formatUnixDateMedium(event.amendedAt) : null;
  const why = event.amendmentNote;
  if (who && on && why) return translate("Amended by {0} on {1}: {2}", who, on, why);
  if (who && on) return translate("Amended by {0} on {1}", who, on);
  if (on && why) return translate("Amended on {0}: {1}", on, why);
  if (on) return translate("Amended on {0}", on);
  return translate("Amended");
}
