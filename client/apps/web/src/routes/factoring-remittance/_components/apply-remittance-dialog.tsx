import { FactoringMovementPreview } from "@/components/factoring/factoring-movement-preview";
import { FACTORING_REMITTANCE_LINE_KIND_LABELS } from "@/lib/factoring-remittance";
import {
  applyFactoringRemittance,
  type FactoringRemittance,
} from "@/lib/graphql/factoring-remittance";
import { factoringRemittance as factoringRemittanceQueries } from "@/lib/queries/factoring-remittance";
import { invalidateFactoringQueries } from "@/lib/queries/factoring-roots";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { describeError } from "@trenova/shared/lib/error-presentation";
import { formatMinor } from "@trenova/shared/lib/money";
import { toast } from "sonner";

/**
 * Shows the journal every matched line would book, then books them all in one
 * go. What is still to resolve stays open on the remittance.
 */
export function ApplyRemittanceDialog({
  remittance,
  open,
  onOpenChange,
}: {
  remittance: FactoringRemittance;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const input = { id: remittance.id, version: remittance.version };
  const plan = useQuery({ ...factoringRemittanceQueries.plan(input), enabled: open });
  const money = (minor: number) => formatMinor(minor, remittance.currencyCode);

  const apply = useMutation({
    mutationFn: () => applyFactoringRemittance(input),
    onSuccess: (applied) => {
      invalidateFactoringQueries(queryClient);
      toast.success(t("Remittance applied"), {
        description: t(
          "{0, plural, one {# line booked.} other {# lines booked.}}",
          applied.appliedCount,
        ),
      });
      onOpenChange(false);
    },
    onError: (error) => {
      invalidateFactoringQueries(queryClient);
      const described = describeError(error);
      toast.error(t("The remittance was not applied"), {
        description: described.detail ?? described.message,
      });
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="xl" className="max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{t("Apply remittance")}</DialogTitle>
          <DialogDescription>
            {t(
              "Every matched line is booked to the ledger through the factor's schedules. Only an opposite entry undoes a booking.",
            )}
          </DialogDescription>
        </DialogHeader>
        {plan.isLoading ? (
          <div className="flex flex-col gap-2" aria-busy="true">
            <Skeleton className="h-16 w-full" />
            <Skeleton className="h-40 w-full" />
          </div>
        ) : plan.error ? (
          <Alert size="sm" variant="destructive">
            <AlertDescription>
              {describeError(plan.error).detail ?? t("What it would book could not be worked out.")}
            </AlertDescription>
          </Alert>
        ) : plan.data ? (
          <div className="flex flex-col gap-4">
            <p className="text-sm">
              {t(
                "{0, plural, one {# invoice funded} other {# invoices funded}} (advance {1}, fee {2}), {3, plural, one {# rejected} other {# rejected}}, {4, plural, one {# reserve released} other {# reserves released}} ({5}) and {6, plural, one {# charged back} other {# charged back}} ({7}).",
                plan.data.fundedCount,
                money(plan.data.advanceMinor),
                money(plan.data.feeMinor),
                plan.data.rejectedCount,
                plan.data.releasedCount,
                money(plan.data.releasedMinor),
                plan.data.chargebackCount,
                money(plan.data.chargebackMinor),
              )}
            </p>
            {plan.data.leftOpenCount > 0 ? (
              <Alert size="sm" variant="warning">
                <AlertDescription>
                  {t(
                    "{0, plural, one {# line is} other {# lines are}} still to resolve and stay open.",
                    plan.data.leftOpenCount,
                  )}
                </AlertDescription>
              </Alert>
            ) : null}
            <ol className="flex flex-col gap-3">
              {plan.data.lines.map((line) => (
                <li key={line.lineId} className="rounded-lg border p-3">
                  <p className="mb-2 text-sm font-medium">
                    {t(
                      "Row {0}: {1} of invoice {2}",
                      line.rowNumber,
                      FACTORING_REMITTANCE_LINE_KIND_LABELS[line.kind],
                      line.invoiceNumber,
                    )}
                  </p>
                  {line.movement ? (
                    <FactoringMovementPreview
                      movement={line.movement}
                      currencyCode={remittance.currencyCode}
                    />
                  ) : (
                    <p className="text-muted-foreground text-sm">
                      {t("The invoice goes back to the factoring queue; nothing is booked.")}
                    </p>
                  )}
                </li>
              ))}
            </ol>
          </div>
        ) : null}
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Back")}
          </Button>
          <Button
            type="button"
            disabled={!plan.data || plan.data.lines.length === 0}
            isLoading={apply.isPending}
            onClick={() => apply.mutate()}
          >
            {t("Apply")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
