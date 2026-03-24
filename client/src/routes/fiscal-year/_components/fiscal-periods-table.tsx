import { DataTableColorColumn } from "@/components/data-table/_components/data-table-components";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { fiscalPeriodStatusChoices, periodTypeChoices } from "@/lib/choices";
import { formatToUserTimezone } from "@/lib/date";
import type { FiscalPeriod } from "@/types/fiscal-period";
import {
  CalendarIcon,
  LockIcon,
  MoreHorizontalIcon,
  RotateCcwIcon,
  UnlockIcon,
  XCircleIcon,
} from "lucide-react";
import { useState } from "react";
import { FiscalPeriodStatusActions } from "./fiscal-period-dialog-content";

export type FiscalPeriodAction = "close" | "reopen" | "lock" | "unlock";

export default function FiscalPeriodTable({
  periods,
}: {
  periods: FiscalPeriod[];
}) {
  const [dialogOpen, setDialogOpen] = useState(false);
  const [selectedPeriod, setSelectedPeriod] = useState<FiscalPeriod | null>(
    null,
  );
  const [selectedAction, setSelectedAction] =
    useState<FiscalPeriodAction | null>(null);

  const handleAction = (period: FiscalPeriod, action: FiscalPeriodAction) => {
    setSelectedPeriod(period);
    setSelectedAction(action);
    setDialogOpen(true);
  };

  if (!periods || periods.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center gap-2 py-12 text-center">
        <CalendarIcon className="size-8 text-muted-foreground" />
        <p className="text-sm text-muted-foreground">
          No fiscal periods found for this fiscal year.
        </p>
      </div>
    );
  }

  const sortedPeriods = [...periods].sort(
    (a, b) => a.periodNumber - b.periodNumber,
  );

  return (
    <div className="rounded-lg border bg-card">
      <Table containerClassName="max-h-[300px]">
        <TableHeader className="sticky top-0 z-30">
          <TableRow>
            <TableHead>Status</TableHead>
            <TableHead>Name</TableHead>
            <TableHead>Type</TableHead>
            <TableHead>Date Range</TableHead>
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
            const statusChoice = fiscalPeriodStatusChoices.find(
              (c) => c.value === period.status,
            );
            const typeChoice = periodTypeChoices.find(
              (c) => c.value === period.periodType,
            );
            const actions = getAvailableActions(period.status);

            return (
              <TableRow key={period.id}>
                <TableCell>
                  {statusChoice ? (
                    <DataTableColorColumn
                      text={statusChoice.label}
                      color={statusChoice.color}
                    />
                  ) : (
                    period.status
                  )}
                </TableCell>
                <TableCell className="text-sm font-medium">
                  {period.name}
                </TableCell>
                <TableCell>
                  {typeChoice ? (
                    <DataTableColorColumn
                      text={typeChoice.label}
                      color={typeChoice.color}
                    />
                  ) : (
                    period.periodType
                  )}
                </TableCell>
                <TableCell>
                  <span className="font-mono text-xs whitespace-nowrap">
                    {formatToUserTimezone(period.startDate, {
                      showTime: false,
                      showDate: true,
                    })}{" "}
                    -{" "}
                    {formatToUserTimezone(period.endDate, {
                      showTime: false,
                      showDate: true,
                    })}
                  </span>
                </TableCell>
                <TableCell>
                  {actions.length > 0 && (
                    <DropdownMenu>
                      <DropdownMenuTrigger
                        render={
                          <Button variant="ghost" size="icon-sm">
                            <MoreHorizontalIcon className="size-4" />
                          </Button>
                        }
                      />
                      <DropdownMenuContent align="end">
                        {actions.map((action) => (
                          <DropdownMenuItem
                            key={action.id}
                            startContent={<action.icon className="size-4" />}
                            title={action.label}
                            onClick={(e) => {
                              e.stopPropagation();
                              handleAction(period, action.id);
                            }}
                          />
                        ))}
                      </DropdownMenuContent>
                    </DropdownMenu>
                  )}
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
      <FiscalPeriodStatusActions
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        record={selectedPeriod ?? undefined}
        action={selectedAction || "close"}
      />
    </div>
  );
}

function getAvailableActions(status: string) {
  const actions: {
    id: FiscalPeriodAction;
    label: string;
    icon: React.ComponentType<{ className?: string }>;
  }[] = [];

  if (status === "Open") {
    actions.push({ id: "close", label: "Close Period", icon: XCircleIcon });
  }
  if (status === "Closed") {
    actions.push({
      id: "reopen",
      label: "Reopen Period",
      icon: RotateCcwIcon,
    });
    actions.push({ id: "lock", label: "Lock Period", icon: LockIcon });
  }
  if (status === "Locked") {
    actions.push({ id: "unlock", label: "Unlock Period", icon: UnlockIcon });
  }

  return actions;
}
