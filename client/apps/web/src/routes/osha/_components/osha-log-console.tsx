import { useT } from "@trenova/shared/i18n/use-t";
import { InfoPopover } from "@/components/info-popover";
import { usePermission } from "@/hooks/use-permission";
import {
  certifyOshaSummary,
  deleteWorkerInjury,
  OSHA_LOG_KEY,
  OSHA_SUMMARIES_KEY,
  uncertifyOshaSummary,
  WORKER_INJURIES_KEY,
  type OshaLogCase,
} from "@/lib/graphql/worker-injury";
import {
  caseLabel,
  certificationTrack,
  summaryCaption,
  yearOf,
  yearsOffered,
  type CaseFilter,
} from "@/lib/osha-log";
import { InjuryDialog } from "@/routes/worker/_components/safety/injury-dialog";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Stepper } from "@trenova/shared/components/ui/stepper";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { MilestoneIcon, Trash2Icon } from "lucide-react";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import { OshaCaseSheet } from "./osha-case-sheet";
import { OshaCaseTable } from "./osha-case-table";
import { OshaOverview } from "./osha-overview";
import { OshaLogSkeleton } from "./osha-skeleton";
import { OshaSummaryCard } from "./osha-summary-card";
import { OshaSummaryDialog } from "./osha-summary-dialog";
import { oshaLogQuery, oshaSummariesQuery } from "./queries";

