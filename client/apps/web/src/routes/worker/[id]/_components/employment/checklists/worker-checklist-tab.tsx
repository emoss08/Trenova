import { ReasonDialog } from "@/components/reason-dialog";
import { SectionPanel } from "@/components/section-panel";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useSingleFlight } from "@/hooks/use-single-flight";
import { usePermission } from "@/hooks/use-permission";
import {
  cancelWorkerChecklist,
  completeWorkerChecklistItem,
  reopenWorkerChecklistItem,
  startWorkerChecklist,
  type WorkerChecklistItemRow,
  type WorkerChecklistRow,
} from "@/lib/graphql/worker-checklist";
import { useQuery } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet } from "@trenova/shared/components/ui/empty-sheet";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useState } from "react";
import { ChecklistsSkeletonBody } from "../employment-skeleton";
import {
  checklistTemplatesQuery,
  employmentEventsQuery,
  workerChecklistsQuery,
} from "../employment-reads";
import { ListSketch } from "../list-sketch";
import { ChecklistCard } from "./checklist-card";
import { buildEmploymentCycles } from "./checklist-cycles";
import { ClosedChecklists } from "./closed-checklists";
import { ItemNoteDialog, type ItemNoteMode } from "./item-note-dialog";
import { useChecklistInvalidation } from "./use-checklist-invalidation";

type NoteState = { mode: ItemNoteMode; item: WorkerChecklistItemRow };

export type ChecklistsViewProps = {
  workerId: string;
  /** A checklist a link sent the reader to (`checklist` in the address). */
  focusId: string | null;
};

/**
 * Employment › Checklists: what is open in the current employment, item by item, beside a way
 * to start one by hand and the checklists already closed. Onboarding and offboarding start from
 * the hire and the termination; only a manual template starts here.
 */
export default function WorkerChecklistTab({ workerId, focusId }: ChecklistsViewProps) {
  const t = useT();
  const { allowed: canStart } = usePermission(Resource.WorkerChecklist, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.WorkerChecklist, Operation.Update);
  const { allowed: canCancel } = usePermission(Resource.WorkerChecklist, Operation.Cancel);
  const invalidate = useChecklistInvalidation(workerId);
  const [noteState, setNoteState] = useState<NoteState | null>(null);
  const [cancelling, setCancelling] = useState<WorkerChecklistRow | null>(null);

  const checklistsQuery = useQuery(workerChecklistsQuery(workerId));
  const eventsQuery = useQuery(employmentEventsQuery(workerId));

  const complete = useApiMutation({
    resourceName: "Checklist item",
    scope: { id: `worker-checklist-items:${workerId}` },
    mutationFn: (item: WorkerChecklistItemRow) =>
      completeWorkerChecklistItem({ id: item.id, version: item.version }),
    onSuccess: () => void invalidate(),
  });
  const reopen = useApiMutation({
    resourceName: "Checklist item",
    scope: { id: `worker-checklist-items:${workerId}` },
    mutationFn: (item: WorkerChecklistItemRow) => reopenWorkerChecklistItem(item.id, item.version),
    onSuccess: () => void invalidate(),
  });
  const cancel = useApiMutation({
    resourceName: "Checklist",
    scope: { id: `worker-checklist-cancel:${workerId}` },
    mutationFn: ({ checklist, reason }: { checklist: WorkerChecklistRow; reason: string }) =>
      cancelWorkerChecklist({ id: checklist.id, reason, version: checklist.version }),
    onSuccess: () => {
      setCancelling(null);
      void invalidate();
    },
  });

  const itemOnce = useSingleFlight((action: "complete" | "reopen", item: WorkerChecklistItemRow) =>
    action === "complete" ? complete.mutateAsync(item) : reopen.mutateAsync(item),
  );
  const cancelOnce = useSingleFlight((input: { checklist: WorkerChecklistRow; reason: string }) =>
    cancel.mutateAsync(input),
  );

  if (checklistsQuery.isPending || eventsQuery.isPending) return <ChecklistsSkeletonBody />;

  const failed = checklistsQuery.isError
    ? checklistsQuery
    : eventsQuery.isError
      ? eventsQuery
      : null;
  if (failed) {
    return (
      <ErrorState
        layout="sheet"
        error={failed.error}
        onRetry={() => {
          void checklistsQuery.refetch();
          void eventsQuery.refetch();
        }}
        title={t("Couldn't load the checklists")}
        sketch={<ListSketch />}
      />
    );
  }

  const checklists = checklistsQuery.data ?? [];
  const cycles = buildEmploymentCycles(eventsQuery.data ?? [], checklists);
  const currentOpen = (cycles[0]?.checklists ?? []).filter((row) => row.status === "Open");
  const openIds = new Set(currentOpen.map((row) => row.id));
  const closed = checklists.filter((row) => !openIds.has(row.id) && row.status !== "Open");

  const busyItemId = complete.isPending
    ? complete.variables?.id
    : reopen.isPending
      ? reopen.variables?.id
      : undefined;
  const permissions = { canUpdate, canCancel };

  return (
    <div className="grid items-start gap-4 lg:grid-cols-[minmax(0,1fr)_22rem]">
      <div className="flex min-w-0 flex-col gap-4">
        {currentOpen.length === 0 ? (
          <SectionPanel title={t("In progress")}>
            <EmptySheet
              title={checklists.length === 0 ? t("No checklists yet") : t("Nothing in progress")}
              description={t(
                "Onboarding starts when a hire is recorded and offboarding when a termination is. Anything else is started beside this.",
              )}
              sketch={<ListSketch />}
            />
          </SectionPanel>
        ) : (
          currentOpen.map((checklist) => (
            <ChecklistCard
              key={checklist.id}
              checklist={checklist}
              permissions={permissions}
              busyItemId={busyItemId}
              highlighted={focusId === checklist.id}
              onComplete={(item) => void itemOnce.run("complete", item)}
              onSkip={(item) => setNoteState({ mode: "skip", item })}
              onNotApplicable={(item) => setNoteState({ mode: "notApplicable", item })}
              onReopen={(item) => void itemOnce.run("reopen", item)}
              onCancel={setCancelling}
            />
          ))
        )}
      </div>

      <div className="flex min-w-0 flex-col gap-4">
        {canStart ? (
          <StartChecklistPanel
            workerId={workerId}
            openTemplateIds={new Set(currentOpen.map((row) => row.templateId))}
          />
        ) : null}
        <ClosedChecklists
          checklists={closed}
          expandedId={focusId && !openIds.has(focusId) ? focusId : null}
        />
      </div>

      <ItemNoteDialog
        open={noteState !== null}
        onOpenChange={(isOpen) => {
          if (!isOpen) setNoteState(null);
        }}
        workerId={workerId}
        mode={noteState?.mode ?? "skip"}
        item={noteState?.item ?? null}
      />
      <ReasonDialog
        open={cancelling !== null}
        title={t("Cancel this checklist?")}
        description={t(
          "Open items stop and nothing more settles. Settled items stay on the record. This cannot be undone; start a new checklist to try again.",
        )}
        placeholder={t("e.g. Started from the wrong template")}
        confirmLabel={t("Cancel checklist")}
        cancelLabel={t("Keep checklist")}
        destructive
        pending={cancelOnce.pending}
        onConfirm={(reason) => {
          if (!cancelling) return;
          void cancelOnce.run({ checklist: cancelling, reason });
        }}
        onOpenChange={(isOpen) => {
          if (!isOpen && !cancelOnce.pending) setCancelling(null);
        }}
      />
    </div>
  );
}

