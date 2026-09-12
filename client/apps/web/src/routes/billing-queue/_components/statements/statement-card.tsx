import { BillingRecordCard } from "@/components/billing/billing-record-card";
import { billsInLabel, cadenceLabel } from "@/lib/billing-schedule";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type { OpenStatement } from "@trenova/shared/types/statement";
import { BotIcon, PauseCircleIcon } from "lucide-react";

/**
 * Bar showing how far into the period the statement is.
 *
 * A statement's most-asked question is "how much longer", and a date alone does
 * not answer it at a glance the way a bar filling up does.
 */
function PeriodProgress({ statement, nowSeconds }: { statement: OpenStatement; nowSeconds: number }) {
  const span = statement.periodEnd - statement.periodStart;
  const elapsed = nowSeconds - statement.periodStart;
  const pct = span > 0 ? Math.min(100, Math.max(0, (elapsed / span) * 100)) : 100;
  const due = statement.periodEnd <= nowSeconds;

  return (
    <div className="bg-muted h-0.5 w-full overflow-hidden rounded-full" aria-hidden>
      <div
        className={cn("h-full rounded-full transition-[width]", due ? "bg-amber-500" : "bg-brand")}
        style={{ width: `${pct}%` }}
      />
    </div>
  );
}

export function StatementCard({
  statement,
  nowSeconds,
  isSelected,
  onClick,
}: {
  statement: OpenStatement;
  nowSeconds: number;
  isSelected: boolean;
  onClick: () => void;
}) {
  const empty = statement.shipmentCount === 0;
  const due = statement.periodEnd <= nowSeconds;

  return (
    <div className="flex flex-col">
      <BillingRecordCard
        title={statement.customerName}
        auxiliary={
          statement.customerCode ? (
            <span className="text-muted-foreground font-mono text-[10px]">
              {statement.customerCode}
            </span>
          ) : null
        }
        amount={
          empty ? (
            <span className="text-muted-foreground text-xs font-normal">Nothing yet</span>
          ) : (
            formatCurrency(Number(statement.totalAmount ?? 0), statement.currencyCode)
          )
        }
        subtitle={
          empty
            ? `${cadenceLabel(statement.cycle)} · no shipments this period`
            : `${statement.shipmentCount} shipment${statement.shipmentCount === 1 ? "" : "s"} · ` +
              `${statement.invoiceCount} invoice${statement.invoiceCount === 1 ? "" : "s"} when it bills`
        }
        meta={
          <div className="flex flex-col gap-1.5">
            <PeriodProgress statement={statement} nowSeconds={nowSeconds} />
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-1.5">
                <span className="text-muted-foreground text-[11px]">
                  {cadenceLabel(statement.cycle)}
                </span>
                {statement.autoBill && (
                  <Tooltip>
                    <TooltipTrigger
                      render={
                        <span
                          tabIndex={0}
                          className="text-muted-foreground/70 inline-flex"
                          aria-label="Bills automatically"
                        >
                          <BotIcon className="size-3" />
                        </span>
                      }
                    />
                    <TooltipContent side="right">
                      Bills automatically when the period closes — no review step
                    </TooltipContent>
                  </Tooltip>
                )}
                {statement.belowMinimum && !empty && (
                  <Tooltip>
                    <TooltipTrigger
                      render={
                        <span
                          tabIndex={0}
                          className="inline-flex text-amber-600 dark:text-amber-400"
                          aria-label="Under the customer's minimum"
                        >
                          <PauseCircleIcon className="size-3" />
                        </span>
                      }
                    />
                    <TooltipContent side="right">
                      Under this customer&apos;s invoice minimum — it will roll into next period
                      instead of billing
                    </TooltipContent>
                  </Tooltip>
                )}
              </div>
              <span
                className={cn(
                  "text-[11px]",
                  due ? "font-medium text-amber-600 dark:text-amber-400" : "text-muted-foreground/70",
                )}
              >
                {billsInLabel(statement.periodEnd, nowSeconds)}
              </span>
            </div>
          </div>
        }
        isSelected={isSelected}
        onClick={onClick}
      />
    </div>
  );
}
