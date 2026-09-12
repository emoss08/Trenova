import { useT } from "@trenova/shared/i18n/use-t";
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
import { summarizeSheets } from "@/lib/time-attendance";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { CompositionBar } from "@trenova/shared/components/ui/composition-bar";
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
import { useMemo, useState } from "react";
import { toast } from "sonner";
import { PayrollRunsEmpty } from "./time-attendance-empty";
import { VoidExportDialog } from "./void-export-dialog";

type PeriodValue = "1" | "2" | "4";

const PERIOD_ITEMS = [
  { value: "1", label: "Weekly" },
  { value: "2", label: "Fortnightly" },
  { value: "4", label: "Four weeks" },
] satisfies { value: PeriodValue; label: string }[];

const SECONDS_IN_DAY = 86400;

function hourSegments(totals: {
  regularMinutes: number;
  overtimeMinutes: number;
  paidLeaveMinutes: number;
}) {
  return [
    { key: "regular", label: "Regular", value: totals.regularMinutes },
    { key: "overtime", label: "Overtime", value: totals.overtimeMinutes },
    { key: "leave", label: "Paid leave", value: totals.paidLeaveMinutes },
  ];
}

/**
 * Payroll runs. A run locks the weeks it carried, so a period cannot be sent
 * twice without somebody voiding the first one. Nothing in a payroll system
 * reads worse in an audit than a period somebody was paid for twice.
 */
export function PayrollPanel() {
  const t = useT();

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

  const readySheets = useMemo(() => ready.data ?? [], [ready.data]);
  const readySummary = useMemo(() => summarizeSheets(readySheets), [readySheets]);

  if (!canExport) return null;

  const runs = exports.data ?? [];
  const live = runs.filter((run) => run.status !== "Voided");

  return (
    <div className="flex flex-col gap-4">
      <div className="grid grid-cols-6 gap-3">
        <KpiStat
          label={t("Ready to run")}
          value={String(readySheets.length)}
          tone={readySheets.length > 0 ? "success" : "muted"}
          icon={<ClipboardCheckIcon className="size-[11px]" />}
          sub={
            readySheets.length > 0
              ? `${readySummary.workers} ${readySummary.workers === 1 ? "person" : "people"} in the period`
              : "Approved weeks in the period not yet sent"
          }
        />
        <KpiStat
          label={t("Hours in the run")}
          value={formatHours(readySummary.totalMinutes)}
          icon={<TimerIcon className="size-[11px]" />}
          sub={
            readySummary.overtimeMinutes > 0
              ? `${formatHours(readySummary.overtimeMinutes)} of it overtime, on ${readySummary.overtimeWeeks} week${readySummary.overtimeWeeks === 1 ? "" : "s"}`
              : "Regular, overtime and paid leave together"
          }
        />
        <KpiStat
          label={t("Runs sent")}
          value={String(live.length)}
          icon={<BanknoteIcon className="size-[11px]" />}
          sub={runs.length > live.length ? `${runs.length - live.length} voided` : "None voided"}
        />
      </div>

      <section
        aria-labelledby="run-payroll-heading"
        className="bg-card flex flex-col overflow-hidden rounded-lg border"
      >
        <header className="flex flex-wrap items-center justify-between gap-3 border-b px-4 py-3">
          <div className="flex min-w-0 flex-col gap-0.5">
            <h3 id="run-payroll-heading" className="text-sm font-medium">
              {t("Run payroll")}
            </h3>
            <p className="text-muted-foreground text-xs">
              {t("Every approved week in the period that has not gone out yet. The weeks lock to the run, so the same period cannot be sent twice.")}
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <SegmentedControl<PeriodValue>
              items={PERIOD_ITEMS}
              value={period}
              onValueChange={setPeriod}
              aria-label={t("Payroll period length")}
            />
            <div className="flex items-center gap-1">
              <Button
                size="icon-sm"
                variant="outline"
                onClick={() => setPeriodStart(addRotaWeeks(periodStart, -weeks))}
                aria-label={t("Earlier period")}
              >
                <ChevronLeftIcon className="size-3.5" />
              </Button>
              <span className="min-w-36 text-center text-sm font-medium tabular-nums">
                {formatShiftDate(periodStart)} – {formatShiftDate(periodEnd - SECONDS_IN_DAY)}
              </span>
              <Button
                size="icon-sm"
                variant="outline"
                onClick={() => setPeriodStart(addRotaWeeks(periodStart, weeks))}
                aria-label={t("Later period")}
              >
                <ChevronRightIcon className="size-3.5" />
              </Button>
            </div>
            <Button
              size="sm"
              isLoading={generating}
              disabled={readySheets.length === 0}
              onClick={() => generate()}
            >
              <PlayIcon className="size-3.5" />
              {readySheets.length === 0
                ? t("Nothing to run")
                : t("Run {0} timesheet{1}", readySheets.length, readySheets.length === 1 ? "" : "s")}
            </Button>
          </div>
        </header>

        {ready.isLoading ? (
          <div className="flex flex-col gap-2 p-3">
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-2/3" />
          </div>
        ) : readySheets.length === 0 ? (
          <p className="text-muted-foreground px-4 py-3 text-xs">
            {t("No approved week in this period is waiting to be sent. Move the period, or approve some weeks first.")}
          </p>
        ) : (
          <>
            <ul
              aria-label={t("Timesheets in the run")}
              className="max-h-72 divide-y overflow-y-auto text-xs"
            >
              {readySheets.map((sheet) => (
                <li
                  key={sheet.id}
                  className="grid grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto] items-center gap-4 px-4 py-2"
                >
                  <span className="flex min-w-0 flex-wrap items-center gap-2">
                    <span className="truncate font-medium">
                      {sheet.worker
                        ? `${sheet.worker.firstName} ${sheet.worker.lastName}`
                        : sheet.workerId}
                    </span>
                    <span className="text-muted-foreground tabular-nums">
                      {t("Week of {0}", formatShiftDate(sheet.periodStart))}
                    </span>
                  </span>
                  <CompositionBar
                    size="sm"
                    showLegend={false}
                    aria-label={`${sheet.worker ? `${sheet.worker.firstName} ${sheet.worker.lastName}` : sheet.workerId}'s hours`}
                    formatValue={formatHours}
                    segments={hourSegments(sheet)}
                  />
                  <span className="font-mono tabular-nums">
                    {formatHours(sheet.totalMinutes)}
                    {sheet.overtimeMinutes > 0 ? (
                      <span className="text-muted-foreground">
                        {t("· {0} OT", formatHours(sheet.overtimeMinutes))}
                      </span>
                    ) : null}
                  </span>
                </li>
              ))}
            </ul>
            <footer className="bg-muted/40 flex flex-wrap items-center justify-between gap-3 border-t px-4 py-2 text-xs">
              <CompositionBar
                size="sm"
                className="max-w-md min-w-0 flex-1"
                aria-label={t("Hours in the run")}
                formatValue={formatHours}
                segments={hourSegments(readySummary)}
              />
              <span className="font-mono text-sm font-semibold tabular-nums">
                {formatHours(readySummary.totalMinutes)}
              </span>
            </footer>
          </>
        )}
      </section>

      <section aria-labelledby="payroll-runs-heading" className="flex flex-col gap-2">
        <h3 id="payroll-runs-heading" className="text-sm font-medium">
          {t("Runs")}
        </h3>
        {exports.isLoading ? (
          <div className="flex flex-col gap-2">
            <Skeleton className="h-16 rounded-lg" />
            <Skeleton className="h-16 rounded-lg" />
          </div>
        ) : runs.length === 0 ? (
          <PayrollRunsEmpty
            title={t("No payroll runs yet")}
            description={t("Approve some weeks, pick the period, and run it. Each run is kept with the file it produced.")}
          />
        ) : (
          <ul className="bg-card divide-y overflow-hidden rounded-lg border">
            {runs.map((run) => (
              <ExportRow key={run.id} run={run} onVoid={() => setVoiding(run)} />
            ))}
          </ul>
        )}
      </section>

      <VoidExportDialog run={voiding} onOpenChange={(open) => !open && setVoiding(null)} />
    </div>
  );
}

