import { EmptyState } from "@/components/empty-state";
import { KpiStat } from "@/components/kpi/kpi-stat";
import { usePermission } from "@/hooks/use-permission";
import {
  fetchPayrollExportRows,
  fetchPayrollExports,
  fetchTimesheets,
  generatePayrollExport,
  PAYROLL_EXPORTS_KEY,
  TIMESHEETS_KEY,
  type PayrollExportRow,
} from "@/lib/graphql/timesheet";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { addRotaWeeks, formatShiftDate, startOfRotaWeek } from "@trenova/shared/lib/scheduling";
import { buildPayrollCsv, formatHours } from "@trenova/shared/lib/timesheet";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import {
  BanknoteIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  ClipboardCheckIcon,
  DownloadIcon,
  FileSpreadsheetIcon,
  PlayIcon,
  TimerIcon,
} from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { VoidExportDialog } from "./void-export-dialog";

type PeriodValue = "1" | "2" | "4";

const PERIOD_ITEMS = [
  { value: "1", label: "Weekly" },
  { value: "2", label: "Fortnightly" },
  { value: "4", label: "Four weeks" },
] satisfies { value: PeriodValue; label: string }[];

const SECONDS_IN_DAY = 86400;

/**
 * Payroll runs. A run locks the weeks it carried, so a period cannot be sent
 * twice without somebody voiding the first one — nothing in a payroll system
 * reads worse in an audit than a period somebody was paid for twice.
 */
export function PayrollPanel() {
  const queryClient = useQueryClient();
  const { allowed: canExport } = usePermission(Resource.Timesheet, Operation.Export);
  const [period, setPeriod] = useState<PeriodValue>("1");
  const [periodStart, setPeriodStart] = useState(() =>
    // The period the payroll clerk is most likely to want is the one that has
    // just finished, not the one people are still working.
    addRotaWeeks(startOfRotaWeek(Math.floor(Date.now() / 1000)), -1),
  );
  const [voiding, setVoiding] = useState<PayrollExportRow | null>(null);

  const weeks = Number(period);
  const periodEnd = addRotaWeeks(periodStart, weeks);

  const exports = useQuery({
    queryKey: [PAYROLL_EXPORTS_KEY],
    queryFn: ({ signal }) => fetchPayrollExports(undefined, { signal }),
    enabled: canExport,
  });

  // What the run would pick up, so the button says what it is about to do.
  const ready = useQuery({
    queryKey: [TIMESHEETS_KEY, "ready", periodStart, periodEnd],
    queryFn: ({ signal }) =>
      fetchTimesheets(
        {
          statuses: ["Approved"],
          from: periodStart,
          to: periodEnd,
          unexportedOnly: true,
        },
        { signal },
      ),
    enabled: canExport,
  });

  const { mutate: generate, isPending: generating } = useMutation({
    mutationFn: () => generatePayrollExport({ periodStart, periodEnd }),
    onSuccess: (run) => {
      toast.success(`Run created over ${run.timesheetCount} timesheet(s)`);
      void queryClient.invalidateQueries({ queryKey: [PAYROLL_EXPORTS_KEY] });
      void queryClient.invalidateQueries({ queryKey: [TIMESHEETS_KEY] });
    },
    onError: (error: Error) => toast.error(error.message),
  });

  if (!canExport) return null;

  const readySheets = ready.data ?? [];
  const readyMinutes = readySheets.reduce(
    (sum, sheet) => sum + sheet.regularMinutes + sheet.overtimeMinutes + sheet.paidLeaveMinutes,
    0,
  );
  const runs = exports.data ?? [];
  const live = runs.filter((run) => run.status !== "Voided");

  return (
    <div className="flex flex-col gap-4">
      <div className="grid grid-cols-6 gap-3">
        <KpiStat
          label="Ready to run"
          value={String(readySheets.length)}
          tone={readySheets.length > 0 ? "success" : "muted"}
          icon={<ClipboardCheckIcon className="size-[11px]" />}
          sub="Approved weeks in the period not yet sent"
        />
        <KpiStat
          label="Hours in the run"
          value={formatHours(readyMinutes)}
          icon={<TimerIcon className="size-[11px]" />}
          sub="Regular, overtime and paid leave together"
        />
        <KpiStat
          label="Runs sent"
          value={String(live.length)}
          icon={<BanknoteIcon className="size-[11px]" />}
          sub={runs.length > live.length ? `${runs.length - live.length} voided` : "None voided"}
        />
      </div>

      <section className="border-border/80 bg-card flex flex-wrap items-center justify-between gap-4 rounded-xl border p-4">
        <div className="flex flex-col gap-2">
          <p className="text-sm font-semibold">Run payroll</p>
          <div className="flex items-center gap-1">
            <Button
              size="sm"
              variant="outline"
              onClick={() => setPeriodStart(addRotaWeeks(periodStart, -weeks))}
              aria-label="Earlier period"
            >
              <ChevronLeftIcon className="size-3.5" />
            </Button>
            <span className="min-w-52 text-center text-sm font-medium tabular-nums">
              {formatShiftDate(periodStart)} – {formatShiftDate(periodEnd - SECONDS_IN_DAY)}
            </span>
            <Button
              size="sm"
              variant="outline"
              onClick={() => setPeriodStart(addRotaWeeks(periodStart, weeks))}
              aria-label="Later period"
            >
              <ChevronRightIcon className="size-3.5" />
            </Button>
          </div>
          <p className="text-muted-foreground text-xs">
            Every approved week in the period that has not gone out yet. The weeks are locked to the
            run, so the same period cannot be sent twice.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <SegmentedControl<PeriodValue>
            items={PERIOD_ITEMS}
            value={period}
            onValueChange={setPeriod}
            aria-label="Payroll period length"
          />
          <Button
            size="sm"
            isLoading={generating}
            disabled={readySheets.length === 0}
            onClick={() => generate()}
          >
            <PlayIcon className="size-3.5" />
            {readySheets.length === 0
              ? "Nothing to run"
              : `Run ${readySheets.length} timesheet${readySheets.length === 1 ? "" : "s"}`}
          </Button>
        </div>
      </section>

      {exports.isLoading ? (
        <div className="flex flex-col gap-2">
          <Skeleton className="h-16 rounded-xl" />
          <Skeleton className="h-16 rounded-xl" />
        </div>
      ) : runs.length === 0 ? (
        <EmptyState
          className="max-w-none"
          title="No payroll runs yet"
          description="Approve some weeks, pick the period, and run it. Each run is kept with the file it produced."
          icons={[ClipboardCheckIcon, BanknoteIcon, FileSpreadsheetIcon]}
        />
      ) : (
        <ul className="flex flex-col gap-2">
          {runs.map((run) => (
            <ExportRow key={run.id} run={run} onVoid={() => setVoiding(run)} />
          ))}
        </ul>
      )}

      <VoidExportDialog run={voiding} onOpenChange={(open) => !open && setVoiding(null)} />
    </div>
  );
}