function StartChecklistPanel({
  workerId,
  openTemplateIds,
}: {
  workerId: string;
  openTemplateIds: ReadonlySet<string | null>;
}) {
  const t = useT();
  const invalidate = useChecklistInvalidation(workerId);
  const [templateId, setTemplateId] = useState("");
  const templatesQuery = useQuery(checklistTemplatesQuery());
  const start = useApiMutation({
    resourceName: "Checklist",
    scope: { id: `worker-checklist-start:${workerId}` },
    mutationFn: (id: string) => startWorkerChecklist({ workerId, templateId: id }),
    onSuccess: () => {
      setTemplateId("");
      void invalidate();
    },
  });

  const startOnce = useSingleFlight((id: string) => start.mutateAsync(id));
  const startable = (templatesQuery.data ?? []).filter(
    (template) => template.trigger === "Manual" && !openTemplateIds.has(template.id),
  );
  const items = startable.map((template) => ({ value: template.id, label: template.name }));

  return (
    <SectionPanel title={t("Start a checklist")}>
      <div className="flex flex-col gap-1.5 p-3">
        <div className="grid grid-cols-[minmax(0,1fr)_auto] gap-2">
          <Select
            value={templateId}
            items={items}
            onValueChange={(value) => setTemplateId(value as string)}
            disabled={templatesQuery.isPending || items.length === 0}
          >
            <SelectTrigger size="sm" aria-label={t("Template")}>
              <SelectValue
                placeholder={
                  templatesQuery.isError
                    ? t("Templates could not be loaded")
                    : items.length === 0 && !templatesQuery.isPending
                      ? t("No template to start")
                      : t("Choose a template")
                }
              />
            </SelectTrigger>
            <SelectContent>
              {startable.map((template) => (
                <SelectItem key={template.id} value={template.id}>
                  <span className="flex w-full items-center justify-between gap-3">
                    <span className="truncate">{template.name}</span>
                    <span className="text-xs text-muted-foreground tabular-nums">
                      {t("{0, plural, one {# item} other {# items}}", template.items.length)}
                    </span>
                  </span>
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button
            size="sm"
            disabled={!templateId}
            isLoading={startOnce.pending}
            onClick={() => void startOnce.run(templateId)}
          >
            {t("Start")}
          </Button>
        </div>
        <p className="text-xs text-muted-foreground">
          {t("Items, owners and the due date come from the template.")}
        </p>
      </div>
    </SectionPanel>
  );
}
