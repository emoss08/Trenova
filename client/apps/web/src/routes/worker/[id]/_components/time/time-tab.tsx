import { Kept } from "@/components/kept";
import type { WorkerTabProps, WorkerViewOf } from "@/routes/worker/use-worker-state";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";
import { TimeSkeleton } from "./time-skeleton";

const HoursOfService = lazy(() => import("./hours/hours-of-service-view"));
const Schedule = lazy(() => import("./schedule/schedule-view"));
const TimeOff = lazy(() => import("./time-off/time-off-view"));
const Leave = lazy(() => import("./leave/leave-view"));

function selectedView(view: WorkerTabProps["view"]): WorkerViewOf<"time"> {
  return view === "schedule" || view === "time-off" || view === "leave" ? view : "hours-of-service";
}

/** Time: hours of service, schedule, time off and leave, each kept once it has been shown. */
export default function TimeTab({ workerId, worker, view, open, onOpen }: WorkerTabProps) {
  const selected = selectedView(view);
  return (
    <>
      <Kept shown={selected === "hours-of-service"}>
        <LazyComponent fallback={<TimeSkeleton view="hours-of-service" />}>
          <HoursOfService workerId={workerId} />
        </LazyComponent>
      </Kept>
      <Kept shown={selected === "schedule"}>
        <LazyComponent fallback={<TimeSkeleton view="schedule" />}>
          <Schedule workerId={workerId} worker={worker} open={open} onOpen={onOpen} />
        </LazyComponent>
      </Kept>
      <Kept shown={selected === "time-off"}>
        <LazyComponent fallback={<TimeSkeleton view="time-off" />}>
          <TimeOff workerId={workerId} worker={worker} open={open} onOpen={onOpen} />
        </LazyComponent>
      </Kept>
      <Kept shown={selected === "leave"}>
        <LazyComponent fallback={<TimeSkeleton view="leave" />}>
          <Leave workerId={workerId} open={open} onOpen={onOpen} />
        </LazyComponent>
      </Kept>
    </>
  );
}
