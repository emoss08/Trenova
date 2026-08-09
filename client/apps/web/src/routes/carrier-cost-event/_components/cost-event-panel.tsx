import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { CarrierCostEventStatusBadge } from "@trenova/shared/components/status-badge";
import type { CarrierCostEventRow } from "@/lib/graphql/carrier-settlement";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import type {
  CarrierCostEventStatus,
  CarrierCostEventType,
} from "@trenova/shared/types/carrier-settlement";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { costEventTypeLabel } from "./cost-event-columns";

function formatDate(unix?: number | null): string {
  return formatUnixDateMedium(unix, { fallback: "—" });
}

export function CostEventPanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<CarrierCostEventRow>) {
  if (mode !== "edit" || !row) return null;

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={`Cost Event — ${row.proNumber || costEventTypeLabel(row.eventType as CarrierCostEventType)}`}
      description={row.carrier?.name}
      size="md"
    >
      <div className="flex flex-col gap-4 p-4">
        <div className="flex items-center gap-2">
          <CarrierCostEventStatusBadge status={row.status as CarrierCostEventStatus} />
          {row.voidReason && (
            <span className="text-xs text-red-600 dark:text-red-400">{row.voidReason}</span>
          )}
        </div>
        <div className="overflow-hidden rounded-lg border">
          <table className="w-full text-xs">
            <tbody>
              <tr className="border-b">
                <td className="px-3 py-2 font-medium">Type</td>
                <td className="px-3 py-2 text-right">
                  {costEventTypeLabel(row.eventType as CarrierCostEventType)}
                </td>
              </tr>
              <tr className="border-b">
                <td className="px-3 py-2 font-medium">Description</td>
                <td className="px-3 py-2 text-right">{row.description || "—"}</td>
              </tr>
              <tr className="border-b">
                <td className="px-3 py-2 font-medium">Accrued</td>
                <td className="px-3 py-2 text-right">{formatDate(row.eventDate)}</td>
              </tr>
              <tr className="border-b">
                <td className="px-3 py-2 font-medium">Pro Number</td>
                <td className="px-3 py-2 text-right font-mono">{row.proNumber || "—"}</td>
              </tr>
              <tr className="border-b">
                <td className="px-3 py-2 font-medium">Settlement</td>
                <td className="px-3 py-2 text-right font-mono">
                  {row.settlementId || "Unsettled"}
                </td>
              </tr>
              <tr className="bg-muted/30">
                <td className="px-3 py-2 font-semibold">Amount</td>
                <td className="px-3 py-2 text-right font-semibold">
                  <AmountDisplay
                    value={row.amountMinor}
                    variant="auto"
                    currency={row.currencyCode}
                  />
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <p className="text-[11px] text-muted-foreground">
          Cost events accrue automatically when a carrier-covered shipment reaches your configured
          pay trigger and are locked once attached to a settlement.
        </p>
      </div>
    </DataTablePanelContainer>
  );
}
