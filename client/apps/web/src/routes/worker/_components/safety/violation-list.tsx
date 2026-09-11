import { useT } from "@trenova/shared/i18n/use-t";
import { usePermission } from "@/hooks/use-permission";
import {
  deleteSafetyViolation,
  fetchSafetyViolations,
  SAFETY_VIOLATIONS_KEY,
  type SafetyViolationRow,
} from "@/lib/graphql/fleet-safety";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { csaBasicLabel } from "@trenova/shared/lib/csa";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { PlusIcon, Trash2Icon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { ViolationDialog } from "./violation-dialog";

/**
 * The violations cited on one safety event. A roadside inspection routinely
 * produces several, in different CSA BASICs, so they are rows rather than a
 * field on the event — and the fleet scorecard is only as real as these are.
 */
export function ViolationList({
  safetyEventId,
  suggestedBasic,
}: {
  safetyEventId: string;
  suggestedBasic: string | null;
}) {
  const t = useT();

  const queryClient = useQueryClient();
  const { allowed: canRead } = usePermission(Resource.WorkerSafetyEvent, Operation.Read);
  const { allowed: canRecord } = usePermission(Resource.WorkerSafetyEvent, Operation.Create);
  const { allowed: canDelete } = usePermission(Resource.WorkerSafetyEvent, Operation.Delete);
  const [dialog, setDialog] = useState<{ violation: SafetyViolationRow | null } | null>(null);

  const violationsQuery = useQuery({
    queryKey: [SAFETY_VIOLATIONS_KEY, safetyEventId],
    queryFn: ({ signal }) => fetchSafetyViolations({ safetyEventId }, { signal }),
    enabled: canRead,
  });

  const removeMutation = useMutation({
    mutationFn: (id: string) => deleteSafetyViolation(id),
    onSuccess: () => {
      toast.success(t("Violation removed"));
      void queryClient.invalidateQueries({ queryKey: [SAFETY_VIOLATIONS_KEY, safetyEventId] });
      void queryClient.invalidateQueries({ queryKey: ["fleet-safety"] });
    },
    onError: (error: Error) =>
      toast.error(t("Could not remove the violation"), { description: error.message }),
  });

  if (!canRead) return null;

  const violations = violationsQuery.data ?? [];

  return (
    <div className="border-border/60 mt-1 border-t pt-2">
      <div className="flex items-center justify-between gap-2">
        <span className="text-muted-foreground text-[11px] font-medium">
          {t("Violations cited {0}", violations.length > 0 ? ` (${violations.length})` : "")}
        </span>
        {canRecord ? (
          <Button
            size="xs"
            variant="ghost"
            onClick={() => setDialog({ violation: null })}
            aria-label={t("Cite a violation")}
          >
            <PlusIcon className="size-3" />
            {t("Cite one")}
          </Button>
        ) : null}
      </div>

      {violations.length === 0 ? (
        <p className="text-muted-foreground mt-1 text-[11px]">
          {t("None keyed in. The fleet scorecard falls back to the kind of event, which is an estimate rather than what the inspection actually said.")}
        </p>
      ) : (
        <ul className="mt-1 flex flex-col gap-1">
          {violations.map((violation) => (
            <li key={violation.id} className="flex items-center justify-between gap-2 text-[11px]">
              <span className="flex min-w-0 items-center gap-1.5">
                <Badge variant="secondary">{csaBasicLabel(violation.basic)}</Badge>
                {violation.code ? (
                  <span className="text-muted-foreground shrink-0 tabular-nums">
                    {violation.code}
                  </span>
                ) : null}
                <span className="truncate">{t(violation.description)}</span>
                {violation.outOfService ? <Badge variant="inactive">{t("OOS")}</Badge> : null}
              </span>
              <span className="flex shrink-0 items-center gap-1">
                <span className="text-muted-foreground tabular-nums">
                  {t("weight {0}", violation.severityWeight)}
                </span>
                {canRecord ? (
                  <Button
                    size="xxs"
                    variant="ghost"
                    onClick={() => setDialog({ violation })}
                    aria-label={`Edit ${violation.description}`}
                  >
                    {t("Edit")}
                  </Button>
                ) : null}
                {canDelete ? (
                  <Button
                    size="xxs"
                    variant="ghost"
                    className="text-destructive"
                    isLoading={removeMutation.isPending}
                    onClick={() => removeMutation.mutate(violation.id)}
                    aria-label={`Remove ${violation.description}`}
                  >
                    <Trash2Icon className="size-3" />
                  </Button>
                ) : null}
              </span>
            </li>
          ))}
        </ul>
      )}

      <ViolationDialog
        open={dialog !== null}
        onOpenChange={(open) => !open && setDialog(null)}
        safetyEventId={safetyEventId}
        suggestedBasic={suggestedBasic}
        violation={dialog?.violation ?? null}
      />
    </div>
  );
}