export default function OshaLogConsole() {
  const t = useT();

  const queryClient = useQueryClient();
  const { allowed: canRead } = usePermission(Resource.WorkerInjury, Operation.Read);
  const { allowed: canUpdate } = usePermission(Resource.WorkerInjury, Operation.Update);
  const { allowed: canDelete } = usePermission(Resource.WorkerInjury, Operation.Delete);
  const { allowed: canCertify } = usePermission(Resource.WorkerInjury, Operation.Manage);
  const [now] = useState(() => Math.floor(Date.now() / 1000));
  const thisYear = yearOf(now);
  const [year, setYear] = useState(thisYear);
  const [filter, setFilter] = useState<CaseFilter>("all");
  const [query, setQuery] = useState("");
  const [summaryOpen, setSummaryOpen] = useState(false);
  const [openCaseId, setOpenCaseId] = useState<string | null>(null);
  const [editing, setEditing] = useState<OshaLogCase | null>(null);
  const [deleting, setDeleting] = useState<OshaLogCase | null>(null);

  const logQuery = useQuery({ ...oshaLogQuery(year), enabled: canRead });
  const summariesQuery = useQuery({ ...oshaSummariesQuery(), enabled: canRead });

  const years = useMemo(() => yearsOffered(thisYear), [thisYear]);
  const summariesByYear = useMemo(
    () => new Map((summariesQuery.data ?? []).map((row) => [row.year, row])),
    [summariesQuery.data],
  );
  const yearItems = useMemo(
    () =>
      years.map((option) => ({
        value: String(option),
        label: String(option),
        caption: summariesQuery.data ? summaryCaption(summariesByYear.get(option)) : undefined,
      })),
    [years, summariesQuery.data, summariesByYear],
  );

  const log = logQuery.data;
  const track = useMemo(
    () =>
      log
        ? certificationTrack({
            totals: log.totals,
            summary: log.summary,
            postFrom: log.postFrom,
            postThrough: log.postThrough,
            now,
            formatDate: (unix) => formatUnixDateMedium(unix),
          })
        : [],
    [log, now],
  );
  // Read back from the query rather than kept as a copy, so the sheet shows
  // an edit the moment the log is refetched.
  const openCase = useMemo(
    () => log?.cases.find((entry) => entry.id === openCaseId) ?? null,
    [log, openCaseId],
  );

  const invalidate = () =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: [OSHA_LOG_KEY] }),
      queryClient.invalidateQueries({ queryKey: [OSHA_SUMMARIES_KEY] }),
    ]);

  const certifyMutation = useMutation({
    mutationFn: () => certifyOshaSummary(year),
    onSuccess: () => {
      toast.success(t("Summary certified"), {
        description: t("Post it where employees can see it, from February 1 to April 30."),
      });
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error(t("Could not certify the summary"), { description: error.message }),
  });

  const uncertifyMutation = useMutation({
    mutationFn: () => uncertifyOshaSummary(year),
    onSuccess: () => {
      toast.success(t("Summary reopened"), {
        description: t("The certification has been cleared so the figures can be corrected."),
      });
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error(t("Could not reopen the summary"), { description: error.message }),
  });

  const deleteMutation = useMutation({
    mutationFn: (entry: OshaLogCase) => deleteWorkerInjury(entry.id),
    onSuccess: (_, entry) => {
      toast.success(`Case ${caseLabel(entry)} deleted`, {
        description: t("The case number is not reused, so two cases can never share one."),
      });
      setDeleting(null);
      if (openCaseId === entry.id) setOpenCaseId(null);
      void queryClient.invalidateQueries({ queryKey: [WORKER_INJURIES_KEY, entry.workerId] });
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error(t("Could not delete the case"), { description: error.message }),
  });

  // The log carries names, body parts and claims. Somebody without the grant
  // sees nothing rather than an empty page implying a clean year.
  if (!canRead) return null;

  if (logQuery.isLoading || !log) return <OshaLogSkeleton />;

  return (
    <div className="flex flex-col gap-4">
      <SegmentedControl<string>
        items={yearItems}
        value={String(year)}
        onValueChange={(value) => {
          setYear(Number(value));
          setFilter("all");
          setQuery("");
        }}
        aria-label={t("Log year")}
      />

      <OshaOverview log={log} />

      <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_18rem]">
        <OshaSummaryCard
          log={log}
          canUpdate={canUpdate}
          canCertify={canCertify}
          certifying={certifyMutation.isPending}
          reopening={uncertifyMutation.isPending}
          onEditFigures={() => setSummaryOpen(true)}
          onCertify={() => certifyMutation.mutate()}
          onReopen={() => uncertifyMutation.mutate()}
        />
        <aside className="bg-card flex min-w-0 flex-col rounded-lg border">
          <header className="flex items-center gap-2 border-b px-3 py-2">
            <MilestoneIcon className="text-muted-foreground size-3.5" />
            <h2 className="text-sm font-medium">{t("Where {0} stands", log.year)}</h2>
            <InfoPopover title={`Where ${log.year} stands`}>
              {t("The year on its way to a posted 300A. OSHA wants the summary certified by a company executive and posted where employees can see it from 1 February to 30 April of the following year (29 CFR 1904.32).")}
            </InfoPopover>
          </header>
          <div className="p-3">
            <Stepper key={log.year} steps={track} aria-label={`Where ${log.year} stands`} />
          </div>
        </aside>
      </div>

      <OshaCaseTable
        year={log.year}
        cases={log.cases}
        filter={filter}
        onFilterChange={setFilter}
        query={query}
        onQueryChange={setQuery}
        canUpdate={canUpdate}
        canDelete={canDelete}
        onOpen={(entry) => setOpenCaseId(entry.id)}
        onEdit={setEditing}
        onDelete={setDeleting}
      />

      <OshaCaseSheet
        entry={openCase}
        onOpenChange={(open) => !open && setOpenCaseId(null)}
        canUpdate={canUpdate}
        canDelete={canDelete}
        onEdit={setEditing}
        onDelete={setDeleting}
      />

      <OshaSummaryDialog
        open={summaryOpen}
        onOpenChange={setSummaryOpen}
        year={year}
        summary={log.summary ?? null}
      />

      {editing ? (
        <InjuryDialog
          open
          onOpenChange={(open) => !open && setEditing(null)}
          workerId={editing.workerId}
          injury={editing}
        />
      ) : null}

      <AlertDialog open={deleting !== null} onOpenChange={(open) => !open && setDeleting(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogMedia>
              <Trash2Icon />
            </AlertDialogMedia>
            <AlertDialogTitle>{t("Delete case {0}?", deleting ? caseLabel(deleting) : "")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t("The case comes off the log and out of the totals. Its number is never reused, and the rule expects a recordable case to stay on the log for five years, so delete only a case that was recorded in error.")}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("Keep the case")}</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={deleteMutation.isPending}
              onClick={() => deleting && deleteMutation.mutate(deleting)}
            >
              {t("Delete case")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
