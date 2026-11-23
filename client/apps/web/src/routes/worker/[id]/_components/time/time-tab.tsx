import { Kept } from "@/components/kept";
import { lazyComponent } from "@/lib/lazy-component";
import {
  CLOSED_WORKER_OPEN_STATE as CLOSED,
  type WorkerTabProps,
  type WorkerViewOf,
} from "@/routes/worker/use-worker-state";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { WORKER_VIEW_MODULES } from "../worker-view-modules";
import { TimeSkeleton } from "./time-skeleton";

const HoursOfService = lazyComponent(
  WORKER_VIEW_MODULES["hours-of-service"],
  (module) => module.default,
);
const Schedule = lazyComponent(WORKER_VIEW_MODULES.schedule, (module) => module.default);
const TimeOff = lazyComponent(WORKER_VIEW_MODULES["time-off"], (module) => module.default);
const Leave = lazyComponent(WORKER_VIEW_MODULES.leave, (module) => module.default);

function selectedView(view: WorkerTabProps["view"]): WorkerViewOf<"time"> {
  return view === "schedule" || view === "time-off" || view === "leave" ? view : "hours-of-service";
}

/**
 * Time: hours of service, schedule, time off and leave. Each view's code is read when it is first
 * shown (or earlier, when the section strip preloads it), and the view is kept once it has been.
 */
export default function TimeTab({ workerId, worker, view, open, onOpen }: WorkerTabProps) {
  const selected = selectedView(view);
  return (
    <>
      <Kept shown={selected === "hours-of-service"}>
        <LazyComponent fallback={<TimeSkeleton view="hours-of-service" />}>
          <HoursOfService.Component workerId={workerId} />
        </LazyComponent>
      </Kept>
      <Kept shown={selected === "schedule"}>
        <LazyComponent fallback={<TimeSkeleton view="schedule" />}>
          <Schedule.Component
            workerId={workerId}
            worker={worker}
            open={selected === "schedule" ? open : CLOSED}
            onOpen={onOpen}
          />
        </LazyComponent>
      </Kept>
      <Kept shown={selected === "time-off"}>
        <LazyComponent fallback={<TimeSkeleton view="time-off" />}>
          <TimeOff.Component
            workerId={workerId}
            worker={worker}
            open={selected === "time-off" ? open : CLOSED}
            onOpen={onOpen}
          />
        </LazyComponent>
      </Kept>
      <Kept shown={selected === "leave"}>
        <LazyComponent fallback={<TimeSkeleton view="leave" />}>
          <Leave.Component
            workerId={workerId}
            open={selected === "leave" ? open : CLOSED}
            onOpen={onOpen}
          />
        </LazyComponent>
      </Kept>
    </>
  );
}
