import { SectionPanel } from "@/components/section-panel";
import type { VirtualRowsHandle } from "@/components/virtual-rows";
import { lazyWorkPane, preloadOn } from "@/components/work-pane";
import { usePermission } from "@/hooks/use-permission";
import {
  fetchWorkerEmploymentEvent,
  fetchWorkerEmploymentEvents,
  WORKER_EMPLOYMENT_EVENTS_KEY,
  type WorkerEmploymentEventRow,
} from "@/lib/graphql/worker-employment";
import { workerOverviewQuery } from "@/lib/queries/worker";
import type { WorkerRecord } from "@/routes/worker/use-worker-state";
import { useQuery } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { PlusIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet } from "@trenova/shared/components/ui/empty-sheet";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useEffect, useRef, useState } from "react";
import { LeaveStandingStrip } from "../../time/leave/leave-standing-strip";
import { ListSketch } from "../../list-sketch";
import { useRecordById } from "../../use-record-by-id";
import { useWorkerFormGate } from "../../use-worker-form-gate";
import { useWorkerList } from "../../use-worker-list";
import { WORKER_FORM_PANES } from "../../worker-form-panes";
import { HistorySkeletonBody } from "../employment-skeleton";
import { EmploymentDates } from "./employment-dates";
import {
  EMPLOYMENT_EVENT_GROUP_LABELS,
  EMPLOYMENT_EVENT_GROUPS,
  employmentEventKindsIn,
  type EmploymentEventGroup,
} from "./employment-event-meta";
import { TimelineList } from "./timeline-list";

type KindFilter = EmploymentEventGroup | "all";

const AmendEventPane = lazyWorkPane(
  WORKER_FORM_PANES["amend-event"],
  (module) => module.AmendEventPane,
  {
    isOpen: (props) => props.event !== null,
    openKey: (props) => props.event?.id,
    title: (_props, t) => t("Amend event"),
    fields: 4,
  },
);

export type HistoryViewProps = {
  workerId: string;
  worker: WorkerRecord;
  /** An event a link sent the reader to (`event` in the address). */
  highlightId: string | null;
  /** The event the address opens the Amend pane on (`form=amend-event`). */
  amendingId: string | null;
  /** The Record event pane, read on intent from the empty state's button. */
  recordPane: { preload: () => Promise<unknown> };
  onRecord: () => void;
  onAmend: (event: WorkerEmploymentEventRow) => void;
  onCloseAmend: () => void;
};

/**
 * Employment › History: every event, year by year and newest first, a page at a time, filtered
 * by kind on the server, beside the dates the employment turns on and the worker's leave
 * standing. The count is the overview's, or a kind's total from the server, so it never depends
 * on how much has been paged in.
 */
export default function WorkerTimelineTab({
  workerId,
  worker,
  highlightId,
  amendingId,
  recordPane,
  onRecord,
  onAmend,
  onCloseAmend,
}: HistoryViewProps) {
  const t = useT();
  const { allowed: canRecord } = usePermission(Resource.WorkerEmploymentEvent, Operation.Create);
  const { allowed: canAmend } = usePermission(Resource.WorkerEmploymentEvent, Operation.Update);
  const [filter, setFilter] = useState<KindFilter>("all");
  const listRef = useRef<VirtualRowsHandle>(null);
  const overview = useQuery(workerOverviewQuery(workerId));
  const events = useWorkerList({
    root: WORKER_EMPLOYMENT_EVENTS_KEY,
    workerId,
    variables: [filter],
    keepPreviousRows: true,
    fetchPage: ({ first, after, signal }) =>
      fetchWorkerEmploymentEvents(
        {
          workerId,
          kinds: filter === "all" ? undefined : employmentEventKindsIn(filter),
          first,
          after,
        },
        { signal },
      ),
  });
  const rows = events.rows;
  const highlighted = highlightId !== null && rows.some((event) => event.id === highlightId);
  // The event the Amend pane opens on: among the pages read, or past them, read by id.
  const amendRecord = useRecordById({
    root: WORKER_EMPLOYMENT_EVENTS_KEY,
    workerId,
    id: amendingId,
    rows,
    rowsSettled: !events.isPending,
    fetchRecord: fetchWorkerEmploymentEvent,
    enabled: canAmend,
  });

  useEffect(() => {
    if (highlighted && highlightId) listRef.current?.scrollToKey(highlightId);
  }, [highlighted, highlightId]);

  const opens = useWorkerFormGate(amendingId === null ? null : "amend-event", {
    "amend-event": amendRecord.isPending ? "waiting" : canAmend && amendRecord.record !== null,
  });

  if (events.isPending) return <HistorySkeletonBody />;

  if (events.isError) {
    return (
      <ErrorState
        layout="sheet"
        error={events.error}
        onRetry={events.refetch}
        title={t("The employment history could not be loaded")}
        sketch={<ListSketch />}
      />
    );
  }

  const total = overview.data?.employment?.eventCount ?? null;
  const hasAny = filter !== "all" || rows.length > 0 || (total ?? 0) > 0;
  const amending = opens("amend-event") ? amendRecord.record : null;
  // The overview's count for every event; a kind's own count is the server's, on its first page.
  const count = filter === "all" ? total : events.isStale ? null : events.total;
  const filterItems = [
    { value: "all", label: t("All kinds") },
    ...EMPLOYMENT_EVENT_GROUPS.map((group) => ({
      value: group,
      label: EMPLOYMENT_EVENT_GROUP_LABELS[group],
    })),
  ];

  return (
    <div className="grid items-start gap-4 lg:grid-cols-[minmax(0,1fr)_22rem]">
      <SectionPanel
        title={t("Employment history")}
        count={count ?? undefined}
        action={
          hasAny ? (
            <Select
              value={filter}
              items={filterItems}
              onValueChange={(value) => setFilter(value as KindFilter)}
            >
              <SelectTrigger
                className="w-auto gap-1.5 text-xs"
                aria-label={t("Show events of kind")}
              >
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {filterItems.map((item) => (
                  <SelectItem key={item.value} value={item.value}>
                    {item.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          ) : null
        }
      >
        {!hasAny ? (
          <EmptySheet
            title={t("No employment events yet")}
            description={t(
              "Hiring, transfers, rate changes, leave and the end of employment are recorded here, newest first.",
            )}
            sketch={<ListSketch />}
            action={
              canRecord ? (
                <Button size="sm" {...preloadOn(recordPane)} onClick={onRecord}>
                  <PlusIcon className="size-3.5" />
                  {t("Record event")}
                </Button>
              ) : undefined
            }
          />
        ) : rows.length === 0 ? (
          <p className="px-3 py-3 text-xs text-muted-foreground">
            {t("No events of this kind.")}{" "}
            <button
              type="button"
              className="rounded-sm ui-focus-ring text-brand hover:underline"
              onClick={() => setFilter("all")}
            >
              {t("Show all kinds")}
            </button>
          </p>
        ) : (
          <TimelineList
            events={rows}
            canAmend={canAmend}
            onAmend={onAmend}
            amendPane={AmendEventPane}
            highlightId={highlightId}
            more={events}
            stale={events.isStale}
            handleRef={listRef}
          />
        )}
      </SectionPanel>

      <div className="flex min-w-0 flex-col gap-4">
        <EmploymentDates workerId={workerId} worker={worker} />
        <LeaveStandingStrip workerId={workerId} />
      </div>

      <AmendEventPane.Component event={amending} workerId={workerId} onClose={onCloseAmend} />
    </div>
  );
}
