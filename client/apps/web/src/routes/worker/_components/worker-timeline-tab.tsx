import { usePermission } from "@/hooks/use-permission";
import {
  fetchWorkerEmploymentEvents,
  WORKER_EMPLOYMENT_EVENTS_KEY,
  type WorkerEmploymentEventRow,
} from "@/lib/graphql/worker-employment";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
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

export default function WorkerTimelineTab({
  workerId,
  worker,
}: {
  workerId: string;
  worker: EmploymentSheetWorker;
}) {
  const { allowed: canRecord } = usePermission(Resource.WorkerEmploymentEvent, Operation.Create);
  const { allowed: canAmend } = usePermission(Resource.WorkerEmploymentEvent, Operation.Update);
  const [activeKinds, setActiveKinds] = useState<Set<EmploymentEventKind>>(() => new Set());
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
    () => (activeKinds.size === 0 ? events : events.filter((event) => activeKinds.has(event.kind))),
    [events, activeKinds],
  );

  const toggleKind = (kind: EmploymentEventKind) => {
    setActiveKinds((current) => {
      const next = new Set(current);
      if (next.has(kind)) {
        next.delete(kind);
      } else {
        next.add(kind);
      }
      return next;
    });
  };

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
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h3 className="text-sm font-semibold">Employment timeline</h3>
          <p className="text-muted-foreground text-xs">
            Every hire, transfer, leave and termination, with who recorded it.
          </p>
        </div>
        {canRecord ? (
          <Button size="sm" onClick={() => setSheet({ mode: "record" })}>
            <PlusIcon className="size-3.5" />
            Record event
          </Button>
        ) : null}
      </div>

      <LeaveStandingStrip workerId={workerId} />

      {presentKinds.length > 1 ? (
        <div className="flex flex-wrap gap-1.5" role="group" aria-label="Filter by event kind">
          {presentKinds.map((kind) => {
            const pressed = activeKinds.has(kind);
            return (
              <button
                key={kind}
                type="button"
                aria-pressed={pressed}
                onClick={() => toggleKind(kind)}
                className={cn(
                  "border-border rounded-full border px-2.5 py-0.5 text-[11px] font-medium transition-colors",
                  pressed
                    ? "bg-primary text-primary-foreground border-primary"
                    : "text-muted-foreground hover:bg-muted",
                )}
              >
                {EMPLOYMENT_EVENT_LABELS[kind]}
              </button>
            );
          })}
        </div>
      ) : null}

      {visible.length === 0 ? (
        <div className="border-border text-muted-foreground flex flex-col items-center gap-2 rounded-lg border border-dashed px-4 py-8 text-center text-sm">
          <HistoryIcon className="size-5" />
          {events.length === 0 ? "No employment events yet" : "Nothing matches the selected kinds"}
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
