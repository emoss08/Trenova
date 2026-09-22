import { useApiMutation } from "@/hooks/use-api-mutation";
import type { FiscalPeriodAction } from "@/lib/fiscal-period-actions";
import { apiService } from "@/services/api";
import type { FiscalPeriod } from "@/types/fiscal-period";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
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
import { Label } from "@trenova/shared/components/ui/label";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertTriangleIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

type FiscalPeriodActionDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  period: FiscalPeriod | null;
  action: FiscalPeriodAction | null;
  onCompleted: (period: FiscalPeriod) => void;
};

export function FiscalPeriodActionDialog({
  open,
  onOpenChange,
  period,
  action,
  onCompleted,
}: FiscalPeriodActionDialogProps) {
  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      {period && action && (
        <FiscalPeriodActionDialogContent
          key={`${period.id}:${action}`}
          period={period}
          action={action}
          onOpenChange={onOpenChange}
          onCompleted={onCompleted}
        />
      )}
    </AlertDialog>
  );
}

type ActionCopy = {
  title: string;
  intro: string;
  effectsHeading: string;
  effects: string[];
  confirm: string;
  pending: string;
  success: string;
  successDescription: string;
  destructive: boolean;
};

function actionCopy(action: FiscalPeriodAction, period: FiscalPeriod, t: TranslateFn): ActionCopy {
  switch (action) {
    case "activate":
      return {
        title: t("Open fiscal period"),
        intro: t("You are about to open {0}.", period.name),
        effectsHeading: t("Opening this period will:"),
        effects: [
          t("Accept subledger postings such as invoices and vendor bills"),
          t("Accept manual journal entries"),
        ],
        confirm: t("Open period"),
        pending: t("Opening..."),
        success: t("Period opened"),
        successDescription: t("{0} is now open", period.name),
        destructive: false,
      };
    case "lock":
      return {
        title: t("Lock fiscal period"),
        intro: t("You are about to lock {0}.", period.name),
        effectsHeading: t("Locking this period will:"),
        effects: [
          t("Block subledger postings such as billing, receivables, and payables"),
          t("Keep accepting manual journal entries for accruals and adjustments"),
          t("Allow unlocking if subledger postings need to resume"),
        ],
        confirm: t("Lock period"),
        pending: t("Locking..."),
        success: t("Period locked"),
        successDescription: t("{0} is now locked", period.name),
        destructive: false,
      };
    case "unlock":
      return {
        title: t("Unlock fiscal period"),
        intro: t("You are about to unlock {0}.", period.name),
        effectsHeading: t("Unlocking this period will:"),
        effects: [t("Return the period to Open"), t("Accept subledger postings again")],
        confirm: t("Unlock period"),
        pending: t("Unlocking..."),
        success: t("Period unlocked"),
        successDescription: t("{0} is open again", period.name),
        destructive: false,
      };
    case "close":
      return {
        title: t("Close fiscal period"),
        intro: t("You are about to close {0}.", period.name),
        effectsHeading: t("Closing this period will:"),
        effects: [
          t("Block every posting, including manual journal entries"),
          t("Require reopening, with a recorded reason, before anything can change"),
        ],
        confirm: t("Close period"),
        pending: t("Closing..."),
        success: t("Period closed"),
        successDescription: t("{0} is now closed", period.name),
        destructive: true,
      };
    case "reopen":
      return {
        title: t("Reopen fiscal period"),
        intro: t("You are about to reopen {0}.", period.name),
        effectsHeading: t("Reopening this period will:"),
        effects: [
          t("Return the period to Open"),
          t("Accept subledger postings and manual journal entries again"),
          t("Record who reopened it, when, and why"),
        ],
        confirm: t("Reopen period"),
        pending: t("Reopening..."),
        success: t("Period reopened"),
        successDescription: t("{0} is open again", period.name),
        destructive: true,
      };
  }
}

function runAction(action: FiscalPeriodAction, periodId: FiscalPeriod["id"], reason: string) {
  const service = apiService.fiscalPeriodService;
  switch (action) {
    case "activate":
      return service.activate(periodId);
    case "lock":
      return service.lock(periodId);
    case "unlock":
      return service.unlock(periodId);
    case "close":
      return service.close(periodId);
    case "reopen":
      return service.reopen(periodId, reason.trim());
  }
}

