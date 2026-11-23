import type { WorkerForm } from "@/routes/worker/use-worker-state";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { createContext, use, useEffect } from "react";

/** Takes the form out of the address, in place; the page provides it. */
export const WorkerFormDropContext = createContext<() => void>(() => {});

/**
 * Whether a form the address names can open: `true` opens it, `false` refuses it (its record is
 * missing or not found, not in an editable state, or the reader may not do it), `"waiting"` while
 * the reads that decide it are still out.
 */
export type WorkerFormDecision = boolean | "waiting";

/**
 * The one rule for a form the address names: it opens when its tab says it can, and when its tab
 * says it cannot, `form` leaves the address (in place, so Back does not return to it), so neither
 * the Add to record menu nor the unsaved-changes guard is left believing a pane is open, and a
 * reload does not bring the dead state back. A tab lists a decision for every form it owns and
 * opens each pane through the returned function:
 *
 *   const opens = useWorkerFormGate(open.form, {
 *     "answer-query": fileQuery.isPending ? "waiting" : canUpdate && answerQuery != null,
 *   });
 *   <AnswerQueryPane.Component open={opens("answer-query")} … />
 *
 * What the address alone rules out (a form whose record it does not name, or whose view the
 * reader may not draw) the page drops before any tab decides (`workerFormBlockedByAddress`).
 */
export function useWorkerFormGate(
  form: WorkerForm | null,
  decisions: Partial<Record<WorkerForm, WorkerFormDecision>>,
): (candidate: WorkerForm) => boolean {
  const drop = use(WorkerFormDropContext);
  // A refusal for want of a permission means nothing until the reader's permissions are known.
  const permissionsKnown = usePermissionStore((store) => store.manifest) !== null;
  const refused = permissionsKnown && form !== null && decisions[form] === false;
  useEffect(() => {
    if (refused) drop();
  }, [refused, drop]);
  return (candidate) => form === candidate && decisions[candidate] === true;
}
