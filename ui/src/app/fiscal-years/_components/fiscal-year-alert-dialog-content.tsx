import {
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { VisuallyHidden } from "@/components/ui/visually-hidden";
import { getTodayDate, toDate } from "@/lib/date";
import { FiscalYearSchema } from "@/lib/schemas/fiscal-year-schema";
import { api } from "@/services/api";
import { APIError } from "@/types/errors";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import { toast } from "sonner";

export function FiscalYearCloseAlertDialogContent({
  record,
}: {
  record: FiscalYearSchema;
}) {
  const queryClient = useQueryClient();
  const today = getTodayDate();

  const { mutateAsync } = useMutation({
    mutationFn: async (id: FiscalYearSchema["id"]) => api.fiscalYear.close(id),
    onSuccess: () => {
      toast.success("Closed successfully", {
        description: `Successfully closed ${record?.year}`,
      });
      queryClient.invalidateQueries({
        queryKey: ["fiscal-year-list"],
      });
    },
    onError: (error: APIError) => {
      if (error.isValidationError()) {
        toast.error("Failed to close fiscal year", {
          description: error.message,
        });
      }

      if (error.isRateLimitError()) {
        toast.error("Rate limit exceeded", {
          description:
            "You have exceeded the rate limit. Please try again later.",
        });
      }
    },
  });

  const handleFiscalYearClose = useCallback(() => {
    mutateAsync(record?.id);
  }, [mutateAsync, record?.id]);

  return (
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>Close Fiscal Year {record?.year}?</AlertDialogTitle>
        <VisuallyHidden>
          <AlertDialogDescription />
        </VisuallyHidden>
        {record && record.endDate > today && (
          <div className="mb-4 flex w-full items-center justify-between rounded-md border border-yellow-600/50 bg-yellow-500/10 p-4">
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
        <div className="flex flex-col space-y-2 text-sm text-muted-foreground">
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
        <AlertDialogAction onClick={handleFiscalYearClose}>
          Close Fiscal Year
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  );
}

export function FiscalYearLockAlertDialogContent({
  record,
}: {
  record: FiscalYearSchema;
}) {
  const queryClient = useQueryClient();

  const { mutateAsync } = useMutation({
    mutationFn: async (id: FiscalYearSchema["id"]) => api.fiscalYear.lock(id),
    onSuccess: () => {
      toast.success("Locked successfully", {
        description: `Successfully locked ${record?.year}`,
      });
      queryClient.invalidateQueries({
        queryKey: ["fiscal-year-list"],
      });
    },
    onError: (error: APIError) => {
      if (error.isValidationError()) {
        toast.error("Failed to lock fiscal year", {
          description: error.message,
        });
      }

      if (error.isRateLimitError()) {
        toast.error("Rate limit exceeded", {
          description:
            "You have exceeded the rate limit. Please try again later.",
        });
      }
    },
  });

  const handleFiscalYearLock = useCallback(() => {
    mutateAsync(record?.id);
  }, [mutateAsync, record?.id]);

  return (
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>Lock Fiscal Year {record?.year}?</AlertDialogTitle>
        <VisuallyHidden>
          <AlertDialogDescription />
        </VisuallyHidden>
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
  record: FiscalYearSchema;
}) {
  const queryClient = useQueryClient();

  const { mutateAsync } = useMutation({
    mutationFn: async (id: FiscalYearSchema["id"]) => api.fiscalYear.unlock(id),
    onSuccess: () => {
      toast.success("Unlocked successfully", {
        description: `Successfully unlocked ${record?.year}`,
      });
      queryClient.invalidateQueries({
        queryKey: ["fiscal-year-list"],
      });
    },
    onError: (error: APIError) => {
      if (error.isValidationError()) {
        toast.error("Failed to unlock fiscal year", {
          description: error.message,
        });
      }

      if (error.isRateLimitError()) {
        toast.error("Rate limit exceeded", {
          description:
            "You have exceeded the rate limit. Please try again later.",
        });
      }
    },
  });

  const handleFiscalYearUnlock = useCallback(() => {
    mutateAsync(record?.id);
  }, [mutateAsync, record?.id]);

  return (
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>Unlock Fiscal Year {record?.year}?</AlertDialogTitle>
        <VisuallyHidden>
          <AlertDialogDescription />
        </VisuallyHidden>

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
