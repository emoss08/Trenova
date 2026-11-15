import type { BulkEditJob } from "@/lib/graphql/bulk-edit";
import { useDataTableView, useDataTableViewActions } from "../data-table-view-context";
import type { DataTableViewState } from "../data-table-view-store";
import { BulkEditDialog } from "./bulk-edit-dialog";
import { BulkEditProgress } from "./bulk-edit-progress";

const selectSelection = (state: DataTableViewState) => state.bulkEditSelection;
const selectJobId = (state: DataTableViewState) => state.bulkEditJobId;

/**
 * The table's bulk edit: the editor while the person chooses what to change, then the edit's
 * progress once it runs. Both open and close through the table's view, so nothing else in the
 * table is drawn again for them.
 */
export function DataTableBulkEdit({
  resource,
  onDone,
}: {
  resource: string | undefined;
  /** The edit finished: the table's rows are read again, and a clean run clears the selection. */
  onDone: (job: BulkEditJob) => void;
}) {
  const selection = useDataTableView(selectSelection);
  const jobId = useDataTableView(selectJobId);
  const { closeBulkEdit, trackBulkEditJob, dismissBulkEditJob } = useDataTableViewActions();

  return (
    <>
      {resource && selection ? (
        <BulkEditDialog
          open
          onOpenChange={(open) => {
            if (!open) closeBulkEdit();
          }}
          resource={resource}
          selection={selection}
          onStarted={(job) => trackBulkEditJob(job.id)}
        />
      ) : null}
      {jobId ? <BulkEditProgress jobId={jobId} onDismiss={dismissBulkEditJob} onDone={onDone} /> : null}
    </>
  );
}
