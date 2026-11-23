import type { WorkerTabProps } from "@/routes/worker/use-worker-state";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";
import { SafetySkeleton } from "./safety-skeleton";

const Safety = lazy(() => import("./worker-safety-tab"));

export default function SafetyTab(props: WorkerTabProps) {
  return (
    <LazyComponent fallback={<SafetySkeleton />}>
      <Safety {...props} />
    </LazyComponent>
  );
}
