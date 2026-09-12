import { useT } from "@trenova/shared/i18n/use-t";
import { periodRange } from "@/lib/billing-schedule";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type { OpenStatement, StatementGroup } from "@trenova/shared/types/statement";
import { CalendarClockIcon, ClockIcon, TriangleAlertIcon } from "lucide-react";
import { useState } from "react";

export type BillStatementDecision = { reason: string };

/**
 * The last thing a biller sees before invoices exist.
 *
 * It states what will be created rather than asking "are you sure": the counts
 * and the total are the confirmation. The reason box only appears when the
 * period has not closed, because that is the only case where the biller is
 * departing from what the customer agreed to — asking for a reason on the
 * ordinary path would train them to type nothing into it.
 */
export function BillStatementDialog({
  open,
  statement,
  groups,
  heldCount,
  total,
  offCycle,
  pending,
  onOpenChange,
  onConfirm,
}: {
  open: boolean;
  statement: OpenStatement;
  groups: readonly StatementGroup[];
  heldCount: number;
  total: number;
  offCycle: boolean;
  pending: boolean;
  onOpenChange: (next: boolean) => void;
  onConfirm: (decision: BillStatementDecision) => void;
}) {
  const t = useT();

  const [reason, setReason] = useState("");

  const billable = groups.filter((group) => !group.belowMinimum);
  const held = groups.length - billable.length;
  const reasonMissing = offCycle && reason.trim().length === 0;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>
            {t("Bill {0, plural, one {# invoice} other {# invoices}} for {1}", billable.length, statement.customerName)}
          </DialogTitle>
          <DialogDescription>
            {t("Covers {0}. Each invoice is created as a draft — it still has to be posted and sent.", periodRange(statement.periodStart, statement.periodEnd))}
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-2">
          <div className="bg-muted/40 flex flex-col gap-1 rounded-lg border p-3 text-xs">
            {billable.map((group) => (
              <div key={group.key} className="flex items-center justify-between gap-3">
                <span className="truncate">{t(group.label)}</span>
                <span className="text-muted-foreground shrink-0 tabular-nums">
                  {t("{0} shp", group.shipmentCount)}
                </span>
              </div>
            ))}
            <div className="mt-1 flex items-center justify-between gap-3 border-t pt-2 text-sm font-semibold">
              <span>{t("Total")}</span>
              <span className="tabular-nums">
                {formatCurrency(total, statement.currencyCode)}
              </span>
            </div>
          </div>

          {heldCount > 0 && (
            <p className="text-muted-foreground text-xs">
              {t("{0, plural, one {# shipment} other {# shipments}} you held back stay approved and uninvoiced, and land on next period's statement.", heldCount)}
            </p>
          )}

          {statement.heldCount > 0 && (
            <div className="flex items-start gap-2 rounded-lg border border-amber-300 bg-amber-50/60 p-3 dark:border-amber-900 dark:bg-amber-950/30">
              <ClockIcon className="mt-0.5 size-3.5 shrink-0 text-amber-600 dark:text-amber-400" />
              <p className="text-xs text-amber-800 dark:text-amber-200">
                {statement.heldCount} more shipment{statement.heldCount === 1 ? "" : "s"} for this
                period {statement.heldCount === 1 ? "is" : "are"} still in review and will not be
                on this invoice. Billing now leaves{" "}
                {formatCurrency(Number(statement.heldAmount ?? 0), statement.currencyCode)}{" "}
                unbilled.
              </p>
            </div>
          )}

          {held > 0 && (
            <p className="text-muted-foreground text-xs">
              {t("{0, plural, one {# invoice} other {# invoices}} under the customer's minimum {1} skipped, and {2} shipments roll into next period.", held, held === 1 ? "is" : "are", held === 1 ? "its" : "their")}
            </p>
          )}

          {offCycle ? (
            <div className="flex flex-col gap-2 rounded-lg border border-amber-300 bg-amber-50/60 p-3 dark:border-amber-900 dark:bg-amber-950/30">
              <div className="flex items-start gap-2">
                <TriangleAlertIcon className="mt-0.5 size-3.5 shrink-0 text-amber-600 dark:text-amber-400" />
                <p className="text-xs text-amber-800 dark:text-amber-200">
                  {t("This period has not closed yet. Billing now does not move {0}'s cycle — anything delivered for the rest of the period still bills on the original date.", statement.customerName)}
                </p>
              </div>
              <Textarea
                value={reason}
                onChange={(event) => setReason(event.target.value)}
                placeholder={t("Why is this being billed early? (e.g. customer is closing their books)")}
                rows={2}
                className="text-xs"
                aria-label={t("Reason for billing before the cycle closes")}
              />
            </div>
          ) : (
            <div className="text-muted-foreground flex items-start gap-2 text-xs">
              <CalendarClockIcon className="mt-0.5 size-3.5 shrink-0" />
              <span>{t("This period has closed, so this is the invoice the customer expects.")}</span>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button
            disabled={reasonMissing || billable.length === 0}
            isLoading={pending}
            loadingText={t("Billing...")}
            onClick={() => onConfirm({ reason: reason.trim() })}
            title={
              reasonMissing
                ? "Say why this is billing before the cycle closes"
                : billable.length === 0
                  ? "Every invoice on this statement is under the customer's minimum"
                  : undefined
            }
          >
            {t("Bill {0, plural, one {# invoice} other {# invoices}}", billable.length)}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
