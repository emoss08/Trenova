import { useT } from "@trenova/shared/i18n/use-t";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { formatCurrency } from "@trenova/shared/lib/utils";
import { SplitIcon } from "lucide-react";

/**
 * The least an allocation row has to carry to be named. Shipment charges hold
 * numbers after zod parsing; order charges come off GraphQL as decimal strings.
 */
export type PayerChipAllocation = {
  id?: string | null;
  billToCustomerId: string;
  method: string;
  percent?: number | string | null;
  amount?: number | string | null;
  billToCustomer?: { id: string; name: string; code?: string | null } | null;
};

export function allocationPayerLabel(row: PayerChipAllocation): string {
  const snapshot = row.billToCustomer;
  if (!snapshot) return row.billToCustomerId;
  return snapshot.code ? `${snapshot.code} – ${snapshot.name}` : snapshot.name;
}

/**
 * Who pays this charge when it is not the shipment's own payer: a single named
 * customer, or a split across several. Nothing is shown in the ordinary case.
 */
export function ChargePayerChip({
  allocations,
  currencyCode = "USD",
}: {
  allocations: readonly PayerChipAllocation[] | null | undefined;
  currencyCode?: string;
}) {
  const t = useT();
  const rows = (allocations ?? []).filter((row) => row?.billToCustomerId);
  if (rows.length === 0) return null;

  if (rows.length === 1) {
    return (
      <span
        className="text-2xs shrink-0 rounded-md bg-accent-indigo/10 px-1 py-0.5 text-accent-indigo-on-subtle"
        data-testid="charge-payer-chip"
      >
        {t("Bill to {0}", allocationPayerLabel(rows[0]))}
      </span>
    );
  }

  return (
    <Tooltip>
      <TooltipTrigger>
        <span
          className="text-2xs flex shrink-0 items-center gap-1 rounded-md bg-accent-indigo/10 px-1 py-0.5 text-accent-indigo-on-subtle"
          data-testid="charge-payer-chip"
        >
          <SplitIcon className="size-2.5" />
          {t("Split {0} ways", rows.length)}
        </span>
      </TooltipTrigger>
      <TooltipContent side="top" sideOffset={6}>
        <div className="space-y-0.5">
          {rows.map((row, idx) => (
            <p key={row.id ?? idx} className="text-xs tabular-nums">
              {allocationPayerLabel(row)}
              {" · "}
              {row.method === "Amount"
                ? formatCurrency(Number(row.amount ?? 0), currencyCode)
                : `${Number(row.percent ?? 0)}%`}
            </p>
          ))}
        </div>
      </TooltipContent>
    </Tooltip>
  );
}
