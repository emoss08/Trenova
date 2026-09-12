import { useT } from "@trenova/shared/i18n/use-t";
import { InfoPopover } from "@/components/info-popover";
import { usePermission } from "@/hooks/use-permission";
import {
  cancelWorkerChecklist,
  completeWorkerChecklistItem,
  fetchActiveWorkerChecklistTemplates,
  fetchWorkerChecklists,
  reopenWorkerChecklistItem,
  startWorkerChecklist,
  WORKER_CHECKLIST_TEMPLATES_KEY,
  WORKER_CHECKLISTS_KEY,
  type WorkerChecklistItemRow,
  type WorkerChecklistRow,
} from "@/lib/graphql/worker-checklist";
import {
  fetchWorkerEmploymentEvents,
  WORKER_EMPLOYMENT_EVENTS_KEY,
} from "@/lib/graphql/worker-employment";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { CheckIcon, ChevronDownIcon, ClipboardListIcon, PlayIcon } from "lucide-react";
import { useMemo, useState, type ReactNode } from "react";
import { toast } from "sonner";
import { ChecklistCard } from "./checklist/checklist-card";
import {
  buildEmploymentCycles,
  CYCLE_STAGE_LABELS,
  CYCLE_STEPS,
  cycleStage,
  type EmploymentCycle,
} from "./checklist/checklist-cycles";
import { ItemNoteDialog, type ItemNoteMode } from "./checklist/item-note-dialog";
import { useChecklistInvalidation } from "./checklist/use-checklist-invalidation";

type NoteState = { mode: ItemNoteMode; item: WorkerChecklistItemRow };

/**
 * The checklist tab reads as a process rather than a pile of lists. An
 * employment opens with a hire, runs through onboarding, is active, and
 * closes with a termination and an offboarding; the server starts those
 * checklists from the events, so the only thing started here by hand is a
 * custom checklist. Earlier employments sit in the history with their own
 * onboarding and offboarding, which is how a driver who left and came back
 * ends up with two of each.
 */
