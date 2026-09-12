import { useT } from "@trenova/shared/i18n/use-t";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { apiService } from "@/services/api";
import type { FiscalPeriod } from "@/types/fiscal-period";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import { toast } from "sonner";

type Action = "close" | "reopen" | "lock" | "unlock";

interface FiscalPeriodStatusActionsProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  record?: FiscalPeriod;
  action: Action;
}

export function FiscalPeriodStatusActions({
  open,
  onOpenChange,
  record,
  action,
}: FiscalPeriodStatusActionsProps) {
  if (!record) return null;

  const content = (() => {
    switch (action) {
      case "close":
        return <CloseDialog record={record} onOpenChange={onOpenChange} />;
      case "reopen":
        return <ReopenDialog record={record} onOpenChange={onOpenChange} />;
      case "lock":
        return <LockDialog record={record} onOpenChange={onOpenChange} />;
      case "unlock":
        return <UnlockDialog record={record} onOpenChange={onOpenChange} />;
      default:
        return null;
    }
  })();

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      {content}
    </AlertDialog>
  );
}

function CloseDialog({
  record,
  onOpenChange,
}: {
  record: FiscalPeriod;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();

  const queryClient = useQueryClient();

  const { mutateAsync, isPending } = useApiMutation({
    mutationFn: async (id: FiscalPeriod["id"]) => apiService.fiscalPeriodService.close(id),
    onSuccess: () => {
      toast.success(t("Closed successfully"), {
        description: `Successfully closed Period ${record.periodNumber}`,
      });
      void queryClient.invalidateQueries({
        queryKey: ["fiscal-period-list"],
      });
      onOpenChange(false);
    },
  });

  const handleClose = useCallback(() => {
    void mutateAsync(record.id);
  }, [mutateAsync, record.id]);

  return (
    <AlertDialogContent className="min-w-md">
      <AlertDialogHeader>
        <AlertDialogTitle className="flex items-center gap-2">{t("Close Fiscal Period")}</AlertDialogTitle>
        <AlertDialogDescription className="space-y-2">
          <p>
            {t("You are about to close")} <strong>{t("Period {0}", record.periodNumber)}</strong>
            {record.name && ` (${record.name})`}.
          </p>
          <p className="text-muted-foreground text-sm">{t("Closing this period will:")}</p>
          <ul className="text-muted-foreground list-inside list-disc space-y-1 text-sm">
            <li>{t("Prevent new transactions from being posted")}</li>
            <li>{t("Require reopening to make any changes")}</li>
            <li>{t("Enable locking once all reconciliations are complete")}</li>
          </ul>
          <p className="text-destructive font-semibold">{t("Are you sure you want to continue?")}</p>
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel disabled={isPending}>{t("Cancel")}</AlertDialogCancel>
        <AlertDialogAction variant="destructive" onClick={handleClose} disabled={isPending}>
          {isPending ? t("Closing...") : t("Close Period")}
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  );
}

function ReopenDialog({
  record,
  onOpenChange,
}: {
  record: FiscalPeriod;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();

  const queryClient = useQueryClient();

  const { mutateAsync, isPending } = useApiMutation({
    mutationFn: async (id: FiscalPeriod["id"]) => apiService.fiscalPeriodService.reopen(id),
    onSuccess: () => {
      toast.success(t("Reopened successfully"), {
        description: `Successfully reopened Period ${record.periodNumber}`,
      });
      void queryClient.invalidateQueries({
        queryKey: ["fiscal-period-list"],
      });
      onOpenChange(false);
    },
  });

  const handleReopen = useCallback(() => {
    void mutateAsync(record.id);
  }, [mutateAsync, record.id]);

  return (
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle className="flex items-center gap-2">
          {t("Reopen Fiscal Period")}
        </AlertDialogTitle>
        <AlertDialogDescription className="space-y-2">
          <p>
            {t("You are about to reopen")} <strong>{t("Period {0}", record.periodNumber)}</strong>
            {record.name && ` (${record.name})`}.
          </p>
          <p className="text-muted-foreground text-sm">{t("Reopening this period will:")}</p>
          <ul className="text-muted-foreground list-inside list-disc space-y-1 text-sm">
            <li>{t("Allow new transactions to be posted")}</li>
            <li>{t("Enable edits to existing entries")}</li>
            <li>{t("Require reclosing before locking")}</li>
          </ul>
          <p className="font-semibold text-yellow-500">
            {t("This action should only be taken when adjustments are necessary.")}
          </p>
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel disabled={isPending}>{t("Cancel")}</AlertDialogCancel>
        <AlertDialogAction onClick={handleReopen} disabled={isPending}>
          {isPending ? t("Reopening...") : t("Reopen Period")}
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  );
}

function LockDialog({
  record,
  onOpenChange,
}: {
  record: FiscalPeriod;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();

  const queryClient = useQueryClient();

  const { mutateAsync, isPending } = useApiMutation({
    mutationFn: async (id: FiscalPeriod["id"]) => apiService.fiscalPeriodService.lock(id),
    onSuccess: () => {
      toast.success(t("Locked successfully"), {
        description: `Successfully locked Period ${record.periodNumber}`,
      });
      void queryClient.invalidateQueries({
        queryKey: ["fiscal-period-list"],
      });
      onOpenChange(false);
    },
  });

  const handleLock = useCallback(() => {
    void mutateAsync(record.id);
  }, [mutateAsync, record.id]);

  return (
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle className="flex items-center gap-2">{t("Lock Fiscal Period")}</AlertDialogTitle>
        <AlertDialogDescription className="space-y-2">
          <p>
            {t("You are about to lock")} <strong>{t("Period {0}", record.periodNumber)}</strong>
            {record.name && ` (${record.name})`}.
          </p>
          <p className="text-muted-foreground text-sm">{t("Locking this period will:")}</p>
          <ul className="text-muted-foreground list-inside list-disc space-y-1 text-sm">
            <li>{t("Permanently prevent all changes")}</li>
            <li>{t("Finalize all financial data")}</li>
            <li>{t("Require special permission to unlock")}</li>
            <li>{t("Complete the period close workflow")}</li>
          </ul>
          <p className="text-destructive font-semibold">
            {t("This is typically done after audits are complete. Continue?")}
          </p>
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel disabled={isPending}>{t("Cancel")}</AlertDialogCancel>
        <AlertDialogAction onClick={handleLock} disabled={isPending}>
          {isPending ? t("Locking...") : t("Lock Period")}
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  );
}

function UnlockDialog({
  record,
  onOpenChange,
}: {
  record: FiscalPeriod;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();

  const queryClient = useQueryClient();

  const { mutateAsync, isPending } = useMutation({
    mutationFn: async (id: FiscalPeriod["id"]) => apiService.fiscalPeriodService.unlock(id),
    onSuccess: () => {
      toast.success(t("Unlocked successfully"), {
        description: `Successfully unlocked Period ${record.periodNumber}`,
      });
      void queryClient.invalidateQueries({
        queryKey: ["fiscal-period-list"],
      });
      onOpenChange(false);
    },
  });

  const handleUnlock = useCallback(() => {
    void mutateAsync(record.id);
  }, [mutateAsync, record.id]);

  return (
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle className="flex items-center gap-2">
          {t("Unlock Fiscal Period")}
        </AlertDialogTitle>
        <AlertDialogDescription className="space-y-2">
          <p>
            {t("You are about to unlock")} <strong>{t("Period {0}", record.periodNumber)}</strong>
            {record.name && ` (${record.name})`}.
          </p>
          <p className="text-muted-foreground text-sm">{t("Unlocking this period will:")}</p>
          <ul className="text-muted-foreground list-inside list-disc space-y-1 text-sm">
            <li>{t("Return the period to Closed status")}</li>
            <li>{t("Allow reopening if needed")}</li>
            <li>{t("Require manager approval")}</li>
            <li>{t("Create an audit trail entry")}</li>
          </ul>
          <p className="text-destructive font-semibold">
            {t("This action should only be taken in exceptional circumstances with proper authorization.")}
          </p>
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel disabled={isPending}>{t("Cancel")}</AlertDialogCancel>
        <AlertDialogAction onClick={handleUnlock} disabled={isPending}>
          {isPending ? t("Unlocking...") : t("Unlock Period")}
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  );
}
