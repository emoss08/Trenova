import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { FiscalPeriodSchema } from "@/lib/schemas/fiscal-period-schema";
import { api } from "@/services/api";
import { APIError } from "@/types/errors";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import { toast } from "sonner";

type Action = "close" | "reopen" | "lock" | "unlock";

interface FiscalPeriodStatusActionsProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  record?: FiscalPeriodSchema;
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
  record: FiscalPeriodSchema;
  onOpenChange: (open: boolean) => void;
}) {
  const queryClient = useQueryClient();

  const { mutateAsync, isPending } = useMutation({
    mutationFn: async (id: FiscalPeriodSchema["id"]) =>
      api.fiscalPeriod.close(id),
    onSuccess: () => {
      toast.success("Closed successfully", {
        description: `Successfully closed Period ${record.periodNumber}`,
      });
      queryClient.invalidateQueries({
        queryKey: ["fiscal-period-list"],
      });
      onOpenChange(false);
    },
    onError: (error: APIError) => {
      if (error.isValidationError()) {
        toast.error("Failed to close fiscal period", {
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

  const handleClose = useCallback(() => {
    mutateAsync(record.id);
  }, [mutateAsync, record.id]);

  return (
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle className="flex items-center gap-2">
          Close Fiscal Period
        </AlertDialogTitle>
        <AlertDialogDescription className="space-y-2">
          <p>
            You are about to close <strong>Period {record.periodNumber}</strong>
            {record.name && ` (${record.name})`}.
          </p>
          <p className="text-sm text-muted-foreground">
            Closing this period will:
          </p>
          <ul className="list-inside list-disc space-y-1 text-sm text-muted-foreground">
            <li>Prevent new transactions from being posted</li>
            <li>Require reopening to make any changes</li>
            <li>Enable locking once all reconciliations are complete</li>
          </ul>
          <p className="font-semibold text-destructive">
            Are you sure you want to continue?
          </p>
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel disabled={isPending}>Cancel</AlertDialogCancel>
        <AlertDialogAction onClick={handleClose} disabled={isPending}>
          {isPending ? "Closing..." : "Close Period"}
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  );
}

function ReopenDialog({
  record,
  onOpenChange,
}: {
  record: FiscalPeriodSchema;
  onOpenChange: (open: boolean) => void;
}) {
  const queryClient = useQueryClient();

  const { mutateAsync, isPending } = useMutation({
    mutationFn: async (id: FiscalPeriodSchema["id"]) =>
      api.fiscalPeriod.reopen(id),
    onSuccess: () => {
      toast.success("Reopened successfully", {
        description: `Successfully reopened Period ${record.periodNumber}`,
      });
      queryClient.invalidateQueries({
        queryKey: ["fiscal-period-list"],
      });
      onOpenChange(false);
    },
    onError: (error: APIError) => {
      if (error.isValidationError()) {
        toast.error("Failed to reopen fiscal period", {
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

  const handleReopen = useCallback(() => {
    mutateAsync(record.id);
  }, [mutateAsync, record.id]);

  return (
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle className="flex items-center gap-2">
          Reopen Fiscal Period
        </AlertDialogTitle>
        <AlertDialogDescription className="space-y-2">
          <p>
            You are about to reopen{" "}
            <strong>Period {record.periodNumber}</strong>
            {record.name && ` (${record.name})`}.
          </p>
          <p className="text-sm text-muted-foreground">
            Reopening this period will:
          </p>
          <ul className="list-inside list-disc space-y-1 text-sm text-muted-foreground">
            <li>Allow new transactions to be posted</li>
            <li>Enable edits to existing entries</li>
            <li>Require reclosing before locking</li>
          </ul>
          <p className="font-semibold text-yellow-500">
            This action should only be taken when adjustments are necessary.
          </p>
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel disabled={isPending}>Cancel</AlertDialogCancel>
        <AlertDialogAction onClick={handleReopen} disabled={isPending}>
          {isPending ? "Reopening..." : "Reopen Period"}
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  );
}

function LockDialog({
  record,
  onOpenChange,
}: {
  record: FiscalPeriodSchema;
  onOpenChange: (open: boolean) => void;
}) {
  const queryClient = useQueryClient();

  const { mutateAsync, isPending } = useMutation({
    mutationFn: async (id: FiscalPeriodSchema["id"]) =>
      api.fiscalPeriod.lock(id),
    onSuccess: () => {
      toast.success("Locked successfully", {
        description: `Successfully locked Period ${record.periodNumber}`,
      });
      queryClient.invalidateQueries({
        queryKey: ["fiscal-period-list"],
      });
      onOpenChange(false);
    },
    onError: (error: APIError) => {
      if (error.isValidationError()) {
        toast.error("Failed to lock fiscal period", {
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

  const handleLock = useCallback(() => {
    mutateAsync(record.id);
  }, [mutateAsync, record.id]);

  return (
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle className="flex items-center gap-2">
          Lock Fiscal Period
        </AlertDialogTitle>
        <AlertDialogDescription className="space-y-2">
          <p>
            You are about to lock <strong>Period {record.periodNumber}</strong>
            {record.name && ` (${record.name})`}.
          </p>
          <p className="text-sm text-muted-foreground">
            Locking this period will:
          </p>
          <ul className="list-inside list-disc space-y-1 text-sm text-muted-foreground">
            <li>Permanently prevent all changes</li>
            <li>Finalize all financial data</li>
            <li>Require special permission to unlock</li>
            <li>Complete the period close workflow</li>
          </ul>
          <p className="font-semibold text-destructive">
            This is typically done after audits are complete. Continue?
          </p>
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel disabled={isPending}>Cancel</AlertDialogCancel>
        <AlertDialogAction onClick={handleLock} disabled={isPending}>
          {isPending ? "Locking..." : "Lock Period"}
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  );
}

function UnlockDialog({
  record,
  onOpenChange,
}: {
  record: FiscalPeriodSchema;
  onOpenChange: (open: boolean) => void;
}) {
  const queryClient = useQueryClient();

  const { mutateAsync, isPending } = useMutation({
    mutationFn: async (id: FiscalPeriodSchema["id"]) =>
      api.fiscalPeriod.unlock(id),
    onSuccess: () => {
      toast.success("Unlocked successfully", {
        description: `Successfully unlocked Period ${record.periodNumber}`,
      });
      queryClient.invalidateQueries({
        queryKey: ["fiscal-period-list"],
      });
      onOpenChange(false);
    },
    onError: (error: APIError) => {
      if (error.isValidationError()) {
        toast.error("Failed to unlock fiscal period", {
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

  const handleUnlock = useCallback(() => {
    mutateAsync(record.id);
  }, [mutateAsync, record.id]);

  return (
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle className="flex items-center gap-2">
          Unlock Fiscal Period
        </AlertDialogTitle>
        <AlertDialogDescription className="space-y-2">
          <p>
            You are about to unlock{" "}
            <strong>Period {record.periodNumber}</strong>
            {record.name && ` (${record.name})`}.
          </p>
          <p className="text-sm text-muted-foreground">
            Unlocking this period will:
          </p>
          <ul className="list-inside list-disc space-y-1 text-sm text-muted-foreground">
            <li>Return the period to Closed status</li>
            <li>Allow reopening if needed</li>
            <li>Require manager approval</li>
            <li>Create an audit trail entry</li>
          </ul>
          <p className="font-semibold text-destructive">
            This action should only be taken in exceptional circumstances with
            proper authorization.
          </p>
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel disabled={isPending}>Cancel</AlertDialogCancel>
        <AlertDialogAction onClick={handleUnlock} disabled={isPending}>
          {isPending ? "Unlocking..." : "Unlock Period"}
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  );
}