type FiscalPeriodActionDialogContentProps = {
  period: FiscalPeriod;
  action: FiscalPeriodAction;
  onOpenChange: (open: boolean) => void;
  onCompleted: (period: FiscalPeriod) => void;
};

function FiscalPeriodActionDialogContent({
  period,
  action,
  onOpenChange,
  onCompleted,
}: FiscalPeriodActionDialogContentProps) {
  const t = useT();
  const queryClient = useQueryClient();
  const [reason, setReason] = useState("");

  const copy = actionCopy(action, period, t);
  const needsReason = action === "reopen";
  const checksClose = action === "close";

  const closeCheck = useQuery({
    queryKey: ["fiscal-period-close-blockers", period.id],
    queryFn: () => apiService.fiscalPeriodService.closeBlockers(period.id),
    enabled: checksClose,
    staleTime: 0,
  });

  const mutation = useApiMutation({
    mutationFn: () => runAction(action, period.id, reason),
    resourceName: "Fiscal Period",
    onSuccess: (updated: FiscalPeriod) => {
      toast.success(copy.success, { description: copy.successDescription });
      onCompleted({ ...period, ...updated });
      void queryClient.invalidateQueries({ queryKey: ["fiscal-year-list"] });
      void queryClient.invalidateQueries({ queryKey: ["fiscal-year-close-preview"] });
      onOpenChange(false);
    },
  });

  const closeBlocked =
    checksClose && (closeCheck.isLoading || closeCheck.isError || !closeCheck.data?.canClose);
  const confirmDisabled =
    mutation.isPending || closeBlocked || (needsReason && reason.trim().length === 0);

  return (
    <AlertDialogContent className="min-w-md">
      <AlertDialogHeader>
        <AlertDialogTitle>{copy.title}</AlertDialogTitle>
        <AlertDialogDescription render={<div />} className="flex flex-col gap-2 text-left">
          <p>{copy.intro}</p>
          <p className="text-muted-foreground text-sm">{copy.effectsHeading}</p>
          <ul className="text-muted-foreground list-inside list-disc space-y-1 text-sm">
            {copy.effects.map((effect) => (
              <li key={effect}>{effect}</li>
            ))}
          </ul>
        </AlertDialogDescription>
        {checksClose && closeCheck.isLoading && <Skeleton className="h-12 w-full" />}
        {checksClose && closeCheck.isError && (
          <Alert variant="destructive">
            <AlertTriangleIcon />
            <AlertTitle>{t("Close check unavailable")}</AlertTitle>
            <AlertDescription>
              <p>{t("The period could not be checked for close blockers. Try again.")}</p>
            </AlertDescription>
          </Alert>
        )}
        {checksClose && closeCheck.data && closeCheck.data.blockers.length > 0 && (
          <Alert variant="destructive">
            <AlertTriangleIcon />
            <AlertTitle>
              {closeCheck.data.blockers.length === 1
                ? t("1 issue blocks this close")
                : t("{0} issues block this close", closeCheck.data.blockers.length)}
            </AlertTitle>
            <AlertDescription>
              <ul className="list-inside list-disc">
                {closeCheck.data.blockers.map((blocker) => (
                  <li key={`${blocker.field}-${blocker.message}`}>{blocker.message}</li>
                ))}
              </ul>
            </AlertDescription>
          </Alert>
        )}
        {needsReason && (
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="fiscal-period-reopen-reason">{t("Reason")}</Label>
            <Textarea
              id="fiscal-period-reopen-reason"
              value={reason}
              onChange={(event) => setReason(event.target.value)}
              placeholder={t("Why is this period being reopened?")}
              rows={3}
            />
            <p className="text-muted-foreground text-xs">
              {t("Recorded on the period for the audit trail.")}
            </p>
          </div>
        )}
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel disabled={mutation.isPending}>{t("Cancel")}</AlertDialogCancel>
        <AlertDialogAction
          variant={copy.destructive ? "destructive" : "default"}
          disabled={confirmDisabled}
          onClick={() => mutation.mutate()}
        >
          {mutation.isPending ? copy.pending : copy.confirm}
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  );
}
