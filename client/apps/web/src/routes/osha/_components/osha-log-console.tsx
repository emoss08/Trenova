import { usePermission } from "@/hooks/use-permission";
import {
  certifyOshaSummary,
  fetchOshaLog,
  OSHA_LOG_KEY,
  uncertifyOshaSummary,
} from "@/lib/graphql/worker-injury";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDate, getTodayDate } from "@trenova/shared/lib/date";
import {
  caseClassificationLabel,
  classificationTone,
  formatRate,
  illnessTypeLabel,
} from "@trenova/shared/lib/injury";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useState } from "react";
import { toast } from "sonner";
import { OshaSummaryDialog } from "./osha-summary-dialog";

/** How many years back the picker offers. Five is the retention period. */
const YEARS_OFFERED = 5;

function currentYear(): number {
  return new Date(getTodayDate() * 1000).getUTCFullYear();
}

export default function OshaLogConsole() {
  const queryClient = useQueryClient();
  const { allowed: canUpdate } = usePermission(Resource.WorkerInjury, Operation.Update);
  const { allowed: canCertify } = usePermission(Resource.WorkerInjury, Operation.Manage);
  const [year, setYear] = useState(currentYear());
  const [summaryOpen, setSummaryOpen] = useState(false);

  const logQuery = useQuery({
    queryKey: [OSHA_LOG_KEY, year],
    queryFn: ({ signal }) => fetchOshaLog(year, { signal }),
  });

  const invalidate = () => queryClient.invalidateQueries({ queryKey: [OSHA_LOG_KEY] });

  const certifyMutation = useMutation({
    mutationFn: () => certifyOshaSummary(year),
    onSuccess: () => {
      toast.success("Summary certified", {
        description: "Post it where employees can see it, from February 1 to April 30.",
      });
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error("Could not certify the summary", { description: error.message }),
  });

  const uncertifyMutation = useMutation({
    mutationFn: () => uncertifyOshaSummary(year),
    onSuccess: () => {
      toast.success("Summary reopened", {
        description: "The certification has been cleared so the figures can be corrected.",
      });
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error("Could not reopen the summary", { description: error.message }),
  });

  if (logQuery.isLoading) {
    return (
      <div className="flex flex-col gap-3">
        <Skeleton className="h-32 w-full" />
        <Skeleton className="h-40 w-full" />
      </div>
    );
  }

  const log = logQuery.data;
  if (!log) return null;

  const totals = log.totals;
  const summary = log.summary;
  const years = Array.from({ length: YEARS_OFFERED }, (_, index) => currentYear() - index);

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-center gap-1.5">
        {years.map((option) => (
          <Button
            key={option}
            size="sm"
            variant={option === year ? "default" : "outline"}
            onClick={() => setYear(option)}
          >
            {option}
          </Button>
        ))}
      </div>

      <section className="rounded-lg border p-4">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              <h2 className="text-sm font-medium">300A summary for {log.year}</h2>
              <Badge variant={summary?.status === "Certified" ? "active" : "warning"}>
                {summary?.status === "Certified" ? "Certified" : "Draft"}
              </Badge>
              {totals.openCases > 0 ? (
                <Badge variant="warning">
                  {totals.openCases} case{totals.openCases === 1 ? "" : "s"} still open
                </Badge>
              ) : null}
            </div>
            <p className="text-muted-foreground mt-1 text-xs">
              Post from {formatUnixDate(log.postFrom)} to {formatUnixDate(log.postThrough)}
              {summary?.certifiedAt ? ` · certified ${formatUnixDate(summary.certifiedAt)}` : ""}
            </p>
          </div>
          <div className="flex shrink-0 gap-2">
            {canUpdate ? (
              <Button size="sm" variant="outline" onClick={() => setSummaryOpen(true)}>
                {summary ? "Edit figures" : "Start the summary"}
              </Button>
            ) : null}
            {canCertify && summary && summary.status !== "Certified" ? (
              <Button
                size="sm"
                isLoading={certifyMutation.isPending}
                onClick={() => certifyMutation.mutate()}
              >
                Certify
              </Button>
            ) : null}
            {canCertify && summary?.status === "Certified" ? (
              <Button
                size="sm"
                variant="ghost"
                isLoading={uncertifyMutation.isPending}
                onClick={() => uncertifyMutation.mutate()}
              >
                Reopen
              </Button>
            ) : null}
          </div>
        </div>

        <dl className="mt-4 grid grid-cols-2 gap-3 text-xs sm:grid-cols-4 lg:grid-cols-6">
          <Figure label="Deaths" value={totals.deaths} alarm={totals.deaths > 0} />
          <Figure label="Days away" value={totals.daysAwayCases} />
          <Figure label="Transfer / restriction" value={totals.jobTransferCases} />
          <Figure label="Other recordable" value={totals.otherRecordableCases} />
          <Figure label="Total recordable" value={totals.totalRecordableCases} />
          <Figure label="Total days away" value={totals.totalDaysAway} />
        </dl>

        <dl className="mt-3 grid grid-cols-2 gap-3 text-xs sm:grid-cols-4 lg:grid-cols-6">
          <Figure label="Injuries" value={totals.injuryCount} />
          <Figure label="Skin disorders" value={totals.skinDisorderCount} />
          <Figure label="Respiratory" value={totals.respiratoryCount} />
          <Figure label="Poisonings" value={totals.poisoningCount} />
          <Figure label="Hearing loss" value={totals.hearingLossCount} />
          <Figure label="Other illnesses" value={totals.otherIllnessCount} />
        </dl>

        <div className="border-border/60 mt-4 grid grid-cols-2 gap-3 border-t pt-3 text-xs sm:grid-cols-4">
          <div>
            <dt className="text-muted-foreground text-[11px]">TRIR</dt>
            <dd className="font-semibold tabular-nums">
              {formatRate(log.totalRecordableIncidentRate)}
            </dd>
          </div>
          <div>
            <dt className="text-muted-foreground text-[11px]">DART</dt>
            <dd className="font-semibold tabular-nums">{formatRate(log.daysAwayRestrictedRate)}</dd>
          </div>
          <div>
            <dt className="text-muted-foreground text-[11px]">Average employees</dt>
            <dd className="font-semibold tabular-nums">{summary?.averageEmployees ?? "—"}</dd>
          </div>
          <div>
            <dt className="text-muted-foreground text-[11px]">Hours worked</dt>
            <dd className="font-semibold tabular-nums">
              {summary?.totalHoursWorked ? summary.totalHoursWorked.toLocaleString("en-US") : "—"}
            </dd>
          </div>
        </div>
        {log.totalRecordableIncidentRate === null ? (
          <p className="text-muted-foreground mt-2 text-[11px]">
            The rates need the hours worked. A rate with no denominator is not a small number — it
            is not a number.
          </p>
        ) : null}
      </section>

      <section>
        <h2 className="cc-label text-foreground mb-2">
          Log for {log.year} ({log.cases.length} case{log.cases.length === 1 ? "" : "s"})
        </h2>
        {log.cases.length === 0 ? (
          <p className="text-muted-foreground rounded-md border border-dashed p-4 text-xs">
            No case has been recorded for {log.year}.
          </p>
        ) : (
          <ul className="flex flex-col gap-1.5">
            {log.cases.map((entry) => (
              <li
                key={entry.id}
                className={cn(
                  "rounded-md border px-3 py-2 text-xs",
                  !entry.recordable && "opacity-70",
                )}
              >
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <span className="flex flex-wrap items-center gap-2">
                    <span className="text-muted-foreground tabular-nums">
                      {entry.caseYear}-{entry.caseNumber}
                    </span>
                    <span className="font-medium">{entry.logName}</span>
                    <Badge variant={classificationTone(entry.classification)}>
                      {caseClassificationLabel(entry.classification)}
                    </Badge>
                    {entry.recordable ? null : <Badge variant="secondary">Off the log</Badge>}
                    {entry.status === "Open" ? <Badge variant="warning">Open</Badge> : null}
                  </span>
                  <span className="text-muted-foreground tabular-nums">
                    {formatUnixDate(entry.occurredAt)}
                  </span>
                </div>
                <p className="text-muted-foreground mt-1">
                  {illnessTypeLabel(entry.illnessType)}
                  {entry.bodyPart ? ` · ${entry.bodyPart}` : ""}
                  {entry.location ? ` · ${entry.location}` : ""}
                  {entry.daysAway > 0 ? ` · ${entry.daysAway} days away` : ""}
                  {entry.daysRestricted > 0 ? ` · ${entry.daysRestricted} restricted` : ""}
                </p>
              </li>
            ))}
          </ul>
        )}
      </section>

      <OshaSummaryDialog
        open={summaryOpen}
        onOpenChange={setSummaryOpen}
        year={year}
        summary={summary ?? null}
      />
    </div>
  );
}

function Figure({ label, value, alarm }: { label: string; value: number; alarm?: boolean }) {
  return (
    <div>
      <dt className="text-muted-foreground text-[11px]">{label}</dt>
      <dd
        className={cn(
          "text-sm font-semibold tabular-nums",
          alarm && "text-red-600 dark:text-red-400",
        )}
      >
        {value}
      </dd>
    </div>
  );
}
