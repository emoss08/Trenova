import { ConfirmDialog } from "@/components/confirm-dialog";
import type { PreloadIntent } from "@/components/work-pane";
import { RowActionsMenu, type RowAction } from "@/components/row-actions-menu";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import { VirtualRows, type VirtualRow } from "@/components/virtual-rows";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  deleteWorkerInjury,
  OSHA_LOG_KEY,
  WORKER_INJURIES_KEY,
  type WorkerInjuryRow,
} from "@/lib/graphql/worker-injury";
import { useQueryClient } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { Edit02Icon, PlusIcon, Trash01Icon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import {
  caseClassificationLabel,
  claimStatusLabel,
  claimStatusTone,
  classificationTone,
  illnessTypeLabel,
} from "@trenova/shared/lib/injury";
import { useState } from "react";
import { ListMoreLine } from "../list-more";
import { ShimmerLine } from "../tab-skeleton-parts";
import type { WorkerList } from "../use-worker-list";

export type InjuryListProps = {
  workerId: string;
  injuries: WorkerList<WorkerInjuryRow>;
  canRecord: boolean;
  canUpdate: boolean;
  canDelete: boolean;
  onRecord: () => void;
  onEdit: (injury: WorkerInjuryRow) => void;
  /** Reads the case pane's code when a control that opens it is pointed at or focused. */
  paneIntent: PreloadIntent;
  /** A case came off the record, so a pane open on it can close. */
  onDeleted: (injury: WorkerInjuryRow) => void;
};

/**
 * The worker's injury and illness cases beside their safety record: each with its type and body
 * part, its OSHA classification and whether it is on the 300 log, and its claim. The log for the
 * whole establishment is the OSHA page; this is the one person's share of it, newest first, a
 * page at a time.
 */
export function InjuryList({
  workerId,
  injuries,
  canRecord,
  canUpdate,
  canDelete,
  onRecord,
  onEdit,
  paneIntent,
  onDeleted,
}: InjuryListProps) {
  const t = useT();
  const queryClient = useQueryClient();
  const [deleting, setDeleting] = useState<WorkerInjuryRow | null>(null);

  const remove = useApiMutation({
    scope: { id: `worker-injury-delete:${workerId}` },
    resourceName: "Case",
    mutationFn: (injury: WorkerInjuryRow) => deleteWorkerInjury(injury.id),
    onSuccess: (_, injury) => {
      onDeleted(injury);
      return Promise.all([
        queryClient.invalidateQueries({ queryKey: [WORKER_INJURIES_KEY, workerId] }),
        queryClient.invalidateQueries({ queryKey: [OSHA_LOG_KEY] }),
      ]);
    },
  });

  const rows: VirtualRow[] = injuries.rows.map((injury) => ({
    key: injury.id,
    render: () => (
      <InjuryRow
        injury={injury}
        canUpdate={canUpdate}
        canDelete={canDelete}
        deleting={remove.isPending}
        paneIntent={paneIntent}
        onEdit={onEdit}
        onDelete={setDeleting}
      />
    ),
  }));

  return (
    <SectionPanel
      title={t("Injuries and illnesses")}
      // Exact only: the server's total when it sends one, or the rows once every page is in.
      count={injuries.total ?? (injuries.hasMore ? undefined : injuries.rows.length)}
      hint={t("OSHA 300 log")}
      action={
        canRecord ? (
          <Button size="sm" variant="quiet" {...paneIntent} onClick={onRecord}>
            <PlusIcon className="size-3.5" />
            {t("Record a case")}
          </Button>
        ) : null
      }
    >
      {injuries.isPending ? (
        <div aria-busy aria-label={t("Loading the cases")} className="flex flex-col gap-2 p-3">
          <ShimmerLine className="w-3/5" />
          <ShimmerLine className="w-2/5" />
        </div>
      ) : injuries.isError ? (
        <div className="p-3">
          <ErrorState
            error={injuries.error}
            layout="compact"
            title={t("The cases could not be loaded")}
            onRetry={injuries.refetch}
          />
        </div>
      ) : rows.length === 0 ? (
        <SectionPanelQuiet>{t("No cases yet.")}</SectionPanelQuiet>
      ) : (
        <VirtualRows
          rows={rows}
          mask="card"
          className="max-h-80"
          contentClassName="p-1.5"
          hoverGlide
          estimateSize={84}
          onEndReached={
            injuries.hasMore && !injuries.loadMoreFailed ? injuries.loadMore : undefined
          }
          footer={<ListMoreLine more={injuries} />}
          aria-label={t("Injuries and illnesses")}
        />
      )}
      <ConfirmDialog
        open={deleting !== null}
        onOpenChange={(open) => !open && setDeleting(null)}
        title={t("Delete this case?")}
        description={t(
          "It comes off the worker's file and off the OSHA 300 log. Keep it and close it instead if it simply ended. This cannot be undone.",
        )}
        confirmLabel={t("Delete case")}
        pending={remove.isPending}
        onConfirm={() => (deleting ? remove.mutateAsync(deleting) : Promise.resolve(undefined))}
      />
    </SectionPanel>
  );
}

function InjuryRow({
  injury,
  canUpdate,
  canDelete,
  deleting,
  paneIntent,
  onEdit,
  onDelete,
}: {
  injury: WorkerInjuryRow;
  canUpdate: boolean;
  canDelete: boolean;
  deleting: boolean;
  paneIntent: PreloadIntent;
  onEdit: (injury: WorkerInjuryRow) => void;
  onDelete: (injury: WorkerInjuryRow) => void;
}) {
  const t = useT();
  const actions: RowAction[] = [];
  if (canUpdate) {
    actions.push({
      id: "edit",
      label: t("Edit case"),
      icon: Edit02Icon,
      preload: paneIntent.onFocus,
      onSelect: () => onEdit(injury),
    });
  }
  if (canDelete) {
    actions.push({
      id: "delete",
      label: t("Delete case"),
      icon: Trash01Icon,
      destructive: true,
      disabled: deleting,
      onSelect: () => onDelete(injury),
    });
  }
  const days = [
    injury.daysAway > 0
      ? t("{0, plural, one {# day away} other {# days away}}", injury.daysAway)
      : null,
    injury.daysRestricted > 0
      ? t("{0, plural, one {# day restricted} other {# days restricted}}", injury.daysRestricted)
      : null,
  ].filter(Boolean);
  return (
    <div
      data-glide-target
      data-testid={`injury-${injury.id}`}
      className="grid grid-cols-[minmax(0,1fr)_auto] items-start gap-2 rounded-md p-2"
    >
      <button
        type="button"
        disabled={!canUpdate}
        {...(canUpdate ? paneIntent : {})}
        onClick={() => onEdit(injury)}
        className="min-w-0 rounded-sm ui-focus-ring text-left disabled:cursor-default"
      >
        <span className="flex flex-wrap items-center gap-x-2 gap-y-1">
          <span className="font-medium">
            {injury.bodyPart
              ? t("{0}, {1}", illnessTypeLabel(injury.illnessType), injury.bodyPart.toLowerCase())
              : illnessTypeLabel(injury.illnessType)}
          </span>
          <Badge variant={classificationTone(injury.classification)}>
            {caseClassificationLabel(injury.classification)}
          </Badge>
          <Badge variant={injury.status === "Open" ? "warning" : "neutral"} dot>
            {injury.status === "Open" ? t("Open") : t("Closed")}
          </Badge>
          {injury.privacyCase ? <Badge variant="neutral">{t("Privacy case")}</Badge> : null}
        </span>
        <span className="mt-0.5 block text-xs text-muted-foreground tabular-nums">
          {injury.recordable
            ? t(
                "{0} · on the {1} OSHA 300 log",
                formatUnixDateMedium(injury.occurredAt),
                injury.caseYear,
              )
            : t("{0} · kept on file, not on the log", formatUnixDateMedium(injury.occurredAt))}
          {days.length > 0 ? ` · ${days.join(" · ")}` : ""}
        </span>
        <span className="mt-1 block text-pretty">{injury.description}</span>
        {injury.claimStatus !== "NotFiled" ? (
          <span className="mt-1 flex items-center gap-2 text-xs">
            <Badge variant={claimStatusTone(injury.claimStatus)} dot>
              {t("Claim {0}", claimStatusLabel(injury.claimStatus).toLowerCase())}
            </Badge>
            {injury.claimNumber ? (
              <span className="font-mono text-muted-foreground">{injury.claimNumber}</span>
            ) : null}
          </span>
        ) : null}
      </button>
      <RowActionsMenu label={t("Case options")} actions={actions} />
    </div>
  );
}
