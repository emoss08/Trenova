import { usePermissionCheck } from "@/hooks/use-permission";
import { getFiscalYearActions, type FiscalYearAction } from "@/lib/fiscal-year-actions";
import type { FiscalYear } from "@/types/fiscal-year";
import { Button } from "@trenova/shared/components/ui/button";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { Resource, type OperationType } from "@trenova/shared/types/permission";
import { PlayIcon, RotateCcwIcon, XCircleIcon, type LucideIcon } from "lucide-react";
import { useCallback, useState } from "react";
import { FiscalYearActionDialog } from "./fiscal-year-alert-dialog-content";

const actionIcons: Record<FiscalYearAction, LucideIcon> = {
  activate: PlayIcon,
  close: XCircleIcon,
  reopen: RotateCcwIcon,
};

function actionLabel(action: FiscalYearAction, t: TranslateFn) {
  switch (action) {
    case "activate":
      return t("Set as current");
    case "close":
      return t("Close year");
    case "reopen":
      return t("Reopen year");
  }
}

type FiscalYearStatusSummaryProps = Pick<FiscalYear, "status" | "isCurrent">;

function statusSummary({ status, isCurrent }: FiscalYearStatusSummaryProps, t: TranslateFn) {
  if (isCurrent) {
    return t("This is the current fiscal year.");
  }
  switch (status) {
    case "Draft":
      return t("This fiscal year is a draft. Set it as current to open it for posting.");
    case "Open":
      return t("This fiscal year is open but is not the current fiscal year.");
    case "Closed":
      return t("This fiscal year is closed. Reopen it to post corrections.");
    case "PermanentlyClosed":
      return t("This fiscal year is permanently closed and cannot change.");
  }
}

export function FiscalYearStatusSummary({ status, isCurrent }: FiscalYearStatusSummaryProps) {
  const t = useT();

  if (!status) return null;

  return (
    <span className="text-2xs text-muted-foreground">
      {statusSummary({ status, isCurrent }, t)}
    </span>
  );
}

type FiscalYearLifecycleActionsProps = {
  fiscalYear: Pick<FiscalYear, "id" | "year" | "endDate" | "status" | "isCurrent">;
  onCompleted: (fiscalYear: FiscalYear) => void;
};

export function FiscalYearLifecycleActions({
  fiscalYear,
  onCompleted,
}: FiscalYearLifecycleActionsProps) {
  const t = useT();
  const { check } = usePermissionCheck();
  const [action, setAction] = useState<FiscalYearAction | null>(null);

  const actions = getFiscalYearActions({
    status: fiscalYear.status,
    isCurrent: fiscalYear.isCurrent,
    can: (operation: OperationType) => check(Resource.FiscalYear, operation),
  });

  const handleClose = useCallback(() => setAction(null), []);
  const { id } = fiscalYear;

  if (!id) return null;

  return (
    <>
      {actions.map((candidate) => {
        const Icon = actionIcons[candidate];
        return (
          <Button
            key={candidate}
            type="button"
            size="sm"
            variant={candidate === "activate" ? "default" : "outline"}
            onClick={() => setAction(candidate)}
          >
            <Icon className="size-4" />
            {actionLabel(candidate, t)}
          </Button>
        );
      })}
      {action && (
        <FiscalYearActionDialog
          open
          onOpenChange={(open) => {
            if (!open) handleClose();
          }}
          action={action}
          record={{ id, year: fiscalYear.year, endDate: fiscalYear.endDate }}
          onClose={handleClose}
          onCompleted={onCompleted}
        />
      )}
    </>
  );
}
