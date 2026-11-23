import { RowActionsMenu, type RowAction } from "@/components/row-actions-menu";
import { HoverGlideGroup } from "@/components/hover-glide";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
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
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
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
import { ConfirmDialog } from "@/components/confirm-dialog";

export type InjuryListProps = {
  workerId: string;
  injuries: readonly WorkerInjuryRow[];
  loading: boolean;
  error: unknown;
  onRetry: () => void;
  canRecord: boolean;
  canUpdate: boolean;
  canDelete: boolean;
  onRecord: () => void;
  onEdit: (injury: WorkerInjuryRow) => void;
};

/**
 * The worker's injury and illness cases beside their safety record: each with its type and body
 * part, its OSHA classification and whether it is on the 300 log, and its claim. The log for the
 * whole establishment is the OSHA page; this is the one person's share of it.
 */
export function InjuryList({
  workerId,
  injuries,
  loading,
  error,
  onRetry,
  canRecord,
  canUpdate,
  canDelete,
  onRecord,
  onEdit,
}: InjuryListProps) {
  const t = useT();
  const queryClient = useQueryClient();
  const [deleting, setDeleting] = useState<WorkerInjuryRow | null>(null);

  const remove = useApiMutation({
    scope: { id: `worker-injury-delete:${workerId}` },
    resourceName: "Case",
    mutationFn: (id: string) => deleteWorkerInjury(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: [WORKER_INJURIES_KEY, workerId] });
      void queryClient.invalidateQueries({ queryKey: [OSHA_LOG_KEY] });
    },
  });

  return (
    <SectionPanel
      title={t("Injuries and illnesses")}
      count={injuries.length}
      hint={t("OSHA 300 log")}
      action={
        canRecord && injuries.length > 0 ? (
          <Button size="xs" variant="quiet" onClick={onRecord}>
            {t("Record a case")}
          </Button>
        ) : null
      }
    >
      {loading ? (
        <div aria-busy aria-label={t("Loading the cases")} className="flex flex-col gap-2 p-3">
          <span className="block h-3 w-3/5 ui-shimmer rounded-sm" />
          <span className="block h-3 w-2/5 ui-shimmer rounded-sm" />
        </div>
      ) : error ? (
        <ErrorState
          error={error}
          layout="compact"
          title={t("The cases could not be loaded")}
          onRetry={onRetry}
        />
      ) : injuries.length === 0 ? (
        <div className="flex items-center justify-between gap-2 pr-2">
          <SectionPanelQuiet>{t("No cases on record.")}</SectionPanelQuiet>
          {canRecord ? (
            <Button size="sm" variant="quiet" onClick={onRecord}>
              <PlusIcon className="size-3.5" />
              {t("Record a case")}
            </Button>
          ) : null}
        </div>
      ) : (
        <ScrollArea viewportClassName="max-h-80" maskVariant="card">
          <HoverGlideGroup className="flex flex-col p-1.5">
            {injuries.map((injury) => {
              const actions: RowAction[] = [];
              if (canUpdate) {
                actions.push({
                  id: "edit",
                  label: t("Edit case"),
                  icon: Edit02Icon,
                  onSelect: () => onEdit(injury),
                });
              }
              if (canDelete) {
                actions.push({
                  id: "delete",
                  label: t("Delete case"),
                  icon: Trash01Icon,
                  destructive: true,
                  disabled: remove.isPending,
                  onSelect: () => setDeleting(injury),
                });
              }
              const days = [
                injury.daysAway > 0
                  ? t("{0, plural, one {# day away} other {# days away}}", injury.daysAway)
                  : null,
                injury.daysRestricted > 0
                  ? t(
                      "{0, plural, one {# day restricted} other {# days restricted}}",
                      injury.daysRestricted,
                    )
                  : null,
              ].filter(Boolean);
              return (
                <div
                  key={injury.id}
                  data-glide-target
                  data-testid={`injury-${injury.id}`}
                  className="grid grid-cols-[minmax(0,1fr)_auto] items-start gap-2 rounded-md p-2"
                >
                  <button
                    type="button"
                    disabled={!canUpdate}
                    onClick={() => onEdit(injury)}
                    className="min-w-0 rounded-sm ui-focus-ring text-left disabled:cursor-default"
                  >
                    <span className="flex flex-wrap items-center gap-x-2 gap-y-1">
                      <span className="font-medium">
                        {injury.bodyPart
                          ? t(
                              "{0}, {1}",
                              illnessTypeLabel(injury.illnessType),
                              injury.bodyPart.toLowerCase(),
                            )
                          : illnessTypeLabel(injury.illnessType)}
                      </span>
                      <Badge variant={classificationTone(injury.classification)}>
                        {caseClassificationLabel(injury.classification)}
                      </Badge>
                      <Badge variant={injury.status === "Open" ? "warning" : "neutral"} dot>
                        {injury.status === "Open" ? t("Open") : t("Closed")}
                      </Badge>
                      {injury.privacyCase ? (
                        <Badge variant="neutral">{t("Privacy case")}</Badge>
                      ) : null}
                    </span>
                    <span className="mt-0.5 block text-xs text-muted-foreground tabular-nums">
                      {injury.recordable
                        ? t(
                            "{0} · on the {1} OSHA 300 log",
                            formatUnixDateMedium(injury.occurredAt),
                            injury.caseYear,
                          )
                        : t(
                            "{0} · kept on file, not on the log",
                            formatUnixDateMedium(injury.occurredAt),
                          )}
                      {days.length > 0 ? ` · ${days.join(" · ")}` : ""}
                    </span>
                    <span className="mt-1 block text-pretty">{injury.description}</span>
                    {injury.claimStatus !== "NotFiled" ? (
                      <span className="mt-1 flex items-center gap-2 text-xs">
                        <Badge variant={claimStatusTone(injury.claimStatus)}>
                          {t("Claim {0}", claimStatusLabel(injury.claimStatus).toLowerCase())}
                        </Badge>
                        {injury.claimNumber ? (
                          <span className="font-mono text-muted-foreground">
                            {injury.claimNumber}
                          </span>
                        ) : null}
                      </span>
                    ) : null}
                  </button>
                  <RowActionsMenu label={t("Case options")} actions={actions} />
                </div>
              );
            })}
          </HoverGlideGroup>
        </ScrollArea>
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
        onConfirm={() => (deleting ? remove.mutateAsync(deleting.id) : Promise.resolve(undefined))}
      />
    </SectionPanel>
  );
}
