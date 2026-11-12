import { useQueryStates } from "nuqs";
import { useCallback, useState } from "react";
import { transferRunSearchParamsParser } from "./use-billing-queue-state";

/**
 * The bulk transfer dialog and the run it watches. A run id in the URL means somebody
 * followed the completion toast back to a report, so the dialog opens on it. Closing
 * lets go of the run, so a refresh never reopens a report somebody dismissed; opening
 * again reattaches to a run that is still going.
 */
export function useTransferDialog() {
  const [{ transferRun }, setTransferParams] = useQueryStates(transferRunSearchParamsParser);
  const [open, setOpen] = useState(() => Boolean(transferRun));

  const onOpenChange = useCallback(
    (next: boolean) => {
      setOpen(next);
      if (!next) void setTransferParams({ transferRun: null });
    },
    [setTransferParams],
  );

  const onRunIdChange = useCallback(
    (runId: string | null) => {
      void setTransferParams({ transferRun: runId });
      if (runId) setOpen(true);
    },
    [setTransferParams],
  );

  return { open, onOpenChange, runId: transferRun, onRunIdChange };
}
