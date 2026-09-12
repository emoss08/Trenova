import { useT } from "@trenova/shared/i18n/use-t";
import {
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { formatMinor } from "@trenova/shared/lib/benefits";
import { formatUnixDate, getTodayDate } from "@trenova/shared/lib/date";
import { apiService } from "@/services/api";
import type {
  FiscalYearClosePlan,
  FiscalYearClosePlanEntry,
  FiscalYearSubledgerCheck,
} from "@/types/fiscal-year";
import type { FiscalYearRow } from "@/lib/graphql/fiscal-year-table";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertTriangleIcon } from "lucide-react";
import { useCallback, useState } from "react";
import { toast } from "sonner";

export type FiscalYearAction = "activate" | "close" | "reopen";

type DialogProps = {
  record: FiscalYearRow;
  onClose: () => void;
};

function useFiscalYearInvalidation() {
  const queryClient = useQueryClient();

  return useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: ["fiscal-year-list"] });
    void queryClient.invalidateQueries({ queryKey: ["fiscal-year-close-preview"] });
  }, [queryClient]);
}

export function FiscalYearActivateAlertDialogContent({ record, onClose }: DialogProps) {
  const t = useT();

  const invalidate = useFiscalYearInvalidation();

  const { mutateAsync, isPending } = useApiMutation({
    mutationFn: async (id: FiscalYearRow["id"]) => apiService.fiscalYearService.activate(id),
    onSuccess: () => {
      toast.success(t("Activated successfully"), {
        description: `Successfully set ${record?.year} as current`,
      });
      invalidate();
      onClose();
    },
  });

  const handleFiscalYearActivate = useCallback(() => {
    void mutateAsync(record?.id);
  }, [mutateAsync, record?.id]);

  return (
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{t("Set Fiscal Year {0} as Current?", record?.year)}</AlertDialogTitle>
        <div className="text-muted-foreground flex flex-col space-y-2 text-sm">
          <p>{t("This will mark this fiscal year as the active year for transaction posting.")}</p>
          <p>{t("Any currently active fiscal year will be automatically deactivated.")}</p>
        </div>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{t("Cancel")}</AlertDialogCancel>
        <AlertDialogAction disabled={isPending} onClick={handleFiscalYearActivate}>
          {t("Set as Current")}
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  );
}

export function FiscalYearCloseAlertDialogContent({ record, onClose }: DialogProps) {
  const t = useT();

  const invalidate = useFiscalYearInvalidation();
  const today = getTodayDate();

  const {
    data: plan,
    isLoading,
    isError,
  } = useQuery({
    queryKey: ["fiscal-year-close-preview", record?.id],
    queryFn: () => apiService.fiscalYearService.closePreview(record.id),
    enabled: !!record?.id,
    staleTime: 0,
  });

  const { mutateAsync, isPending } = useApiMutation({
    mutationFn: async (id: FiscalYearRow["id"]) => apiService.fiscalYearService.close(id),
    onSuccess: () => {
      toast.success(t("Closed successfully"), {
        description: `Successfully closed ${record?.year}`,
      });
      invalidate();
      onClose();
    },
  });

  const handleFiscalYearClose = useCallback(() => {
    void mutateAsync(record?.id);
  }, [mutateAsync, record?.id]);

  const blocked = !plan?.canClose;

  return (
    <AlertDialogContent className="min-w-lg">
      <AlertDialogHeader>
        <AlertDialogTitle>{t("Close Fiscal Year {0}?", record?.year)}</AlertDialogTitle>
        {record?.endDate && record.endDate > today && (
          <Alert variant="warning">
            <AlertTriangleIcon />
            <AlertTitle>{t("Early close")}</AlertTitle>
            <AlertDescription>
              <p>
                {t("This fiscal year does not end until {0} ( {1} days remaining). Closing early prevents posting transactions for the remainder of the year.", formatUnixDate(record.endDate), Math.ceil((record.endDate - today) / 86400))}
              </p>
            </AlertDescription>
          </Alert>
        )}
        {isLoading && <ClosePreviewSkeleton />}
        {isError && (
          <Alert variant="destructive">
            <AlertTriangleIcon />
            <AlertTitle>{t("Close preview unavailable")}</AlertTitle>
            <AlertDescription>
              <p>{t("The year-end entries could not be calculated. Try again before closing.")}</p>
            </AlertDescription>
          </Alert>
        )}
        {plan && <ClosePreview plan={plan} />}
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{t("Cancel")}</AlertDialogCancel>
        <AlertDialogAction
          variant="destructive"
          disabled={isLoading || isPending || blocked}
          onClick={handleFiscalYearClose}
        >
          {t("Post and Close Year")}
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  );
}

