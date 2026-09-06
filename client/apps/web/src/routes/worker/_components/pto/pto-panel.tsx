import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import type { WorkerPTORow } from "@/lib/graphql/worker-table";
import { PTOFormDialog } from "./pto-form-dialog";

export function PTOPanel({ open, onOpenChange, mode, row }: DataTablePanelProps<WorkerPTORow>) {
  return (
    <PTOFormDialog open={open} onOpenChange={onOpenChange} pto={mode === "edit" ? row : null} />
  );
}