export default function WorkerChecklistTab({ workerId }: { workerId: string }) {
  const t = useT();

  const { allowed: canStart } = usePermission(Resource.WorkerChecklist, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.WorkerChecklist, Operation.Update);
  const { allowed: canCancel } = usePermission(Resource.WorkerChecklist, Operation.Cancel);
  const invalidate = useChecklistInvalidation(workerId);
  const [noteState, setNoteState] = useState<NoteState | null>(null);
  const [historyOpen, setHistoryOpen] = useState(false);
  const [templateId, setTemplateId] = useState("");

  const checklistsQuery = useQuery({
    queryKey: [WORKER_CHECKLISTS_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerChecklists(workerId, true, { signal }),
  });
  const eventsQuery = useQuery({
    queryKey: [WORKER_EMPLOYMENT_EVENTS_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerEmploymentEvents(workerId, undefined, { signal }),
  });
  const templatesQuery = useQuery({
    queryKey: [WORKER_CHECKLIST_TEMPLATES_KEY],
    queryFn: ({ signal }) => fetchActiveWorkerChecklistTemplates({ signal }),
    enabled: canStart,
    staleTime: 5 * 60 * 1000,
  });

  const checklists = useMemo(() => checklistsQuery.data ?? [], [checklistsQuery.data]);
  const cycles = useMemo(
    () => buildEmploymentCycles(eventsQuery.data ?? [], checklists),
    [eventsQuery.data, checklists],
  );
  const current = cycles[0] ?? null;
  const currentOpen = useMemo(
    () => (current?.checklists ?? []).filter((row) => row.status === "Open"),
    [current],
  );
  const currentClosed = useMemo(
    () => (current?.checklists ?? []).filter((row) => row.status !== "Open"),
    [current],
  );
  const previous = useMemo(() => cycles.slice(1), [cycles]);
  const historyCount =
    currentClosed.length + previous.reduce((sum, cycle) => sum + cycle.checklists.length, 0);

  // Only a template the office starts by hand is offered here. Onboarding
  // and offboarding are started by the hire and the termination.
  const startable = useMemo(() => {
    const openTemplateIds = new Set(currentOpen.map((row) => row.templateId));
    return (templatesQuery.data ?? []).filter(
      (template) => template.trigger === "Manual" && !openTemplateIds.has(template.id),
    );
  }, [templatesQuery.data, currentOpen]);
  const startableOptions = useMemo(
    () => startable.map((template) => ({ value: template.id, label: template.name })),
    [startable],
  );

  const complete = useMutation({
    mutationFn: (item: WorkerChecklistItemRow) =>
      completeWorkerChecklistItem({ id: item.id, version: item.version }),
    onSuccess: (checklist) => {
      toast.success(
        checklist.status === "Completed" ? "Checklist complete" : "Item completed",
        checklist.status === "Completed"
          ? { description: "Every required item is settled." }
          : undefined,
      );
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error(t("Could not complete item"), { description: error.message }),
  });
  const reopen = useMutation({
    mutationFn: (item: WorkerChecklistItemRow) => reopenWorkerChecklistItem(item.id, item.version),
    onSuccess: () => {
      toast.success(t("Item reopened"));
      void invalidate();
    },
    onError: (error: Error) => toast.error(t("Could not reopen item"), { description: error.message }),
  });
  const cancel = useMutation({
    mutationFn: (checklist: WorkerChecklistRow) =>
      cancelWorkerChecklist({ id: checklist.id, version: checklist.version }),
    onSuccess: () => {
      toast.success(t("Checklist cancelled"));
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error(t("Could not cancel checklist"), { description: error.message }),
  });
  const start = useMutation({
    mutationFn: (id: string) => startWorkerChecklist({ workerId, templateId: id }),
    onSuccess: (checklist) => {
      toast.success(`${checklist.name} started`);
      setTemplateId("");
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error(t("Could not start checklist"), { description: error.message }),
  });

  const busyItemId = complete.isPending
    ? complete.variables?.id
    : reopen.isPending
      ? reopen.variables?.id
      : undefined;
  const permissions = useMemo(() => ({ canUpdate, canCancel }), [canCancel, canUpdate]);

  if (checklistsQuery.isLoading || eventsQuery.isLoading) {
    return (
      <div className="flex flex-col gap-4">
        <Skeleton className="h-24 w-full rounded-lg" />
        <Skeleton className="h-48 w-full rounded-lg" />
      </div>
    );
  }

  const cardProps = {
    permissions,
    busyItemId,
    onComplete: (item: WorkerChecklistItemRow) => complete.mutate(item),
    onSkip: (item: WorkerChecklistItemRow) => setNoteState({ mode: "skip", item }),
    onNotApplicable: (item: WorkerChecklistItemRow) =>
      setNoteState({ mode: "notApplicable", item }),
    onReopen: (item: WorkerChecklistItemRow) => reopen.mutate(item),
    onCancel: (checklist: WorkerChecklistRow) => cancel.mutate(checklist),
  };

  return (
    <div className="flex flex-col gap-5">
      {current ? <EmploymentProcess cycle={current} /> : null}

      <section className="flex flex-col gap-2">
        <SectionHeading
          count={currentOpen.length}
          help={
            <>
              <p>
                {t("Onboarding starts when a hire is recorded and offboarding when a termination is; only a custom checklist is started by hand.")}
              </p>
              <p>
                {t("Credential, document and portal-access items settle themselves whenever the checklist is read and the evidence exists: an active, unexpired credential of that type, a document of that type on file, or Dash access granted (removed, for offboarding). Once every required item is settled the checklist closes on its own.")}
              </p>
            </>
          }
        >
          {t("In progress")}
        </SectionHeading>
        {currentOpen.length === 0 ? (
          <div className="text-muted-foreground flex flex-col items-center gap-2 rounded-lg border border-dashed px-4 py-8 text-center text-xs">
            <ClipboardListIcon className="size-5" />
            <p className="text-foreground text-sm font-medium">
              {checklists.length === 0 ? t("No checklists yet") : t("Nothing in progress")}
            </p>
            <p>
              {t("Onboarding starts when a hire is recorded and offboarding when a termination is. Anything else is started below.")}
            </p>
          </div>
        ) : (
          currentOpen.map((checklist) => (
            <ChecklistCard key={checklist.id} checklist={checklist} {...cardProps} />
          ))
        )}
      </section>

      {canStart && startable.length > 0 ? (
        <div className="flex flex-wrap items-center justify-between gap-3 rounded-lg border p-3">
          <div className="min-w-0">
            <p className="text-sm font-medium">{t("Start a checklist by hand")}</p>
            <p className="text-muted-foreground text-xs">
              {t("For anything the employment events do not start on their own.")}
            </p>
          </div>
          <div className="flex items-center gap-2">
            <Select
              value={templateId}
              items={startableOptions}
              onValueChange={(value) => setTemplateId(value as string)}
            >
              <SelectTrigger className="h-8 w-52 text-xs" aria-label={t("Start a checklist")}>
                <SelectValue placeholder={t("Choose a checklist")} />
              </SelectTrigger>
              <SelectContent>
                {startableOptions.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {t(option.label)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Button
              size="sm"
              disabled={!templateId || start.isPending}
              isLoading={start.isPending}
              onClick={() => start.mutate(templateId)}
            >
              <PlayIcon className="size-3.5" />
              {t("Start")}
            </Button>
          </div>
        </div>
      ) : null}

      {historyCount > 0 ? (
        <section className="flex flex-col gap-3">
          <Button
            variant="ghost"
            size="sm"
            className="text-muted-foreground w-fit px-1"
            aria-expanded={historyOpen}
            onClick={() => setHistoryOpen((value) => !value)}
          >
            <ChevronDownIcon
              className={cn("size-3.5 transition-transform", historyOpen && "rotate-180")}
            />
            {t("History ({0})", historyCount)}
          </Button>
          {historyOpen ? (
            <>
              {currentClosed.length > 0 ? (
                <div className="flex flex-col gap-2">
                  <SectionHeading>{t("This employment")}</SectionHeading>
                  {currentClosed.map((checklist) => (
                    <ChecklistCard key={checklist.id} checklist={checklist} {...cardProps} />
                  ))}
                </div>
              ) : null}
              {previous.map((cycle, index) => (
                <div
                  key={cycle.openedBy?.id ?? cycle.closedBy?.id ?? `cycle-${index}`}
                  className="flex flex-col gap-2"
                  data-testid="checklist-cycle"
                >
                  <SectionHeading>{describeCycle(cycle)}</SectionHeading>
                  {cycle.checklists.map((checklist) => (
                    <ChecklistCard key={checklist.id} checklist={checklist} {...cardProps} />
                  ))}
                </div>
              ))}
            </>
          ) : null}
        </section>
      ) : null}

      <ItemNoteDialog
        open={noteState !== null}
        onOpenChange={(isOpen) => {
          if (!isOpen) setNoteState(null);
        }}
        workerId={workerId}
        mode={noteState?.mode ?? "skip"}
        item={noteState?.item ?? null}
      />
    </div>
  );
}

function SectionHeading({
  children,
  count,
  help,
}: {
  children: string;
  count?: number;
  help?: ReactNode;
}) {
  return (
    <div className="flex items-baseline justify-between">
      <span className="flex items-center gap-1.5">
        <h4 className="text-muted-foreground text-[11px] font-semibold uppercase">{children}</h4>
        {help ? <InfoPopover title={children}>{help}</InfoPopover> : null}
      </span>
      {count !== undefined && count > 0 ? (
        <span className="text-muted-foreground font-mono text-[11px] tabular-nums">{count}</span>
      ) : null}
    </div>
  );
}

function describeCycle(cycle: EmploymentCycle): string {
  const opened = cycle.openedBy ? formatUnixDateMedium(cycle.openedBy.effectiveAt) : null;
  const closed = cycle.closedBy ? formatUnixDateMedium(cycle.closedBy.effectiveAt) : null;
  if (opened && closed) return `Employment ${opened} – ${closed}`;
  if (opened) return `Employment from ${opened}`;
  if (closed) return `Employment to ${closed}`;
  return "Undated";
}

/**
 * The stepper. The current step is the only one drawn in the primary colour;
 * steps behind it are done and steps ahead are outlines.
 */
function EmploymentProcess({ cycle }: { cycle: EmploymentCycle<WorkerChecklistRow> }) {
  const t = useT();

  const stage = cycleStage(cycle);
  const stageIndex = CYCLE_STEPS.indexOf(stage);
  const running = cycle.checklists.find((row) => row.status === "Open");

  return (
    <div data-testid="employment-process" className="flex flex-col gap-3 rounded-lg border p-4">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <h3 className="text-sm font-semibold">{t("Employment")}</h3>
        <p className="text-muted-foreground text-xs">{describeCycle(cycle)}</p>
      </div>
      <ol className="flex items-center gap-2" aria-label={t("Employment stages")}>
        {CYCLE_STEPS.map((step, index) => {
          const done = index < stageIndex;
          const active = index === stageIndex;
          return (
            <li key={step} className="flex flex-1 items-center gap-2 last:flex-none">
              <span
                className="flex items-center gap-1.5"
                aria-current={active ? "step" : undefined}
              >
                <span
                  className={cn(
                    "inline-flex size-5 items-center justify-center rounded-full border text-[10px] font-medium tabular-nums",
                    active && "border-primary bg-primary text-primary-foreground",
                    done && "border-primary/40 text-primary",
                    !active && !done && "text-muted-foreground",
                  )}
                  aria-hidden
                >
                  {done ? <CheckIcon className="size-3" /> : index + 1}
                </span>
                <span className={cn("text-xs", active ? "font-medium" : "text-muted-foreground")}>
                  {CYCLE_STAGE_LABELS[step]}
                </span>
              </span>
              {index < CYCLE_STEPS.length - 1 ? (
                <span
                  className={cn("h-px flex-1", done ? "bg-primary/40" : "bg-border")}
                  aria-hidden
                />
              ) : null}
            </li>
          );
        })}
      </ol>
      <p className="text-muted-foreground text-xs tabular-nums">
        {running
          ? t("{0} is {1}% through · {2}/{3} required settled{4}", running.name, running.progress.percent, running.progress.requiredDone, running.progress.requiredTotal, running.progress.overdue > 0 ? t("· {0} overdue", running.progress.overdue) : "")
          : stage === "left"
            ? t("Everything for this employment is settled.")
            : t("Nothing is running for this employment.")}
      </p>
    </div>
  );
}