export function FiscalYearReopenAlertDialogContent({ record, onClose }: DialogProps) {
  const t = useT();

  const invalidate = useFiscalYearInvalidation();
  const [reason, setReason] = useState("");

  const { mutateAsync, isPending } = useApiMutation({
    mutationFn: async (reopenReason: string) =>
      apiService.fiscalYearService.reopen(record.id, reopenReason),
    onSuccess: () => {
      toast.success(t("Reopened successfully"), {
        description: `${record?.year} is open again and its closing entries were reversed`,
      });
      invalidate();
      onClose();
    },
  });

  const handleFiscalYearReopen = useCallback(() => {
    void mutateAsync(reason.trim());
  }, [mutateAsync, reason]);

  return (
    <AlertDialogContent className="min-w-lg">
      <AlertDialogHeader>
        <AlertDialogTitle>{t("Reopen Fiscal Year {0}?", record?.year)}</AlertDialogTitle>
        <Alert variant="warning">
          <AlertTriangleIcon />
          <AlertTitle>{t("The close will be reversed")}</AlertTitle>
          <AlertDescription>
            <p>
              {t("Reversing entries are posted against the closing and opening entries this year produced. The originals stay on the ledger, so the audit trail shows both the close and its undo. The year has to be closed again afterwards.")}
            </p>
          </AlertDescription>
        </Alert>
        <div className="flex flex-col gap-1.5">
          <label className="text-sm font-medium" htmlFor="fiscal-year-reopen-reason">
            {t("Reason")}
          </label>
          <Textarea
            id="fiscal-year-reopen-reason"
            value={reason}
            onChange={(event) => setReason(event.target.value)}
            placeholder={t("Why is this year being reopened?")}
            rows={3}
          />
          <p className="text-muted-foreground text-xs">
            {t("Recorded on the fiscal year and on every reversing entry.")}
          </p>
        </div>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{t("Cancel")}</AlertDialogCancel>
        <AlertDialogAction
          variant="destructive"
          disabled={isPending || reason.trim().length === 0}
          onClick={handleFiscalYearReopen}
        >
          {t("Reverse and Reopen")}
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  );
}

function ClosePreviewSkeleton() {
  return (
    <div className="flex flex-col gap-2">
      <Skeleton className="h-16 w-full" />
      <Skeleton className="h-24 w-full" />
    </div>
  );
}

function ClosePreview({ plan }: { plan: FiscalYearClosePlan }) {
  const t = useT();

  if (plan.blockers.length > 0) {
    return (
      <Alert variant="destructive">
        <AlertTriangleIcon />
        <AlertTitle>
          {plan.blockers.length === 1
            ? t("1 issue blocks this close")
            : t("{0} issues block this close", plan.blockers.length)}
        </AlertTitle>
        <AlertDescription>
          <ul className="list-inside list-disc">
            {plan.blockers.map((blocker) => (
              <li key={`${blocker.field}-${blocker.message}`}>{blocker.message}</li>
            ))}
          </ul>
        </AlertDescription>
      </Alert>
    );
  }

  return (
    <div className="flex flex-col gap-3 text-left">
      <div className="grid grid-cols-2 gap-2 text-sm">
        <Figure label={t("Revenue")} value={formatMinor(plan.revenueMinor)} />
        <Figure label={t("Cost of revenue")} value={formatMinor(plan.costOfRevenueMinor)} />
        <Figure label={t("Operating expense")} value={formatMinor(plan.operatingExpenseMinor)} />
        <Figure
          label={plan.netIncomeMinor < 0 ? "Net loss" : "Net income"}
          value={formatMinor(Math.abs(plan.netIncomeMinor))}
          emphasis
        />
      </div>
      <p className="text-muted-foreground text-sm">
        {plan.netIncomeMinor < 0 ? `${t("Debited to")} ` : `${t("Credited to")} `}
        <span className="font-medium">
          {plan.retainedEarningsAccountCode} {plan.retainedEarningsAccountName}
        </span>
        {plan.nextFiscalYearName
          ? t(", with closing balances carried forward into {0}.", plan.nextFiscalYearName)
          : "."}
      </p>
      <div className="flex flex-col gap-2">
        <EntrySummary entry={plan.closingEntry} fallback="No income-statement activity to close." />
        <EntrySummary entry={plan.openingEntry} fallback="No balances to carry forward." />
      </div>
      <SubledgerChecks checks={plan.subledgerChecks} />
    </div>
  );
}

/**
 * The general ledger carries one total per control account; the detail behind it
 * lives in the subledger. This is the close proving the two still agree.
 */
function SubledgerChecks({ checks }: { checks: FiscalYearSubledgerCheck[] }) {
  const t = useT();

  if (checks.length === 0) return null;

  return (
    <div className="flex flex-col gap-1">
      {checks.map((check) => (
        <div
          key={check.key}
          className="flex items-baseline justify-between gap-2 text-xs"
          data-slot="subledger-check"
        >
          <span className="text-muted-foreground">
            {t(check.label)} ({check.accountCode})
          </span>
          {check.reconciled ? (
            <span className="font-mono">{t("reconciled · {0}", formatMinor(check.glBalanceMinor))}</span>
          ) : (
            <span className="text-destructive font-mono">
              {t("off by {0}{1}", formatMinor(check.differenceMinor), check.enforced ? "" : ` ${t("(not enforced)")}`)}
            </span>
          )}
        </div>
      ))}
    </div>
  );
}

function Figure({ label, value, emphasis }: { label: string; value: string; emphasis?: boolean }) {
  return (
    <div className="flex flex-col">
      <span className="text-muted-foreground text-xs">{label}</span>
      <span className={emphasis ? "font-mono font-semibold" : "font-mono"}>{value}</span>
    </div>
  );
}

function EntrySummary({
  entry,
  fallback,
}: {
  entry?: FiscalYearClosePlanEntry | null;
  fallback: string;
}) {
  const t = useT();

  if (!entry) {
    return <p className="text-muted-foreground text-xs">{fallback}</p>;
  }

  return (
    <div className="border-border rounded-md border p-3 text-sm">
      <div className="flex items-baseline justify-between gap-2">
        <span className="font-medium">
          {entry.kind === "Closing" ? t("Closing entry") : t("Opening entry")}
        </span>
        <span className="font-mono text-xs">{formatMinor(entry.totalDebitMinor)}</span>
      </div>
      <p className="text-muted-foreground text-xs">
        {t("{0} {1} into {2}{3} dated {4}", entry.lines.length, entry.lines.length === 1 ? "line" : "lines", entry.fiscalPeriodName, entry.createsPeriod ? ` ${t("(created by this close)")}` : "", formatUnixDate(entry.accountingDate))}
      </p>
    </div>
  );
}
