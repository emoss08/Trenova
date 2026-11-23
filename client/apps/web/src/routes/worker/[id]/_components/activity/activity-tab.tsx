import type { WorkerTabProps } from "@/routes/worker/use-worker-state";
import WorkerActivity from "./worker-activity";

/**
 * The Activity tab is one view, so its chunk is the view: the page lazy-loads this module behind
 * the tab's skeleton, and nothing waits on a second read after it.
 */
export default function ActivityTab({ workerId, onOpen }: WorkerTabProps) {
  return <WorkerActivity workerId={workerId} onOpen={onOpen} />;
}
