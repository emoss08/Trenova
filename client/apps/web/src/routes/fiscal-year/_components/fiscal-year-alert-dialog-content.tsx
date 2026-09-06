import {
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { formatUnixDate, getTodayDate } from "@trenova/shared/lib/date";
import { apiService } from "@/services/api";
import type { FiscalYearRow } from "@/lib/graphql/fiscal-year-table";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import { toast } from "sonner";

export type FiscalYearAction = "activate" | "close";

export function FiscalYearActivateAlertDialogContent({ record }: { record: FiscalYearRow }) {
  const queryClient = useQueryClient();

  const { mutateAsync } = useApiMutation({
    mutationFn: async (id: FiscalYearRow["id"]) => apiService.fiscalYearService.activate(id),
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
        <AlertDialogTitle>Set Fiscal Year {record?.year} as Current?</AlertDialogTitle>
        <div className="text-muted-foreground flex flex-col space-y-2 text-sm">
          <p>This will mark this fiscal year as the active year for transaction posting.</p>
          <p>Any currently active fiscal year will be automatically deactivated.</p>
        </div>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>Cancel</AlertDialogCancel>
        <AlertDialogAction onClick={handleFiscalYearActivate}>Set as Current</AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  );
}

export function FiscalYearCloseAlertDialogContent({ record }: { record: FiscalYearRow }) {
  const queryClient = useQueryClient();
  const today = getTodayDate();

  const { mutateAsync } = useApiMutation({
    mutationFn: async (id: FiscalYearRow["id"]) => apiService.fiscalYearService.close(id),
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
                    <p>This fiscal year does not end until {formatUnixDate(record.endDate)}</p>
                    <p className="font-semibold">
                      ({Math.ceil((record.endDate - today) / 86400)} days remaining).
                    </p>
                  </div>
                  <p>
                    Closing early will prevent posting transactions for the remainder of the period.
                  </p>
                </div>
              </div>
            </div>
          </div>
        )}
        <div className="text-muted-foreground flex flex-col text-sm">
          <p>This prevent new transactions. Only adjusting entries will be allowed.</p>
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
        <AlertDialogAction variant="destructive" onClick={handleFiscalYearClose}>
          Close Fiscal Year
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  );
}
