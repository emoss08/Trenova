import { SectionPanel } from "@/components/section-panel";
import { usePermission } from "@/hooks/use-permission";
import type { WorkerEmploymentEventRow } from "@/lib/graphql/worker-employment";
import type { WorkerRecord } from "@/routes/worker/use-worker-state";
import { useQuery } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { PlusIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet } from "@trenova/shared/components/ui/empty-sheet";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useState } from "react";
import { LeaveStandingStrip } from "../../time/leave/leave-standing-strip";
import { HistorySkeletonBody } from "../employment-skeleton";
import { employmentEventsQuery } from "../employment-reads";
import { ListSketch } from "../list-sketch";
import { EmploymentDates } from "./employment-dates";
import {
  EMPLOYMENT_EVENT_GROUP_LABELS,
  EMPLOYMENT_EVENT_GROUPS,
  employmentEventMeta,
  type EmploymentEventGroup,
} from "./employment-event-meta";
import { TimelineList } from "./timeline-list";

type KindFilter = EmploymentEventGroup | "all";

export type HistoryViewProps = {
  workerId: string;
  worker: WorkerRecord;
  /** An event a link sent the reader to (`event` in the address). */
  highlightId: string | null;
  now: number;
  onRecord: () => void;
  onAmend: (event: WorkerEmploymentEventRow) => void;
};

/**
 * Employment › History: every event, year by year, filtered by kind, beside the dates the
 * employment turns on and the worker's leave standing.
 */
export default function WorkerTimelineTab({
  workerId,
  worker,
  highlightId,
  now,
  onRecord,
  onAmend,
}: HistoryViewProps) {
  const t = useT();
  const { allowed: canRecord } = usePermission(Resource.WorkerEmploymentEvent, Operation.Create);
  const { allowed: canAmend } = usePermission(Resource.WorkerEmploymentEvent, Operation.Update);
  const [filter, setFilter] = useState<KindFilter>("all");
  const eventsQuery = useQuery(employmentEventsQuery(workerId));

  if (eventsQuery.isPending) return <HistorySkeletonBody />;

  if (eventsQuery.isError) {
    return (
      <ErrorState
        layout="sheet"
        error={eventsQuery.error}
        onRetry={() => void eventsQuery.refetch()}
        title={t("Couldn't load the employment history")}
        sketch={<ListSketch />}
      />
    );
  }

  const events = eventsQuery.data;
  const visible =
    filter === "all"
      ? events
      : events.filter((event) => employmentEventMeta(event.kind).group === filter);
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
        hint={
          events.length > 0
            ? t("{0} of {1, plural, one {# event} other {# events}}", visible.length, events.length)
            : undefined
        }
        action={
          events.length > 0 ? (
            <Select
              value={filter}
              items={filterItems}
              onValueChange={(value) => setFilter(value as KindFilter)}
            >
              <SelectTrigger
                size="sm"
                className="h-6 w-auto gap-1.5 text-xs"
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
        {events.length === 0 ? (
          <EmptySheet
            title={t("No employment events yet")}
            description={t(
              "Hiring, transfers, rate changes, leave and the end of employment are recorded here, newest first.",
            )}
            sketch={<ListSketch />}
            action={
              canRecord ? (
                <Button size="sm" onClick={onRecord}>
                  <PlusIcon className="size-3.5" />
                  {t("Record event")}
                </Button>
              ) : undefined
            }
          />
        ) : visible.length === 0 ? (
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
          <ScrollArea className="max-h-[34rem]" maskVariant="card">
            <TimelineList
              events={visible}
              canAmend={canAmend}
              onAmend={onAmend}
              highlightId={highlightId}
            />
          </ScrollArea>
        )}
      </SectionPanel>

      <div className="flex min-w-0 flex-col gap-4">
        <EmploymentDates worker={worker} events={events} now={now} />
        <LeaveStandingStrip workerId={workerId} />
      </div>
    </div>
  );
}
