import type { WorkerTabProps } from "@/routes/worker/use-worker-state";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";
import { ActivitySkeleton } from "./activity-skeleton";

const WorkerActivity = lazy(() => import("./worker-activity"));

export default function ActivityTab({ workerId, onOpen }: WorkerTabProps) {
  return (
    <LazyComponent fallback={<ActivitySkeleton />}>
      <WorkerActivity workerId={workerId} onOpen={onOpen} />
    </LazyComponent>
  );
}