function ExportRow({ run, onVoid }: { run: PayrollExportRow; onVoid: () => void }) {
  const t = useT();

  const [downloading, setDownloading] = useState(false);
  const voided = run.status === "Voided";
  const total = run.regularMinutes + run.overtimeMinutes + run.paidLeaveMinutes;

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
        "grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-3 px-3 py-2.5 text-xs md:grid-cols-[auto_minmax(0,1.2fr)_minmax(0,1fr)_auto]",
        voided && "opacity-70",
      )}
    >
      <span className="bg-accent inline-flex size-7 shrink-0 items-center justify-center rounded-md">
        <FileSpreadsheetIcon className="size-4" />
      </span>
      <div className="flex min-w-0 flex-col gap-0.5">
        <span className="flex flex-wrap items-center gap-2">
          <span className="text-sm font-medium tabular-nums">
            {formatShiftDate(run.periodStart)} – {formatShiftDate(run.periodEnd - SECONDS_IN_DAY)}
          </span>
          <Badge variant={voided ? "inactive" : "active"}>{voided ? t("Voided") : t("Sent")}</Badge>
        </span>
        <span className="text-muted-foreground tabular-nums">
          {t("{0} timesheet{1} {2} {3}", run.timesheetCount, run.timesheetCount === 1 ? "" : "s", run.generatedAt ? t(", sent {0}", formatShiftDate(run.generatedAt)) : "", run.voidReason ? t(". Voided: {0}", run.voidReason) : "")}
        </span>
      </div>
      <div className="col-span-3 flex min-w-0 items-center gap-3 md:col-span-1">
        <CompositionBar
          size="sm"
          showLegend={false}
          className="min-w-0 flex-1"
          aria-label={t("Hours in the run")}
          formatValue={formatHours}
          segments={hourSegments(run)}
        />
        <span className="font-mono text-sm font-semibold tabular-nums">{formatHours(total)}</span>
      </div>
      {!voided ? (
        <div className="flex shrink-0 items-center gap-1.5 md:col-start-4">
          <Button
            size="sm"
            variant="outline"
            isLoading={downloading}
            onClick={() => void download()}
          >
            <DownloadIcon className="size-3.5" />
            {t("CSV")}
          </Button>
          <Button size="sm" variant="ghost" onClick={onVoid}>
            {t("Void")}
          </Button>
        </div>
      ) : (
        <span className="md:col-start-4" />
      )}
    </li>
  );
}
