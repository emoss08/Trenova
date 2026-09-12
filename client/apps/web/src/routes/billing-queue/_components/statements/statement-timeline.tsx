import { useT } from "@trenova/shared/i18n/use-t";
import { billsInLabel, periodRange, splitLabel } from "@/lib/billing-schedule";
import { cn } from "@trenova/shared/lib/utils";
import type { OpenStatement } from "@trenova/shared/types/statement";
import { BotIcon, CheckIcon, ReceiptTextIcon, UserCheckIcon } from "lucide-react";

type StepState = "done" | "active" | "pending";

/**
 * What happens to this freight, in the order it happens.
 *
 * The reason this earns space is that statement billing is the one part of the
 * pipeline where nothing visibly happens for weeks. A biller looking at a
 * customer with thirty shipments and no invoice needs to see that this is the
 * system working, not the system stuck — and exactly when it stops waiting.
 */
export function StatementTimeline({
  statement,
  nowSeconds,
}: {
  statement: OpenStatement;
  nowSeconds: number;
}) {
  const t = useT();

  const due = statement.periodEnd <= nowSeconds;
  const hasFreight = statement.shipmentCount > 0;

  const steps: { key: string; label: string; detail: string; state: StepState; Icon: typeof CheckIcon }[] = [
    {
      key: "accrue",
      label: "Accruing",
      detail: hasFreight
        ? `${statement.shipmentCount} shipment${statement.shipmentCount === 1 ? "" : "s"} · ${periodRange(statement.periodStart, statement.periodEnd)}`
        : `Nothing yet · ${periodRange(statement.periodStart, statement.periodEnd)}`,
      state: due ? "done" : "active",
      Icon: CheckIcon,
    },
    {
      key: "bill",
      label: due ? "Billing" : "Bills",
      detail: billsInLabel(statement.periodEnd, nowSeconds),
      state: due ? "active" : "pending",
      Icon: statement.autoBill ? BotIcon : UserCheckIcon,
    },
    {
      key: "invoice",
      label: statement.invoiceCount === 1 ? "One invoice" : `${statement.invoiceCount} invoices`,
      detail: hasFreight
        ? `${splitLabel(statement.splitBy)} · ${statement.detail === "Summary" ? "summary" : "itemised"}`
        : "Nothing to bill",
      state: "pending",
      Icon: ReceiptTextIcon,
    },
  ];

  return (
    <ol className="flex items-stretch gap-1 px-3 py-2">
      {steps.map((step, index) => (
        <li key={step.key} className="flex min-w-0 flex-1 items-center gap-2">
          <span
            aria-hidden
            className={cn(
              "flex size-5 shrink-0 items-center justify-center rounded-full border",
              step.state === "done" && "border-brand bg-brand text-white",
              step.state === "active" && "border-brand text-brand",
              step.state === "pending" && "text-muted-foreground border-dashed",
            )}
          >
            <step.Icon className="size-3" />
          </span>
          <span className="min-w-0">
            <span
              className={cn(
                "block truncate text-xs font-medium",
                step.state === "pending" && "text-muted-foreground",
              )}
            >
              {t(step.label)}
            </span>
            <span className="text-muted-foreground block truncate text-[11px]">{step.detail}</span>
          </span>
          {index < steps.length - 1 && (
            <span
              aria-hidden
              className={cn(
                "ml-auto hidden h-px flex-1 sm:block",
                steps[index + 1].state === "pending" ? "bg-border" : "bg-brand/40",
              )}
            />
          )}
        </li>
      ))}
    </ol>
  );
}
