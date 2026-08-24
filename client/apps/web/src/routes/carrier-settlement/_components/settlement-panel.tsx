import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import type { CarrierSettlementRow } from "@/lib/graphql/carrier-settlement";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { CarrierSettlementDetail } from "./settlement-detail";

export function CarrierSettlementPanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<CarrierSettlementRow>) {
  if (mode !== "edit" || !row) {
    return null;
  }

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={`Settlement ${row.settlementNumber}`}
      description={row.carrier?.name}
      size="xl"
    >
      <CarrierSettlementDetail settlementId={row.id} onClose={() => onOpenChange(false)} readOnly />
    </DataTablePanelContainer>
  );
}
