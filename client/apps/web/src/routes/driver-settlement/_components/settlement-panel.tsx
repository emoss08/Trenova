import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import type { DriverSettlementRow } from "@/lib/graphql/driver-settlement";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { SettlementDetail } from "./settlement-detail";

export function SettlementPanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<DriverSettlementRow>) {
  if (mode !== "edit" || !row) {
    return null;
  }

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={`Settlement ${row.settlementNumber}`}
      description={row.worker ? `${row.worker.firstName} ${row.worker.lastName}`.trim() : undefined}
      size="xl"
    >
      <SettlementDetail settlementId={row.id} onClose={() => onOpenChange(false)} readOnly />
    </DataTablePanelContainer>
  );
}
