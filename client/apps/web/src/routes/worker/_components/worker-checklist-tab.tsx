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
import { useMutation, useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { CHECKLIST_KIND_LABELS, type ChecklistKind } from "@trenova/shared/types/worker-checklist";
import { ChevronDownIcon, ClipboardListIcon, PlayIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import { ChecklistCard } from "./checklist/checklist-card";
import { ItemNoteDialog, type ItemNoteMode } from "./checklist/item-note-dialog";
import { useChecklistInvalidation } from "./checklist/use-checklist-invalidation";

type NoteState = { mode: ItemNoteMode; item: WorkerChecklistItemRow };

export default function WorkerChecklistTab({ workerId }: { workerId: string }) {
  const { allowed: canStart } = usePermission(Resource.WorkerChecklist, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.WorkerChecklist, Operation.Update);
  const { allowed: canCancel } = usePermission(Resource.WorkerChecklist, Operation.Cancel);
  const invalidate = useChecklistInvalidation(workerId);
  const [noteState, setNoteState] = useState<NoteState | null>(null);
  const [historyOpen, setHistoryOpen] = useState(false);

  const checklistsQuery = useQuery({
    queryKey: [WORKER_CHECKLISTS_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerChecklists(workerId, true, { signal }),
  });
  const templatesQuery = useQuery({
    queryKey: [WORKER_CHECKLIST_TEMPLATES_KEY],
    queryFn: ({ signal }) => fetchActiveWorkerChecklistTemplates({ signal }),
    enabled: canStart,
    staleTime: 5 * 60 * 1000,
  });

  const checklists = useMemo(() => checklistsQuery.data ?? [], [checklistsQuery.data]);
  const open = useMemo(() => checklists.filter((c) => c.status === "Open"), [checklists]);
  const closed = useMemo(() => checklists.filter((c) => c.status !== "Open"), [checklists]);
  const startable = useMemo(() => {
    const openTemplateIds = new Set(open.map((c) => c.templateId));
    return (templatesQuery.data ?? []).filter((template) => !openTemplateIds.has(template.id));
  }, [templatesQuery.data, open]);

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
      toast.error("Could not complete item", { description: error.message }),
  });
  const reopen = useMutation({
    mutationFn: (item: WorkerChecklistItemRow) => reopenWorkerChecklistItem(item.id, item.version),
    onSuccess: () => {
      toast.success("Item reopened");
      void invalidate();
    },
    onError: (error: Error) => toast.error("Could not reopen item", { description: error.message }),
  });
  const cancel = useMutation({
    mutationFn: (checklist: WorkerChecklistRow) =>
      cancelWorkerChecklist({ id: checklist.id, version: checklist.version }),
    onSuccess: () => {
      toast.success("Checklist cancelled");
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error("Could not cancel checklist", { description: error.message }),
  });
  const start = useMutation({
    mutationFn: (templateId: string) => startWorkerChecklist({ workerId, templateId }),
    onSuccess: (checklist) => {
      toast.success(`${checklist.name} started`);
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error("Could not start checklist", { description: error.message }),
  });

  const busyItemId = complete.isPending
    ? complete.variables?.id
    : reopen.isPending
      ? reopen.variables?.id
      : undefined;
  const permissions = useMemo(() => ({ canUpdate, canCancel }), [canCancel, canUpdate]);

  if (checklistsQuery.isLoading) {
    return (
      <div className="flex flex-col gap-3">
        <Skeleton className="h-24 w-full rounded-xl" />
        <Skeleton className="h-40 w-full rounded-xl" />
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
    <div className="flex flex-col gap-4">
      {open.length === 0 ? (
        <div className="border-border text-muted-foreground flex flex-col items-center gap-2 rounded-xl border border-dashed px-4 py-8 text-center text-sm">
          <ClipboardListIcon className="size-5" />
          {checklists.length === 0 ? "No checklists yet" : "No open checklists"}
        </div>
      ) : (
        open.map((checklist) => (
          <ChecklistCard key={checklist.id} checklist={checklist} {...cardProps} />
        ))
      )}

      {canStart && startable.length > 0 ? (
        <section className="flex flex-col gap-2">
          <p className="text-muted-foreground text-[11px] font-semibold tracking-wide uppercase">
            Start a checklist
          </p>
          <div className="flex flex-wrap gap-2">
            {startable.map((template) => (
              <Button
                key={template.id}
                size="sm"
                variant="outline"
                aria-label={`Start ${template.name}`}
                disabled={start.isPending}
                onClick={() => start.mutate(template.id)}
              >
                <PlayIcon className="size-3.5" />
                {template.name}
                <span className="text-muted-foreground ml-1 text-[10px]">
                  {CHECKLIST_KIND_LABELS[template.kind as ChecklistKind] ?? template.kind}
                </span>
              </Button>
            ))}
          </div>
        </section>
      ) : null}

      {closed.length > 0 ? (
        <section className="flex flex-col gap-2">
          <Button
            variant="ghost"
            size="sm"
            className="text-muted-foreground w-fit px-1"
            aria-expanded={historyOpen}
            onClick={() => setHistoryOpen((value) => !value)}
          >
            <ChevronDownIcon
              className={`size-3.5 transition-transform ${historyOpen ? "rotate-180" : ""}`}
            />
            History ({closed.length})
          </Button>
          {historyOpen
            ? closed.map((checklist) => (
                <ChecklistCard key={checklist.id} checklist={checklist} {...cardProps} />
              ))
            : null}
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
