import {
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { getTodayDate, toDate } from "@/lib/date";
import { apiService } from "@/services/api";
import type { FiscalYear } from "@/types/fiscal-year";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import { toast } from "sonner";

export type FiscalYearAction = "activate" | "close" | "lock" | "unlock";

export function FiscalYearActivateAlertDialogContent({
  record,
}: {
  record: FiscalYear;
}) {
  const queryClient = useQueryClient();

  const { mutateAsync } = useApiMutation({
    mutationFn: async (id: FiscalYear["id"]) =>
      apiService.fiscalYearService.activate(id),
    onSuccess: () => {
      toast.success("Activated successfully", {
        description: `Successfully set ${record?.year} as current`,
      });
      void queryClient.invalidateQueries({
        queryKey: ["fiscal-year-list"],
      });
    },
  });

  const handleFiscalYearActivate = useCallback(() => {
    void mutateAsync(record?.id);
  }, [mutateAsync, record?.id]);

  return (
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>
          Set Fiscal Year {record?.year} as Current?
        </AlertDialogTitle>
        <div className="flex flex-col space-y-2 text-sm text-muted-foreground">
          <p>
            This will mark this fiscal year as the active year for transaction
            posting.
          </p>
          <p>
            Any currently active fiscal year will be automatically deactivated.
          </p>
        </div>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>Cancel</AlertDialogCancel>
        <AlertDialogAction onClick={handleFiscalYearActivate}>
          Set as Current
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  );
}

export function FiscalYearCloseAlertDialogContent({
  record,
}: {
  record: FiscalYear;
}) {
  const queryClient = useQueryClient();
  const today = getTodayDate();

  const { mutateAsync } = useApiMutation({
    mutationFn: async (id: FiscalYear["id"]) =>
      apiService.fiscalYearService.close(id),
    onSuccess: () => {
      toast.success("Closed successfully", {
        description: `Successfully closed ${record?.year}`,
      });
      void queryClient.invalidateQueries({
        queryKey: ["fiscal-year-list"],
      });
    },
  });

  const handleFiscalYearClose = useCallback(() => {
    void mutateAsync(record?.id);
  }, [mutateAsync, record?.id]);

  return (
    <AlertDialogContent className="min-w-lg">
      <AlertDialogHeader>
        <AlertDialogTitle>Close Fiscal Year {record?.year}?</AlertDialogTitle>
        {record?.endDate && record.endDate > today && (
          <div className="mb-2 flex w-full items-center justify-between rounded-md border border-yellow-600/50 bg-yellow-500/10 p-4">
            <div className="flex w-full items-center gap-3 text-yellow-600">
              <div className="flex flex-col">
                <p className="text-sm font-medium">Early Close Warning</p>
                <div className="flex flex-col gap-1 text-xs dark:text-yellow-100">
                  <div className="flex flex-row gap-0.5">
                    <p>
                      This fiscal year does not end until{" "}
                      {toDate(record.endDate)?.toLocaleDateString()}
                    </p>
                    <p className="font-semibold">
                      ({Math.ceil((record.endDate - today) / 86400)} days
                      remaining).
                    </p>
                  </div>
                  <p>
                    Closing early will prevent posting transactions for the
                    remainder of the period.
                  </p>
                </div>
              </div>
            </div>
          </div>
        )}
        <div className="flex flex-col text-sm text-muted-foreground">
          <p>
            This prevent new transactions. Only adjusting entries will be
            allowed.
          </p>
          <ul className="list-inside list-disc">
            <li>All shipments are billed</li>
            <li>Depreciation is posted</li>
            <li>Bank reconciliation complete</li>
            <li>Trial balance verified</li>
          </ul>
        </div>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>Cancel</AlertDialogCancel>
        <AlertDialogAction
          variant="destructive"
          onClick={handleFiscalYearClose}
        >
          Close Fiscal Year
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  );
}

export function FiscalYearLockAlertDialogContent({
  record,
}: {
  record: FiscalYear;
}) {
  const queryClient = useQueryClient();

  const { mutateAsync } = useApiMutation({
    mutationFn: async (id: FiscalYear["id"]) =>
      apiService.fiscalYearService.lock(id),
    onSuccess: () => {
      toast.success("Locked successfully", {
        description: `Successfully locked ${record?.year}`,
      });
      void queryClient.invalidateQueries({
        queryKey: ["fiscal-year-list"],
      });
    },
  });

  const handleFiscalYearLock = useCallback(() => {
    void mutateAsync(record?.id);
  }, [mutateAsync, record?.id]);

  return (
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>Lock Fiscal Year {record?.year}?</AlertDialogTitle>
        <div className="flex flex-col space-y-2 text-sm text-muted-foreground">
          <p>
            Locking this fiscal year will make it completely read-only. No
            transactions or adjustments will be allowed.
          </p>
          <p>This is typically done after:</p>
          <ul className="list-inside list-disc">
            <li>Audit completion</li>
            <li>Final management review</li>
            <li>All adjustments finalized</li>
          </ul>
          <p className="font-semibold text-yellow-500">
            Warning: You will need administrator privileges to unlock.
          </p>
        </div>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>Cancel</AlertDialogCancel>
        <AlertDialogAction onClick={handleFiscalYearLock}>
          Lock Fiscal Year
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  );
}

export function FiscalYearUnlockAlertDialogContent({
  record,
}: {
  record: FiscalYear;
}) {
  const queryClient = useQueryClient();

  const { mutateAsync } = useApiMutation({
    mutationFn: async (id: FiscalYear["id"]) =>
      apiService.fiscalYearService.unlock(id),
    onSuccess: () => {
      toast.success("Unlocked successfully", {
        description: `Successfully unlocked ${record?.year}`,
      });
      void queryClient.invalidateQueries({
        queryKey: ["fiscal-year-list"],
      });
    },
  });

  const handleFiscalYearUnlock = useCallback(() => {
    void mutateAsync(record?.id);
  }, [mutateAsync, record?.id]);

  return (
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>Unlock Fiscal Year {record?.year}?</AlertDialogTitle>

        <div className="flex w-full items-center justify-between rounded-md border border-red-600/50 bg-red-500/10 p-4">
          <div className="flex w-full items-center gap-3 text-red-600">
            <div className="flex flex-col">
              <p className="text-sm font-semibold">
                Administrative Action Required!
              </p>
              <p className="text-xs dark:text-red-100">
                Unlocking a fiscal year is typically only done in exceptional
                circumstances with proper authorization. This action will be
                logged for audit purposes.
              </p>
            </div>
          </div>
        </div>

        <div className="flex flex-col space-y-3 text-sm">
          <p className="text-muted-foreground">
            This will change the fiscal year status from <strong>Locked</strong>{" "}
            to <strong>Closed</strong>, allowing limited modifications.
          </p>

          <div className="space-y-2">
            <p className="font-semibold text-foreground">
              Typical reasons for unlocking:
            </p>
            <ul className="ml-2 list-inside list-disc space-y-1 text-muted-foreground">
              <li>Audit adjustments required after lock</li>
              <li>Correction of material accounting errors</li>
              <li>Regulatory compliance requirements</li>
              <li>Court order or legal mandate</li>
            </ul>
          </div>

          <div className="mt-4 space-y-2">
            <p className="font-semibold text-foreground">After unlocking:</p>
            <ul className="ml-2 list-inside list-disc space-y-1 text-muted-foreground">
              <li>Adjusting entries can be posted (if enabled)</li>
              <li>Financial reports may need regeneration</li>
              <li>The year should be re-locked after corrections</li>
              <li>External auditors may need notification</li>
            </ul>
          </div>

          <div className="mt-4 rounded-md bg-muted p-3">
            <p className="text-xs text-muted-foreground">
              <strong>Note:</strong> This action requires administrator
              privileges and will be recorded in the audit log with your user ID
              and timestamp.
            </p>
          </div>
        </div>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>Cancel</AlertDialogCancel>
        <AlertDialogAction onClick={handleFiscalYearUnlock}>
          Unlock Fiscal Year
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  );
}
