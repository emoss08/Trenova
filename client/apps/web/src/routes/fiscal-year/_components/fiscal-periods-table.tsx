import { DataTableColorColumn } from "@/components/data-table/_components/data-table-components";
import { usePermissionCheck } from "@/hooks/use-permission";
import { fiscalPeriodStatusChoices, periodTypeChoices } from "@/lib/choices";
import { FISCAL_CALENDAR_TIMEZONE } from "@/lib/fiscal-calendar";
import {
  getFiscalPeriodActions,
  type FiscalPeriodAction,
  type FiscalPeriodActionBlocker,
} from "@/lib/fiscal-period-actions";
import type { FiscalPeriod } from "@/types/fiscal-period";
import type { FiscalYearStatus } from "@/types/fiscal-year";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import { Resource } from "@trenova/shared/types/permission";
import {
  CalendarIcon,
  LockIcon,
  MoreHorizontalIcon,
  PlayIcon,
  RotateCcwIcon,
  ShieldCheckIcon,
  UnlockIcon,
  XCircleIcon,
  type LucideIcon,
} from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { FiscalPeriodActionDialog } from "./fiscal-period-dialog-content";

const actionIcons: Record<FiscalPeriodAction, LucideIcon> = {
  activate: PlayIcon,
  lock: LockIcon,
  unlock: UnlockIcon,
  close: XCircleIcon,
  reopen: RotateCcwIcon,
};

function actionLabel(action: FiscalPeriodAction, t: TranslateFn) {
  switch (action) {
    case "activate":
      return t("Open period");
    case "lock":
      return t("Lock period");
    case "unlock":
      return t("Unlock period");
    case "close":
      return t("Close period");
    case "reopen":
      return t("Reopen period");
  }
}

function blockerMessage(blocker: FiscalPeriodActionBlocker, t: TranslateFn) {
  switch (blocker.kind) {
    case "fiscalYearClosed":
      return t("The fiscal year is closed. Reopen the fiscal year first.");
    case "fiscalYearPermanentlyClosed":
      return t("The fiscal year is permanently closed.");
    case "earlierPeriodNotOpened":
      return t("Period {0} has not been opened yet", blocker.periodNumber);
    case "earlierPeriodOpen":
      return t("Period {0} is still open", blocker.periodNumber);
    case "laterPeriodClosed":
      return t("Period {0} is already closed", blocker.periodNumber);
  }
}

type FiscalPeriodTableProps = {
  periods: FiscalPeriod[];
  fiscalYearStatus: FiscalYearStatus | undefined;
  onPeriodUpdated: (period: FiscalPeriod) => void;
};

type PendingAction = { period: FiscalPeriod; action: FiscalPeriodAction };

export function FiscalPeriodTable({
  periods,
  fiscalYearStatus,
  onPeriodUpdated,
}: FiscalPeriodTableProps) {
  const t = useT();
  const { check } = usePermissionCheck();

  const [pending, setPending] = useState<PendingAction | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);

  const sortedPeriods = useMemo(
    () => [...periods].sort((a, b) => a.periodNumber - b.periodNumber),
    [periods],
  );

  const can = useCallback(
    (operation: Parameters<typeof check>[1]) => check(Resource.FiscalPeriod, operation),
    [check],
  );

  const handleSelect = useCallback((period: FiscalPeriod, action: FiscalPeriodAction) => {
    setPending({ period, action });
    setDialogOpen(true);
  }, []);

  if (sortedPeriods.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center gap-2 py-12 text-center">
        <CalendarIcon className="text-muted-foreground size-8" />
        <p className="text-muted-foreground text-sm">
          {t("No fiscal periods found for this fiscal year.")}
        </p>
      </div>
    );
  }

  return (
    <div className="bg-card rounded-lg border">
      <Table containerClassName="max-h-[300px]">
        <TableHeader className="sticky top-0 z-30">
          <TableRow>
            <TableHead>{t("Status")}</TableHead>
            <TableHead>{t("Name")}</TableHead>
            <TableHead>{t("Type")}</TableHead>
            <TableHead>{t("Date range")}</TableHead>
            <TableHead className="w-10" />
          </TableRow>
        </TableHeader>
        <TableBody
          // REMINDER: avoids scroll (skipping the table header) when using skip to content
          style={{
            scrollMarginTop: "calc(var(--top-bar-height) + 40px)",
          }}
        >
          {sortedPeriods.map((period) => {
            const statusChoice = fiscalPeriodStatusChoices.find((c) => c.value === period.status);
            const typeChoice = periodTypeChoices.find((c) => c.value === period.periodType);
            const options = getFiscalPeriodActions({
              period,
              periods: sortedPeriods,
              fiscalYearStatus,
              can,
            });

            return (
              <TableRow key={period.id}>
                <TableCell>
                  {statusChoice ? (
                    <DataTableColorColumn text={t(statusChoice.label)} color={statusChoice.color} />
                  ) : (
                    period.status
                  )}
                </TableCell>
                <TableCell className="text-sm font-medium">{period.name}</TableCell>
                <TableCell>
                  {typeChoice ? (
                    <DataTableColorColumn text={t(typeChoice.label)} color={typeChoice.color} />
                  ) : (
                    period.periodType
                  )}
                </TableCell>
                <TableCell>
                  <span className="font-mono text-xs whitespace-nowrap">
                    {formatToUserTimezone(
                      period.startDate,
                      { showTime: false, showDate: true },
                      FISCAL_CALENDAR_TIMEZONE,
                    )}{" "}
                    -{" "}
                    {formatToUserTimezone(
                      period.endDate,
                      { showTime: false, showDate: true },
                      FISCAL_CALENDAR_TIMEZONE,
                    )}
                  </span>
                </TableCell>
                <TableCell>
                  {(options.length > 0 || period.status === "PermanentlyClosed") && (
                    <DropdownMenu>
                      <DropdownMenuTrigger
                        render={
                          <Button
                            type="button"
                            variant="ghost"
                            size="icon-sm"
                            aria-label={t("Actions for {0}", period.name)}
                          >
                            <MoreHorizontalIcon className="size-4" />
                          </Button>
                        }
                      />
                      <DropdownMenuContent align="end" className="max-w-72">
                        {period.status === "PermanentlyClosed" ? (
                          <DropdownMenuItem
                            disabled
                            startContent={<ShieldCheckIcon className="size-4" />}
                            title={t("Permanently closed")}
                            description={t("This period's figures are final and cannot change.")}
                            descriptionClassProps="whitespace-normal"
                          />
                        ) : (
                          options.map(({ action, blocker }) => {
                            const Icon = actionIcons[action];
                            return (
                              <DropdownMenuItem
                                key={action}
                                disabled={blocker !== null}
                                startContent={<Icon className="size-4" />}
                                title={actionLabel(action, t)}
                                description={blocker ? blockerMessage(blocker, t) : undefined}
                                descriptionClassProps="whitespace-normal"
                                onClick={(event) => {
                                  event.stopPropagation();
                                  handleSelect(period, action);
                                }}
                              />
                            );
                          })
                        )}
                      </DropdownMenuContent>
                    </DropdownMenu>
                  )}
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
      <FiscalPeriodActionDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        period={pending?.period ?? null}
        action={pending?.action ?? null}
        onCompleted={onPeriodUpdated}
      />
    </div>
  );
}
