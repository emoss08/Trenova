import { Kept } from "@/components/kept";
import { useWorkPane } from "@/components/work-pane";
import { usePermission } from "@/hooks/use-permission";
import type { WorkerTabProps, WorkerTarget, WorkerViewOf } from "@/routes/worker/use-worker-state";
import { useQuery } from "@tanstack/react-query";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { lazy, useState } from "react";
import { employmentEventsQuery, workerChecklistsQuery } from "./employment-reads";
import {
  ChecklistsSkeletonBody,
  HistorySkeletonBody,
  LifecycleRouteSkeleton,
} from "./employment-skeleton";
import { AmendEventPane } from "./history/amend-event-pane";
import { RecordEventPane } from "./history/record-event-pane";
import { buildLifecycle } from "./lifecycle/lifecycle-model";
import { LifecycleRoute } from "./lifecycle/lifecycle-route";
import { WorkerSectionStrip } from "../worker-strip-slot";

const History = lazy(() => import("./history/worker-timeline-tab"));
const Checklists = lazy(() => import("./checklists/worker-checklist-tab"));

/**
 * Employment: the lifecycle route, then the section strip the shell hands it (the route comes
 * first, so the tab places the strip), then the selected view (History or Checklists), and the
 * Record event and Amend panes, which the address opens (`form=record-event`, `kind` presets
 * the event; `form=amend-event` with `event`).
 */
export default function EmploymentTab({ workerId, worker, view, open, onOpen }: WorkerTabProps) {
  const t = useT();
  const selected: WorkerViewOf<"employment"> = view === "checklists" ? "checklists" : "history";
  const { closeThen } = useWorkPane();
  const { allowed: canRecord } = usePermission(Resource.WorkerEmploymentEvent, Operation.Create);
  const { allowed: canAmend } = usePermission(Resource.WorkerEmploymentEvent, Operation.Update);
  const [now] = useState(() => Math.floor(Date.now() / 1000));

  const eventsQuery = useQuery(employmentEventsQuery(workerId));
  const checklistsQuery = useQuery(workerChecklistsQuery(workerId));

  const go = (target: Omit<Extract<WorkerTarget, { tab: "employment" }>, "tab">) =>
    closeThen(() => onOpen({ tab: "employment", ...target }));
  const closePane = () => onOpen({ tab: "employment", view: selected });
  const recordEvent = () => go({ view: selected, form: "record-event" });

  const amending =
    canAmend && open.form === "amend-event" && open.event
      ? (eventsQuery.data?.find((event) => event.id === open.event) ?? null)
      : null;

  return (
    <div className="flex flex-col gap-4">
      {eventsQuery.data && checklistsQuery.data ? (
        <LifecycleRoute
          stops={buildLifecycle(
            {
              events: eventsQuery.data,
              checklists: checklistsQuery.data,
              hireDate: worker.profile?.hireDate ?? null,
              now,
            },
            t,
          )}
          onRecord={canRecord ? recordEvent : null}
        />
      ) : eventsQuery.isError || checklistsQuery.isError ? null : (
        <LifecycleRouteSkeleton />
      )}
      <WorkerSectionStrip />

      <Kept shown={selected === "history"}>
        <LazyComponent fallback={<HistorySkeletonBody />}>
          <History
            workerId={workerId}
            worker={worker}
            highlightId={open.form ? null : open.event}
            now={now}
            onRecord={recordEvent}
            onAmend={(event) => go({ view: "history", form: "amend-event", event: event.id })}
          />
        </LazyComponent>
      </Kept>
      <Kept shown={selected === "checklists"}>
        <LazyComponent fallback={<ChecklistsSkeletonBody />}>
          <Checklists workerId={workerId} focusId={open.checklist} />
        </LazyComponent>
      </Kept>

      <RecordEventPane
        open={open.form === "record-event" && canRecord}
        presetKind={open.kind}
        workerId={workerId}
        worker={worker}
        onClose={closePane}
      />
      <AmendEventPane event={amending} workerId={workerId} onClose={closePane} />
    </div>
  );
}
