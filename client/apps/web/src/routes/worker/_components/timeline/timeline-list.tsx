import { useT } from "@trenova/shared/i18n/use-t";
import type { WorkerEmploymentEventRow } from "@/lib/graphql/worker-employment";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import {
  EMPLOYMENT_VALUE_HIDDEN_KEYS,
  EMPLOYMENT_VALUE_LABELS,
} from "@trenova/shared/types/worker-employment";
import { ArrowRightIcon, PaperclipIcon, PencilLineIcon } from "lucide-react";
import { useMemo } from "react";
import { employmentEventMeta } from "./employment-event-meta";

type ValuePair = { key: string; label: string; from?: string; to?: string };

function formatValue(key: string, value: string): string {
  if (key === "hireDate" || key === "terminationDate") {
    const unix = Number(value);
    return Number.isFinite(unix) && unix > 0 ? formatUnixDateMedium(unix) : value;
  }
  if (key === "canBeAssigned") return value === "true" ? "Yes" : "No";
  return value;
}

/** Pairs from/to values by key, hiding raw ids when a labelled twin is present. */
export function pairValues(event: WorkerEmploymentEventRow): ValuePair[] {
  const keys = new Set<string>();
  const from = new Map(event.fromValues.map((entry) => [entry.key, entry.value]));
  const to = new Map(event.toValues.map((entry) => [entry.key, entry.value]));
  for (const key of [...from.keys(), ...to.keys()]) {
    if (EMPLOYMENT_VALUE_HIDDEN_KEYS.has(key)) {
      const twin = key.replace(/Id$/, "");
      if (from.has(twin) || to.has(twin)) continue;
    }
    keys.add(key);
  }
  const pairs: ValuePair[] = [];
  for (const key of keys) {
    const fromValue = from.get(key);
    const toValue = to.get(key);
    if (fromValue == null && toValue == null) continue;
    pairs.push({
      key,
      label: EMPLOYMENT_VALUE_LABELS[key] ?? key,
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

type TimelineListProps = {
  events: readonly WorkerEmploymentEventRow[];
  canAmend: boolean;
  onAmend: (event: WorkerEmploymentEventRow) => void;
};

export function TimelineList({ events, canAmend, onAmend }: TimelineListProps) {
  const groups = useMemo(() => groupByYear(events), [events]);

  return (
    <div className="flex flex-col gap-6">
      {groups.map((group) => (
        <section key={group.year} className="flex flex-col gap-3">
          <h4
            data-testid={`timeline-year-${group.year}`}
            className="text-muted-foreground sticky top-0 z-10 bg-background/95 py-1 text-[11px] font-semibold tracking-wide uppercase backdrop-blur"
          >
            {group.year}
          </h4>
          <ol className="border-border relative ml-4 flex flex-col gap-4 border-l pl-6">
            {group.events.map((event) => (
              <TimelineItem key={event.id} event={event} canAmend={canAmend} onAmend={onAmend} />
            ))}
          </ol>
        </section>
      ))}
    </div>
  );
}

function TimelineItem({
  event,
  canAmend,
  onAmend,
}: {
  event: WorkerEmploymentEventRow;
  canAmend: boolean;
  onAmend: (event: WorkerEmploymentEventRow) => void;
}) {
  const t = useT();

  const meta = employmentEventMeta(event.kind);
  const Icon = meta.icon;
  const pairs = useMemo(() => pairValues(event), [event]);
  const recordedBy = event.recordedBy?.name ?? "System";

  return (
    <li
      data-testid={`timeline-event-${event.id}`}
      data-kind={event.kind}
      className="group relative flex flex-col gap-1.5"
    >
      <span
        aria-hidden
        className={cn(
          "bg-background absolute -left-[37px] top-0.5 flex size-6 items-center justify-center rounded-full ring-1",
          meta.toneClass,
        )}
      >
        <Icon className="size-3.5" />
      </span>

      <div className="flex flex-wrap items-center gap-2">
        <span className="text-sm font-semibold">{t(meta.label)}</span>
        <span className="text-muted-foreground text-xs">
          {formatUnixDateMedium(event.effectiveAt)}
        </span>
        {event.amendedAt ? (
          <Badge variant="warning" className="px-1.5 py-0 text-[10px]">
            {t("Amended")}
          </Badge>
        ) : null}
        {canAmend ? (
          <Button
            variant="ghost"
            size="icon"
            className="ml-auto size-7 opacity-0 transition-opacity group-hover:opacity-100 focus-visible:opacity-100"
            aria-label={`Amend ${meta.label}`}
            title={t("Amend")}
            onClick={() => onAmend(event)}
          >
            <PencilLineIcon className="size-3.5" />
          </Button>
        ) : null}
      </div>

      {event.reason ? <p className="text-sm">{event.reason}</p> : null}

      {pairs.length > 0 ? (
        <ul className="flex flex-wrap gap-1.5">
          {pairs.map((pair) => (
            <li
              key={pair.key}
              className="bg-muted/50 border-border flex items-center gap-1 rounded-md border px-2 py-0.5 text-[11px]"
            >
              <span className="text-muted-foreground">{t(pair.label)}:</span>
              {pair.from ? <span className="line-through opacity-70">{pair.from}</span> : null}
              {pair.from && pair.to ? (
                <ArrowRightIcon aria-hidden className="text-muted-foreground size-3" />
              ) : null}
              {pair.to ? <span className="font-medium">{pair.to}</span> : null}
            </li>
          ))}
        </ul>
      ) : null}

      {event.notes ? (
        <p className="text-muted-foreground text-xs whitespace-pre-wrap">{event.notes}</p>
      ) : null}

      <p className="text-muted-foreground flex flex-wrap items-center gap-x-2 text-[11px]">
        <span>{t("Recorded by {0}", recordedBy)}</span>
        {event.document ? (
          <span className="flex items-center gap-1" title={event.document.originalName}>
            <PaperclipIcon className="size-3" />
            {event.document.originalName}
          </span>
        ) : null}
        {event.amendedAt ? (
          <span>
            {t("· Amended by {0}{1}", event.amendedBy?.name ?? "someone", event.amendmentNote ? ` — ${event.amendmentNote}` : "")}
          </span>
        ) : null}
      </p>
    </li>
  );
}
