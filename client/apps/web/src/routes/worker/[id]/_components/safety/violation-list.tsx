import { preloadOn } from "@/components/work-pane";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useOpenedOnce } from "@/hooks/use-opened-once";
import { usePermission } from "@/hooks/use-permission";
import { lazyComponent } from "@trenova/shared/lib/lazy-component";
import { deleteSafetyViolation, type SafetyViolationRow } from "@/lib/graphql/fleet-safety";
import type { WorkerSafetyEventRow } from "@/lib/graphql/worker-safety";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { Edit02Icon, PlusIcon, Trash01Icon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { csaBasicLabel } from "@trenova/shared/lib/csa";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useState } from "react";

const ViolationDialog = lazyComponent(
  () => import("./violation-dialog"),
  (module) => module.ViolationDialog,
);

/**
 * The violations cited on one safety event, as the read pane shows them: each one with its BASIC,
 * code, weight and out-of-service flag, and the means to cite, correct or remove one. A roadside
 * inspection routinely cites several, and the fleet scorecard is only as real as these rows.
 */
export function ViolationList({
  event,
  onChanged,
}: {
  event: WorkerSafetyEventRow;
  onChanged: () => Promise<unknown>;
}) {
  const t = useT();
  const { allowed: canRecord } = usePermission(Resource.WorkerSafetyEvent, Operation.Create);
  const { allowed: canDelete } = usePermission(Resource.WorkerSafetyEvent, Operation.Delete);
  const [dialog, setDialog] = useState<{ violation: SafetyViolationRow | null } | null>(null);
  const opened = useOpenedOnce(dialog !== null);
  const violations = (event.violations ?? []) as SafetyViolationRow[];
  const suggestedBasic =
    event.kind === "Citation"
      ? "UnsafeDriving"
      : event.inspectionResult === "Pass"
        ? null
        : "VehicleMaintenance";

  const remove = useApiMutation({
    scope: { id: `safety-violation-remove:${event.id}` },
    resourceName: "Violation",
    mutationFn: (id: string) => deleteSafetyViolation(id),
    onSuccess: () => void onChanged(),
  });

  return (
    <div className="flex flex-col gap-2">
      {violations.length === 0 ? (
        <p className="text-xs text-muted-foreground">
          {t(
            "None keyed in. The fleet scorecard falls back to the kind of event, which is an estimate rather than what the inspection actually said.",
          )}
        </p>
      ) : (
        <ul className="divide-y divide-border-subtle rounded-md border border-border-subtle bg-sunken">
          {violations.map((violation) => (
            <li
              key={violation.id}
              className="flex items-start justify-between gap-2 px-2.5 py-1.5 text-xs"
            >
              <span className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
                <Badge variant="neutral">{csaBasicLabel(violation.basic)}</Badge>
                {violation.code ? <span className="font-mono">{violation.code}</span> : null}
                <span className="min-w-0">{violation.description}</span>
                {violation.outOfService ? (
                  <Badge variant="danger" dot>
                    {t("Out of service")}
                  </Badge>
                ) : null}
                <span className="text-muted-foreground tabular-nums">
                  {t("weight {0}", violation.severityWeight)}
                </span>
              </span>
              <span className="flex shrink-0 items-center">
                {canRecord ? (
                  <Button
                    size="icon-xs"
                    variant="ghost"
                    aria-label={t("Correct {0}", violation.description)}
                    {...preloadOn(ViolationDialog)}
                    onClick={() => setDialog({ violation })}
                  >
                    <Edit02Icon className="size-3" />
                  </Button>
                ) : null}
                {canDelete ? (
                  <Button
                    size="icon-xs"
                    variant="ghost"
                    aria-label={t("Remove {0}", violation.description)}
                    isLoading={remove.isPending && remove.variables === violation.id}
                    disabled={remove.isPending}
                    onClick={() => void remove.mutateAsync(violation.id).catch(() => undefined)}
                  >
                    <Trash01Icon className="size-3" />
                  </Button>
                ) : null}
              </span>
            </li>
          ))}
        </ul>
      )}
      {canRecord ? (
        <Button
          size="xs"
          variant="quiet"
          className="self-start"
          {...preloadOn(ViolationDialog)}
          onClick={() => setDialog({ violation: null })}
        >
          <PlusIcon className="size-3.5" />
          {t("Cite a violation")}
        </Button>
      ) : null}
      {opened ? (
        <LazyComponent fallback={null}>
          <ViolationDialog.Component
            open={dialog !== null}
            onOpenChange={(open) => !open && setDialog(null)}
            safetyEventId={event.id}
            suggestedBasic={suggestedBasic}
            violation={dialog?.violation ?? null}
            onSaved={onChanged}
          />
        </LazyComponent>
      ) : null}
    </div>
  );
}
