import { VirtualRows, type VirtualRow, type VirtualRowsHandle } from "@/components/virtual-rows";
import { preloadOn } from "@/components/work-pane";
import type { WorkerEmploymentEventRow } from "@/lib/graphql/worker-employment";
import { Edit03Icon, PaperclipIcon, Pencil01Icon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { translate } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium, toUserWallClock } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type { Ref } from "react";
import { ListMoreLine, type ListMore } from "../../list-more";
import {
  EMPLOYMENT_EVENT_TONE_CLASS,
  employmentEventMeta,
  employmentEventName,
} from "./employment-event-meta";
import { pairValues, ValueChips } from "./employment-values";

export type YearGroup = { year: number; events: WorkerEmploymentEventRow[] };

/** The year an event falls in, read in the reader's zone, as its date is shown. */
function yearOf(event: WorkerEmploymentEventRow): number {
  return toUserWallClock(event.effectiveAt)?.getFullYear() ?? 0;
}

export function groupByYear(events: readonly WorkerEmploymentEventRow[]): YearGroup[] {
  const sorted = [...events].sort(
    (a, b) => b.effectiveAt - a.effectiveAt || b.createdAt - a.createdAt,
  );
  const groups: YearGroup[] = [];
  for (const event of sorted) {
    const year = yearOf(event);
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
  /** The amend pane, read on intent when a row's Amend is pointed at or focused. */
  amendPane: { preload: () => Promise<unknown> };
  /** An event a link sent the reader to: marked for a moment (the view scrolls it into view). */
  highlightId?: string | null;
  /** The list's next page and its failure. */
  more: ListMore;
  /** The previous filter's rows, shown dimmed while the new ones load. */
  stale?: boolean;
  handleRef?: Ref<VirtualRowsHandle>;
};

/**
 * The employment history, newest first: the year as a heading that stays pinned while its
 * events scroll under it, each event on one rail with its kind's mark, its changes as chips,
 * its reason and who recorded it. Only the rows near the view are mounted, and the next page
 * is read as the end nears.
 */
export function TimelineList({
  events,
  canAmend,
  onAmend,
  amendPane,
  highlightId,
  more,
  stale = false,
  handleRef,
}: TimelineListProps) {
  const t = useT();
  const rows: VirtualRow[] = groupByYear(events).flatMap((group) => [
    {
      key: `year:${group.year}`,
      sticky: true,
      render: () => <YearHeading year={group.year} />,
    },
    ...group.events.map((event) => ({
      key: event.id,
      render: () => (
        <TimelineItem
          event={event}
          canAmend={canAmend}
          onAmend={onAmend}
          amendPane={amendPane}
          highlighted={highlightId === event.id}
        />
      ),
    })),
  ]);

  return (
    <VirtualRows
      rows={rows}
      aria-label={t("Employment history")}
      mask="card"
      className={cn("max-h-136", stale && "opacity-60 transition-opacity")}
      contentClassName="px-1.5 pb-2"
      hoverGlide
      estimateSize={76}
      handleRef={handleRef}
      onEndReached={more.hasMore && !more.loadMoreFailed ? more.loadMore : undefined}
      footer={<ListMoreLine more={more} />}
    />
  );
}

function YearHeading({ year }: { year: number }) {
  return (
    <div className="bg-card pt-3 pb-1 pl-2">
      <span
        data-testid={`timeline-year-${year}`}
        className="font-mono text-xs text-foreground-subtle tabular-nums"
      >
        {year}
      </span>
    </div>
  );
}

function TimelineItem({
  event,
  canAmend,
  onAmend,
  amendPane,
  highlighted,
}: {
  event: WorkerEmploymentEventRow;
  canAmend: boolean;
  onAmend: (event: WorkerEmploymentEventRow) => void;
  amendPane: { preload: () => Promise<unknown> };
  highlighted: boolean;
}) {
  const t = useT();
  const meta = employmentEventMeta(event.kind);
  const Icon = meta.icon;
  const name = employmentEventName(event.kind);
  const pairs = pairValues(event);
  const when = formatUnixDateMedium(event.effectiveAt);

  return (
    <div className="relative grid grid-cols-[3.25rem_minmax(0,1fr)]">
      <span aria-hidden className="absolute top-0 bottom-0 left-17 w-px bg-border" />
      <span aria-hidden />
      <div
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
                {...preloadOn(amendPane)}
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
      </div>
    </div>
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
