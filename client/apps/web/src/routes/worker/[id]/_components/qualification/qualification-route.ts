import { useWorkPane } from "@/components/work-pane";
import type { WorkerOpenState, WorkerTarget, WorkerViewOf } from "@/routes/worker/use-worker-state";

export type QualificationView = WorkerViewOf<"qualification">;

/** What each Qualification view is given by the tab: the address's open pane and the opener. */
export type QualificationViewProps = {
  workerId: string;
  open: WorkerOpenState;
  onOpen: (target: WorkerTarget) => void;
};

/**
 * How a view opens and closes its panes through the address. `go` asks a dirty pane first, as
 * every change of pane, view or tab must; `closeWhen` returns the `onOpenChange` a pane takes,
 * which clears the address only while it still points at that pane, so a pane displaced by
 * another never wipes the one that replaced it.
 */
export function useQualificationRoute(
  view: QualificationView,
  onOpen: (target: WorkerTarget) => void,
) {
  const { closeThen } = useWorkPane();
  const go = (target: WorkerTarget) => closeThen(() => onOpen(target));
  const closeWhen = (stillOpen: boolean) => (next: boolean) => {
    if (!next && stillOpen) onOpen({ tab: "qualification", view });
  };
  return { go, closeWhen };
}
