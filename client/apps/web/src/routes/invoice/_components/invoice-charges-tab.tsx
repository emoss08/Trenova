import { Badge, type BadgeVariant } from "@trenova/shared/components/ui/badge";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import {
  ORDER_CHARGES_GROUP_KEY,
  groupInvoiceLinesByShipment,
  type InvoiceLineGroup,
} from "@trenova/shared/lib/invoice-lines";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type { Invoice, InvoiceLine, InvoiceLineType } from "@trenova/shared/types/invoice";
import { ChevronDownIcon, ChevronRightIcon } from "lucide-react";
import { useMemo, useState } from "react";

const LINE_TYPE_VARIANTS: Record<InvoiceLineType, BadgeVariant> = {
  Freight: "info",
  Accessorial: "purple",
};

/**
 * Beyond this many shipments the table is long enough that opening every group
 * buries the totals, so all but the first start collapsed.
 */
const COLLAPSE_THRESHOLD = 8;

const COLUMN_COUNT = 6;

function groupHeading(group: InvoiceLineGroup): string {
  if (group.key === ORDER_CHARGES_GROUP_KEY) return "Order charges";
  return group.proNumber ?? group.shipmentId ?? "Unattributed";
}

export function InvoiceChargesTab({ invoice }: { invoice: Invoice }) {
  const groups = useMemo(() => groupInvoiceLinesByShipment(invoice.lines ?? []), [invoice.lines]);

  const [collapsed, setCollapsed] = useState<ReadonlySet<string>>(() =>
    groups.length > COLLAPSE_THRESHOLD
      ? new Set(groups.slice(1).map((group) => group.key))
      : new Set<string>(),
  );

  const isGrouped = groups.length > 1;

  function toggle(key: string) {
    setCollapsed((previous) => {
      const next = new Set(previous);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      return next;
    });
  }

  return (
    <ScrollArea className="h-full">
      <div className="p-4">
        <div className="border-border overflow-hidden rounded-xl border">
          <table className="w-full text-sm">
            <thead className="bg-muted/50 text-muted-foreground text-left">
              <tr>
                <th className="px-4 py-3 font-medium">Line</th>
                <th className="px-4 py-3 font-medium">Description</th>
                <th className="px-4 py-3 font-medium">Type</th>
                <th className="px-4 py-3 text-right font-medium">Quantity</th>
                <th className="px-4 py-3 text-right font-medium">Unit Price</th>
                <th className="px-4 py-3 text-right font-medium">Amount</th>
              </tr>
            </thead>

            {isGrouped ? (
              groups.map((group) => (
                <ShipmentGroupBody
                  key={group.key}
                  group={group}
                  currencyCode={invoice.currencyCode}
                  isCollapsed={collapsed.has(group.key)}
                  onToggle={() => toggle(group.key)}
                />
              ))
            ) : (
              <tbody>
                {(groups[0]?.lines ?? []).map((line) => (
                  <ChargeRow key={line.id} line={line} currencyCode={invoice.currencyCode} />
                ))}
              </tbody>
            )}

            <tfoot className="bg-muted/30 border-t">
              <tr>
                <td
                  colSpan={COLUMN_COUNT - 1}
                  className="text-muted-foreground px-4 py-2.5 text-right text-sm"
                >
                  Subtotal
                </td>
                <td className="px-4 py-2.5 text-right text-sm tabular-nums">
                  {formatCurrency(Number(invoice.subtotalAmount ?? 0), invoice.currencyCode)}
                </td>
              </tr>
              <tr>
                <td
                  colSpan={COLUMN_COUNT - 1}
                  className="text-muted-foreground px-4 py-2.5 text-right text-sm"
                >
                  Other Charges
                </td>
                <td className="px-4 py-2.5 text-right text-sm tabular-nums">
                  {formatCurrency(Number(invoice.otherAmount ?? 0), invoice.currencyCode)}
                </td>
              </tr>
              <tr className="border-t">
                <td
                  colSpan={COLUMN_COUNT - 1}
                  className="px-4 py-3 text-right text-sm font-semibold"
                >
                  Total
                </td>
                <td className="px-4 py-3 text-right text-sm font-bold tabular-nums">
                  {formatCurrency(Number(invoice.totalAmount ?? 0), invoice.currencyCode)}
                </td>
              </tr>
            </tfoot>
          </table>
        </div>
      </div>
    </ScrollArea>
  );
}

function ShipmentGroupBody({
  group,
  currencyCode,
  isCollapsed,
  onToggle,
}: {
  group: InvoiceLineGroup;
  currencyCode: string;
  isCollapsed: boolean;
  onToggle: () => void;
}) {
  const heading = groupHeading(group);
  const lineLabel = group.lines.length === 1 ? "1 line" : `${group.lines.length} lines`;

  return (
    <tbody className="border-t">
      <tr className="bg-muted/30">
        <td colSpan={COLUMN_COUNT} className="px-4 py-2">
          <button
            type="button"
            onClick={onToggle}
            aria-expanded={!isCollapsed}
            className="hover:text-foreground text-muted-foreground flex w-full items-center gap-2 text-left transition-colors"
          >
            {isCollapsed ? (
              <ChevronRightIcon className="size-3.5 shrink-0" />
            ) : (
              <ChevronDownIcon className="size-3.5 shrink-0" />
            )}
            <span className="text-foreground font-mono text-xs font-medium">{heading}</span>
            {group.bol ? <span className="text-xs">BOL {group.bol}</span> : null}
            <span className="text-xs">·</span>
            <span className="text-xs">{lineLabel}</span>
          </button>
        </td>
      </tr>

      {isCollapsed
        ? null
        : group.lines.map((line) => (
            <ChargeRow key={line.id} line={line} currencyCode={currencyCode} />
          ))}

      <tr className="border-t">
        <td
          colSpan={COLUMN_COUNT - 1}
          className="text-muted-foreground px-4 py-2 text-right text-xs"
        >
          {heading} subtotal
        </td>
        <td className="px-4 py-2 text-right text-xs font-medium tabular-nums">
          {formatCurrency(group.subtotal, currencyCode)}
        </td>
      </tr>
    </tbody>
  );
}

function ChargeRow({ line, currencyCode }: { line: InvoiceLine; currencyCode: string }) {
  return (
    <tr className="hover:bg-muted/50 border-t transition-colors">
      <td className="px-4 py-3 font-mono text-xs">{line.lineNumber}</td>
      <td className="px-4 py-3">{line.description}</td>
      <td className="px-4 py-3">
        <Badge variant={LINE_TYPE_VARIANTS[line.type]}>{line.type}</Badge>
      </td>
      <td className="px-4 py-3 text-right tabular-nums">{Number(line.quantity ?? 0)}</td>
      <td className="px-4 py-3 text-right tabular-nums">
        {formatCurrency(Number(line.unitPrice ?? 0), currencyCode)}
      </td>
      <td className="px-4 py-3 text-right font-medium tabular-nums">
        {formatCurrency(Number(line.amount ?? 0), currencyCode)}
      </td>
    </tr>
  );
}
