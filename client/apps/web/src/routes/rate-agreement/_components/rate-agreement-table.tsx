import { DataTable } from "@/components/data-table/data-table";
import { usePermission } from "@/hooks/use-permission";
import { rateAgreementTableGraphQLConfig } from "@/lib/graphql/rate-tables";
import { apiService } from "@/services/api";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { AddRecordAction, DockAction, RowAction } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { RateAgreement } from "@trenova/shared/types/rate";
import { CopyIcon, FileUpIcon, TrendingUpIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import { ImportRateSheetDialog } from "./import-rate-sheet-dialog";
import { getColumns } from "./rate-agreement-columns";
import { RateAgreementPanel } from "./rate-agreement-panel";
import { RateIncreaseDialog } from "./rate-increase-dialog";

export default function RateAgreementTable() {
  const columns = useMemo(() => getColumns(), []);
  const queryClient = useQueryClient();
  const { allowed: canDuplicate } = usePermission(Resource.RateAgreement, Operation.Duplicate);

  const [importOpen, setImportOpen] = useState(false);
  const [increaseOpen, setIncreaseOpen] = useState(false);
  const [increaseSelection, setIncreaseSelection] = useState<RateAgreement[]>([]);

  const { mutate: duplicateAgreement } = useMutation({
    mutationFn: (id: string) => apiService.rateAgreementService.duplicate(id),
    onSuccess: (copy) => {
      toast.success(`Duplicated as ${copy.code} — a fresh draft, ready to edit`);
      void queryClient.invalidateQueries({ queryKey: ["rate-agreement-list"] });
    },
    onError: () => toast.error("The agreement could not be duplicated"),
  });

  const addRecordActions = useMemo<AddRecordAction[]>(
    () => [
      {
        id: "import-rate-sheet",
        label: "Import Rate Sheet",
        description: "Upload a CSV or XLSX rate sheet into an agreement.",
        icon: FileUpIcon,
        onClick: () => setImportOpen(true),
      },
      {
        id: "rate-increase",
        label: "Apply Rate Increase",
        description: "Move every rate for a customer, a carrier, or across the board.",
        icon: TrendingUpIcon,
        onClick: () => {
          setIncreaseSelection([]);
          setIncreaseOpen(true);
        },
      },
    ],
    [],
  );

  const dockActions = useMemo<DockAction<RateAgreement>[]>(
    () => [
      {
        id: "rate-increase-selected",
        label: "Rate Increase",
        icon: TrendingUpIcon,
        clearSelectionOnSuccess: true,
        onClick: (selectedRows) => {
          setIncreaseSelection(selectedRows);
          setIncreaseOpen(true);
        },
      },
    ],
    [],
  );

  const contextMenuActions = useMemo<RowAction<RateAgreement>[]>(
    () => [
      {
        id: "duplicate-agreement",
        label: "Duplicate Agreement",
        icon: CopyIcon,
        hidden: () => !canDuplicate,
        onClick: (row) => {
          const id = row.original.id;
          if (id) duplicateAgreement(id);
        },
      },
    ],
    [canDuplicate, duplicateAgreement],
  );

  return (
    <>
      <DataTable<RateAgreement>
        name="Rate Agreement"
        queryKey="rate-agreement-list"
        graphql={rateAgreementTableGraphQLConfig}
        resource={Resource.RateAgreement}
        columns={columns}
        TablePanel={RateAgreementPanel}
        addRecordActions={addRecordActions}
        dockActions={dockActions}
        contextMenuActions={contextMenuActions}
        enableRowSelection
      />
      <ImportRateSheetDialog open={importOpen} onOpenChange={setImportOpen} />
      <RateIncreaseDialog
        open={increaseOpen}
        onOpenChange={setIncreaseOpen}
        selectedAgreements={increaseSelection}
      />
    </>
  );
}
