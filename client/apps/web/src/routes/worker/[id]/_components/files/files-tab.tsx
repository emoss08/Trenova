import { Kept } from "@/components/kept";
import type { WorkerTabProps } from "@/routes/worker/use-worker-state";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";
import { FilesSkeleton } from "./files-skeleton";

const Documents = lazy(() => import("./documents/worker-documents"));
const DriverPortal = lazy(() => import("./driver-portal/worker-portal-tab"));
const AgentMemory = lazy(() => import("./agent-memory/worker-memories"));

/**
 * The worker's files: their documents, their side of Dash, and what agents remember about them.
 * Uploading opens the page's work pane from `form=upload-document`.
 */
export default function FilesTab({ workerId, worker, view, open, onOpen }: WorkerTabProps) {
  const selected = view ?? "documents";
  return (
    <>
      <Kept shown={selected === "documents"}>
        <LazyComponent fallback={<FilesSkeleton view="documents" />}>
          <Documents workerId={workerId} worker={worker} open={open} onOpen={onOpen} />
        </LazyComponent>
      </Kept>
      <Kept shown={selected === "driver-portal"}>
        <LazyComponent fallback={<FilesSkeleton view="driver-portal" />}>
          <DriverPortal workerId={workerId} worker={worker} onOpen={onOpen} />
        </LazyComponent>
      </Kept>
      <Kept shown={selected === "agent-memory"}>
        <LazyComponent fallback={<FilesSkeleton view="agent-memory" />}>
          <AgentMemory workerId={workerId} workerName={worker.firstName} />
        </LazyComponent>
      </Kept>
    </>
  );
}
