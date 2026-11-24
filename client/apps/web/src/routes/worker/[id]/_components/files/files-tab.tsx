import { Kept } from "@/components/kept";
import { lazyComponent } from "@trenova/shared/lib/lazy-component";
import {
  CLOSED_WORKER_OPEN_STATE as CLOSED,
  type WorkerTabProps,
} from "@/routes/worker/use-worker-state";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { WORKER_VIEW_MODULES } from "../worker-view-modules";
import { FilesSkeleton } from "./files-skeleton";

const Documents = lazyComponent(WORKER_VIEW_MODULES.documents, (module) => module.default);
const DriverPortal = lazyComponent(
  WORKER_VIEW_MODULES["driver-portal"],
  (module) => module.default,
);
const AgentMemory = lazyComponent(WORKER_VIEW_MODULES["agent-memory"], (module) => module.default);

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
          <Documents.Component
            workerId={workerId}
            worker={worker}
            open={selected === "documents" ? open : CLOSED}
            onOpen={onOpen}
          />
        </LazyComponent>
      </Kept>
      <Kept shown={selected === "driver-portal"}>
        <LazyComponent fallback={<FilesSkeleton view="driver-portal" />}>
          <DriverPortal.Component workerId={workerId} worker={worker} onOpen={onOpen} />
        </LazyComponent>
      </Kept>
      <Kept shown={selected === "agent-memory"}>
        <LazyComponent fallback={<FilesSkeleton view="agent-memory" />}>
          <AgentMemory.Component workerId={workerId} workerName={worker.firstName} />
        </LazyComponent>
      </Kept>
    </>
  );
}