function ExportRow({ run, onVoid }: { run: PayrollExportRow; onVoid: () => void }) {
  const [downloading, setDownloading] = useState(false);
  const voided = run.status === "Voided";

  async function download() {
    setDownloading(true);
    try {
      const rows = await fetchPayrollExportRows(run.id);
      const csv = buildPayrollCsv(rows);
      const blob = new Blob([csv], { type: "text/csv;charset=utf-8" });
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = `payroll-${new Date(run.periodStart * 1000).toISOString().slice(0, 10)}.csv`;
      link.click();
      URL.revokeObjectURL(url);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Could not build the file");
    } finally {
      setDownloading(false);
    }
  }

  return (
    <li
      className={cn(
        "border-border/80 bg-card hover:border-border flex flex-wrap items-center justify-between gap-3 rounded-xl border p-3 text-xs transition-colors",
        voided && "opacity-70",
      )}
    >
      <div className="flex items-center gap-3">
        <span
          className={cn(
            "grid size-9 shrink-0 place-items-center rounded-lg",
            voided ? "bg-muted text-muted-foreground" : "bg-emerald-500/10 text-emerald-600",
          )}
        >
          <FileSpreadsheetIcon className="size-4" />
        </span>
        <div className="flex flex-col">
          <span className="flex flex-wrap items-center gap-2">
            <span className="font-medium tabular-nums">
              {formatShiftDate(run.periodStart)} – {formatShiftDate(run.periodEnd - SECONDS_IN_DAY)}
            </span>
            <Badge variant={voided ? "inactive" : "active"}>{voided ? "Voided" : "Sent"}</Badge>
          </span>
          <span className="text-muted-foreground tabular-nums">
            {run.timesheetCount} timesheet{run.timesheetCount === 1 ? "" : "s"} ·{" "}
            {formatHours(run.regularMinutes)} regular · {formatHours(run.overtimeMinutes)} OT
            {run.paidLeaveMinutes > 0 ? ` · ${formatHours(run.paidLeaveMinutes)} leave` : ""}
            {run.generatedAt ? ` · ${formatShiftDate(run.generatedAt)}` : ""}
          </span>
          {run.voidReason ? (
            <span className="text-muted-foreground">Voided: {run.voidReason}</span>
          ) : null}
        </div>
      </div>

      {!voided ? (
        <div className="flex shrink-0 items-center gap-1.5">
          <Button
            size="sm"
            variant="outline"
            isLoading={downloading}
            onClick={() => void download()}
          >
            <DownloadIcon className="size-3.5" />
            CSV
          </Button>
          <Button size="sm" variant="ghost" onClick={onVoid}>
            Void
          </Button>
        </div>
      ) : null}
    </li>
  );
}
