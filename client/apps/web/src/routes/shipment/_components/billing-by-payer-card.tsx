import { useT } from "@trenova/shared/i18n/use-t";
import { summarizeBillingByPayer } from "@trenova/shared/lib/charge-split";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type { Shipment } from "@trenova/shared/types/shipment";
import { useFormContext, useWatch } from "react-hook-form";

/**
 * The invoices a save would produce: one row per payer with their share of
 * freight and accessorials. Nothing is shown for an unsplit shipment billed to
 * its own customer, which is the ordinary case and needs no explaining.
 */
export function BillingByPayerCard() {
  const t = useT();
  const { control } = useFormContext<Shipment>();
  const customerId = useWatch({ control, name: "customerId" });
  const billToCustomerId = useWatch({ control, name: "billToCustomerId" });
  const freightChargeAmount = useWatch({ control, name: "freightChargeAmount" });
  const freightAllocations = useWatch({ control, name: "freightAllocations" });
  const additionalCharges = useWatch({ control, name: "additionalCharges" });
  const customer = useWatch({ control, name: "customer" });
  const billToCustomer = useWatch({ control, name: "billToCustomer" });

  if (!customerId) return null;

  const labels = new Map<string, { name: string; code: string }>();
  if (customer?.id) labels.set(customer.id, { name: customer.name, code: customer.code ?? "" });
  if (billToCustomer?.id) {
    labels.set(billToCustomer.id, { name: billToCustomer.name, code: billToCustomer.code ?? "" });
  }
  for (const row of freightAllocations ?? []) {
    if (row?.billToCustomer?.id) {
      labels.set(row.billToCustomer.id, {
        name: row.billToCustomer.name,
        code: row.billToCustomer.code ?? "",
      });
    }
  }
  for (const charge of additionalCharges ?? []) {
    for (const row of charge?.allocations ?? []) {
      if (row?.billToCustomer?.id) {
        labels.set(row.billToCustomer.id, {
          name: row.billToCustomer.name,
          code: row.billToCustomer.code ?? "",
        });
      }
    }
  }

  const rows = summarizeBillingByPayer(
    {
      customerId,
      billToCustomerId,
      freightChargeAmount,
      freightAllocations: freightAllocations ?? [],
      additionalCharges: additionalCharges ?? [],
    },
    (id) => labels.get(id) ?? null,
  );

  const split = rows.some((row) => row.isSplit) || rows.length > 1;
  const redirected = rows.length === 1 && rows[0].payerId !== customerId;
  if (!split && !redirected) return null;

  return (
    <div className="bg-muted/50 mt-3 rounded-lg border p-2" data-testid="billing-by-payer">
      <div className="mb-2">
        <span className="text-xs font-medium">{t("Billing by payer")}</span>
        <p className="text-2xs text-muted-foreground mt-0.5">
          {t("One invoice is issued to each payer for their share of the charges.")}
        </p>
      </div>
      <div className="divide-y">
        {rows.map((row) => {
          const label = row.payerCode
            ? `${row.payerCode} – ${row.payerName}`
            : row.payerName || row.payerId;
          return (
            <div key={row.payerId} className="flex items-center justify-between gap-3 py-1.5">
              <div className="flex min-w-0 items-center gap-1.5">
                <span className="truncate text-xs font-medium">{label}</span>
                {row.isPrimary ? (
                  <span className="bg-primary/10 text-2xs text-primary rounded-md px-1 py-0.5">
                    {t("Primary")}
                  </span>
                ) : null}
              </div>
              <div className="text-muted-foreground flex shrink-0 items-center gap-3 text-xs tabular-nums">
                <span>{t("Freight {0}", formatCurrency(Number(row.freightAmount ?? 0)))}</span>
                <span>
                  {t("Accessorials {0}", formatCurrency(Number(row.accessorialAmount ?? 0)))}
                </span>
                <span className={cn("text-foreground font-semibold")}>
                  {formatCurrency(Number(row.totalAmount ?? 0))}
                </span>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
