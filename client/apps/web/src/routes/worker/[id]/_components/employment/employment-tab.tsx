import { Kept } from "@/components/kept";
import { lazyWorkPane, useWorkPane } from "@/components/work-pane";
import { usePermission } from "@/hooks/use-permission";
import { lazyComponent } from "@/lib/lazy-component";
import type { WorkerTabProps, WorkerTarget, WorkerViewOf } from "@/routes/worker/use-worker-state";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { ChecklistsSkeletonBody, HistorySkeletonBody } from "./employment-skeleton";
import { EmploymentLifecycle } from "./lifecycle/employment-lifecycle";
import { WORKER_FORM_PANES } from "../worker-form-panes";
import { WORKER_VIEW_MODULES } from "../worker-view-modules";
import { WorkerSectionStrip } from "../worker-strip-slot";

const History = lazyComponent(WORKER_VIEW_MODULES.history, (module) => module.default);
const Checklists = lazyComponent(WORKER_VIEW_MODULES.checklists, (module) => module.default);

const RecordEventPane = lazyWorkPane(
  WORKER_FORM_PANES["record-event"],
  (module) => module.RecordEventPane,
  { title: (_props, t) => t("Record employment event"), fields: 4 },
);

/**
 * Employment: the lifecycle route, then the section strip the shell hands it (the route comes
 * first, so the tab places the strip), then the selected view (History or Checklists), and the
 * Record event pane, which the address opens (`form=record-event`, `kind` presets the event).
 * History owns the Amend pane (`form=amend-event` with `event`), since it holds the rows.
 */
export default function EmploymentTab({ workerId, worker, view, open, onOpen }: WorkerTabProps) {
  const selected: WorkerViewOf<"employment"> = view === "checklists" ? "checklists" : "history";
  const { closeThen } = useWorkPane();
  const { allowed: canRecord } = usePermission(Resource.WorkerEmploymentEvent, Operation.Create);

  const go = (target: Omit<Extract<WorkerTarget, { tab: "employment" }>, "tab">) =>
    closeThen(() => onOpen({ tab: "employment", ...target }));
  const closePane = () => onOpen({ tab: "employment", view: selected });
  const recordEvent = () => go({ view: selected, form: "record-event" });

  return (
    <div className="flex flex-col gap-4">
      <EmploymentLifecycle
        workerId={workerId}
        hireDate={worker.profile?.hireDate ?? null}
        onRecord={canRecord ? recordEvent : null}
        recordPane={RecordEventPane}
      />
      <WorkerSectionStrip />

      <Kept shown={selected === "history"}>
        <LazyComponent fallback={<HistorySkeletonBody />}>
          <History.Component
            workerId={workerId}
            worker={worker}
            highlightId={open.form ? null : open.event}
            amendingId={open.form === "amend-event" ? open.event : null}
            recordPane={RecordEventPane}
            onRecord={recordEvent}
            onAmend={(event) => go({ view: "history", form: "amend-event", event: event.id })}
            onCloseAmend={closePane}
          />
        </LazyComponent>
      </Kept>
      <Kept shown={selected === "checklists"}>
        <LazyComponent fallback={<ChecklistsSkeletonBody />}>
          <Checklists.Component workerId={workerId} focusId={open.checklist} />
        </LazyComponent>
      </Kept>

      <RecordEventPane.Component
        open={open.form === "record-event" && canRecord}
        presetKind={open.kind}
        workerId={workerId}
        worker={worker}
        onClose={closePane}
      />
    </div>
  );
}
