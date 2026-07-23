import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { DriverPayEventStatusBadge } from "@trenova/shared/components/status-badge";
import type { DriverPayEventRow } from "@/lib/graphql/driver-settlement";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import type { DriverPayEventStatus } from "@trenova/shared/types/driver-pay";

export function PayEventPanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<DriverPayEventRow>) {
  if (mode !== "edit" || !row) return null;

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={`Pay Event — ${row.proNumber || row.shipmentId}`}
      description={row.worker ? `${row.worker.firstName} ${row.worker.lastName}`.trim() : undefined}
      size="md"
    >
      <div className="flex flex-col gap-4 p-4">
        <div className="flex items-center gap-2">
          <DriverPayEventStatusBadge status={row.status as DriverPayEventStatus} />
          {row.voidReason && (
            <span className="text-xs text-red-600 dark:text-red-400">{row.voidReason}</span>
          )}
        </div>
        <div className="overflow-hidden rounded-lg border">
          <table className="w-full text-xs">
            <thead className="bg-muted/50 text-left">
              <tr>
                <th className="px-3 py-2 font-medium">Component</th>
                <th className="px-3 py-2 text-right font-medium">Qty × Rate</th>
                <th className="px-3 py-2 text-right font-medium">Amount</th>
              </tr>
            </thead>
            <tbody>
              {(row.components ?? []).map((component, index) => (
                <tr key={`${component.kind}-${index}`} className="border-t">
                  <td className="px-3 py-2 font-medium">{component.description}</td>
                  <td className="px-3 py-2 text-right text-muted-foreground tabular-nums">
                    {Number(component.quantity) > 0
                      ? `${Number(component.quantity).toLocaleString()} × ${Number(
                          component.rate,
                        ).toFixed(4)}`
                      : "—"}
                  </td>
                  <td className="px-3 py-2 text-right">
                    <AmountDisplay value={component.amountMinor} currency={row.currencyCode} />
                  </td>
                </tr>
              ))}
              <tr className="border-t bg-muted/30">
                <td className="px-3 py-2 font-semibold" colSpan={2}>
                  Total
                </td>
                <td className="px-3 py-2 text-right font-semibold">
                  <AmountDisplay
                    value={row.grossAmountMinor}
                    variant="positive"
                    currency={row.currencyCode}
                  />
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <p className="text-[11px] text-muted-foreground">
          Pay events accrue automatically when a shipment reaches your configured pay trigger and
          are locked once settled.
        </p>
      </div>
    </DataTablePanelContainer>
  );
}
