import { useT } from "@trenova/shared/i18n/use-t";
import { InfoPopover } from "@/components/info-popover";
import { usePermission } from "@/hooks/use-permission";
import {
  fetchWorkerEmploymentEvents,
  WORKER_EMPLOYMENT_EVENTS_KEY,
  type WorkerEmploymentEventRow,
} from "@/lib/graphql/worker-employment";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Operation, Resource } from "@trenova/shared/types/permission";
import {
  EMPLOYMENT_EVENT_LABELS,
  type EmploymentEventKind,
} from "@trenova/shared/types/worker-employment";
import { HistoryIcon, PlusIcon } from "lucide-react";
import { useMemo, useState } from "react";
import {
  EmploymentEventSheet,
  type EmploymentSheetWorker,
} from "./timeline/employment-event-dialog";
import { LeaveStandingStrip } from "./leave/leave-standing-strip";
import { ALL_EMPLOYMENT_EVENT_KINDS } from "./timeline/employment-event-meta";
import { TimelineList } from "./timeline/timeline-list";

type SheetState = { mode: "record" } | { mode: "amend"; event: WorkerEmploymentEventRow };
type KindFilter = EmploymentEventKind | "all";

export default function WorkerTimelineTab({
  workerId,
  worker,
}: {
  workerId: string;
  worker: EmploymentSheetWorker;
}) {
  const t = useT();

  const { allowed: canRecord } = usePermission(Resource.WorkerEmploymentEvent, Operation.Create);
  const { allowed: canAmend } = usePermission(Resource.WorkerEmploymentEvent, Operation.Update);
  const [kind, setKind] = useState<KindFilter>("all");
  const [sheet, setSheet] = useState<SheetState | null>(null);

  const eventsQuery = useQuery({
    queryKey: [WORKER_EMPLOYMENT_EVENTS_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerEmploymentEvents(workerId, undefined, { signal }),
  });
  const events = useMemo(() => eventsQuery.data ?? [], [eventsQuery.data]);

  const presentKinds = useMemo(() => {
    const present = new Set(events.map((event) => event.kind));
    return ALL_EMPLOYMENT_EVENT_KINDS.filter((kind) => present.has(kind));
  }, [events]);

  const visible = useMemo(
    () => (kind === "all" ? events : events.filter((event) => event.kind === kind)),
    [events, kind],
  );
  const kindOptions = useMemo<{ value: KindFilter; label: string }[]>(
    () => [
      { value: "all", label: t("All events") },
      ...presentKinds.map((option) => ({ value: option, label: EMPLOYMENT_EVENT_LABELS[option] })),
    ],
    [presentKinds, t],
  );

  if (eventsQuery.isLoading) {
    return (
      <div className="flex flex-col gap-3">
        <Skeleton className="h-9 w-full" />
        <Skeleton className="h-20 w-full" />
        <Skeleton className="h-20 w-full" />
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <LeaveStandingStrip workerId={workerId} />

      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-1.5">
          <p className="text-muted-foreground text-xs">
            {events.length === 0
              ? t("Nothing recorded yet.")
              : t(
                  "{0} of {1, plural, one {# event} other {# events}} shown.",
                  visible.length,
                  events.length,
                )}
          </p>
          <InfoPopover title={t("Employment events")}>
            <p>
              {t(
                "Recording an event is the only way a worker's employment status moves. A leave or suspension takes them off the dispatch board, a termination ends employment and closes their PTO and pay assignments, a rehire reopens it.",
              )}
            </p>
            <p>
              {t(
                "Amending an event corrects what was written and never replays those effects: a corrected termination date does not move the worker's termination date.",
              )}
            </p>
          </InfoPopover>
        </div>
        <div className="flex items-center gap-2">
          {presentKinds.length > 1 ? (
            <Select
              value={kind}
              items={kindOptions}
              onValueChange={(value) => setKind(value as KindFilter)}
            >
              <SelectTrigger className="h-7 w-40 text-xs" aria-label={t("Filter by event kind")}>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {kindOptions.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {t(option.label)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          ) : null}
          {canRecord ? (
            <Button size="sm" onClick={() => setSheet({ mode: "record" })}>
              <PlusIcon className="size-3.5" />
              {t("Record event")}
            </Button>
          ) : null}
        </div>
      </div>

      {visible.length === 0 ? (
        <div className="border-border text-muted-foreground flex flex-col items-center gap-2 rounded-lg border border-dashed px-4 py-8 text-center text-sm">
          <HistoryIcon className="size-5" />
          {events.length === 0
            ? t("No employment events yet")
            : t("Nothing matches the selected kinds")}
        </div>
      ) : (
        <TimelineList
          events={visible}
          canAmend={canAmend}
          onAmend={(event) => setSheet({ mode: "amend", event })}
        />
      )}

      {sheet ? (
        <EmploymentEventSheet
          open
          onOpenChange={(open) => {
            if (!open) setSheet(null);
          }}
          workerId={workerId}
          worker={worker}
          mode={sheet.mode}
          event={sheet.mode === "amend" ? sheet.event : undefined}
          history={events}
        />
      ) : null}
    </div>
  );
}
