import { useT } from "@trenova/shared/i18n/use-t";
import {
  fetchPtoLiabilityReport,
  PTO_LIABILITY_REPORT_KEY,
  type PTOLiabilityRow,
} from "@/lib/graphql/pto-policy";
import { Badge } from "@trenova/shared/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { PTO_TERMINATION_ACTION_LABELS } from "@trenova/shared/types/pto-policy";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router";

export function formatPtoDays(value: string): string {
  const parsed = Number(value);
  return Number.isFinite(parsed)
    ? parsed.toLocaleString(undefined, { maximumFractionDigits: 2 })
    : value;
}

function workerName(row: PTOLiabilityRow): string {
  const worker = row.worker;
  if (!worker) return "Unknown worker";
  return worker.wholeName || `${worker.firstName} ${worker.lastName}`.trim();
}

export function PTOLiabilityDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();

  const { data, isLoading, isError } = useQuery({
    queryKey: [PTO_LIABILITY_REPORT_KEY],
    queryFn: ({ signal }) => fetchPtoLiabilityReport(undefined, { signal }),
    enabled: open,
    staleTime: 60 * 1000,
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[85vh] flex-col gap-0 sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>{t("PTO liability")}</DialogTitle>
          <DialogDescription>
            {t(
              "Every tracked balance valued against its policy's termination rule. Liability is what the organisation would owe if everyone left {0}.",
              data ? formatUnixDate(data.asOf) : "today",
            )}
          </DialogDescription>
        </DialogHeader>

        {isLoading ? (
          <div className="flex flex-col gap-2 py-4">
            <Skeleton className="h-14 w-full" />
            <Skeleton className="h-40 w-full" />
          </div>
        ) : isError || !data ? (
          <p className="text-destructive py-6 text-center text-sm">
            {t("The report could not be loaded. Try again in a moment.")}
          </p>
        ) : (
          <>
            <div className="my-3 grid grid-cols-2 gap-2 sm:grid-cols-4">
              <Stat label={t("Workers tracked")} value={data.workersTracked.toLocaleString()} />
              <Stat label={t("Banked days")} value={formatPtoDays(data.totalBalanceDays)} />
              <Stat
                label={t("Owed on exit")}
                value={formatPtoDays(data.liabilityDays)}
                tone="text-amber-700 dark:text-amber-400"
              />
              <Stat
                label={t("Would be forfeited")}
                value={formatPtoDays(data.forfeitableDays)}
                tone="text-muted-foreground"
              />
            </div>
            <div className="min-h-0 flex-1 overflow-auto rounded-lg border">
              {data.rows.length === 0 ? (
                <p className="text-muted-foreground p-6 text-center text-sm">
                  {t("No worker has a tracked balance yet.")}
                </p>
              ) : (
                <Table>
                  <TableHeader className="bg-muted/40 sticky top-0">
                    <TableRow>
                      <TableHead>{t("Worker")}</TableHead>
                      <TableHead>{t("Type")}</TableHead>
                      <TableHead className="text-right">{t("Balance")}</TableHead>
                      <TableHead className="text-right">{t("Accrued YTD")}</TableHead>
                      <TableHead className="text-right">{t("Used YTD")}</TableHead>
                      <TableHead>{t("On exit")}</TableHead>
                      <TableHead className="text-right">{t("Liability")}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {data.rows.map((row) => (
                      <TableRow key={`${row.workerId}-${row.ptoType}`}>
                        <TableCell className="font-medium">
                          <Link
                            to={`/hr/workers?panelType=edit&panelEntityId=${row.workerId}&pageTab=pto`}
                            className="hover:underline"
                          >
                            {workerName(row)}
                          </Link>
                        </TableCell>
                        <TableCell>{row.ptoType}</TableCell>
                        <TableCell className="text-right tabular-nums">
                          {formatPtoDays(row.balanceDays)}
                        </TableCell>
                        <TableCell className="text-muted-foreground text-right tabular-nums">
                          {formatPtoDays(row.accruedYtdDays)}
                        </TableCell>
                        <TableCell className="text-muted-foreground text-right tabular-nums">
                          {formatPtoDays(row.usedYtdDays)}
                        </TableCell>
                        <TableCell>
                          <Badge
                            variant="outline"
                            className={cn(
                              "px-1.5 py-0 text-[10px]",
                              row.onTermination === "PayOut"
                                ? "border-amber-500/40 text-amber-700 dark:text-amber-400"
                                : "text-muted-foreground",
                            )}
                          >
                            {PTO_TERMINATION_ACTION_LABELS[row.onTermination]}
                          </Badge>
                        </TableCell>
                        <TableCell
                          className={cn(
                            "text-right font-medium tabular-nums",
                            Number(row.liabilityDays) === 0 && "text-muted-foreground",
                          )}
                        >
                          {formatPtoDays(row.liabilityDays)}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
            </div>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}

function Stat({ label, value, tone }: { label: string; value: string; tone?: string }) {
  return (
    <div className="bg-muted/30 rounded-lg border p-3">
      <p className="text-muted-foreground text-[11px] font-medium uppercase">{label}</p>
      <p className={cn("mt-1 text-lg font-semibold tabular-nums", tone)}>{value}</p>
    </div>
  );
}
