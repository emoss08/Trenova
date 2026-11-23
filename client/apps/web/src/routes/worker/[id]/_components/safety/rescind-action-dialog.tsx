import { ReasonDialog } from "@/components/reason-dialog";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  rescindDisciplinaryAction,
  type WorkerDisciplinaryActionRow,
} from "@/lib/graphql/worker-safety";
import { type TranslateFn, useT } from "@trenova/shared/i18n/use-t";
import type { DisciplinaryLevel } from "@trenova/shared/types/worker-safety";
import { useWorkerInvalidation } from "../use-worker-invalidation";

export type RescindActionDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  action: WorkerDisciplinaryActionRow | null;
};

/** Withdraws a disciplinary action with the reason an auditor will read. */
export function RescindActionDialog({
  open,
  onOpenChange,
  workerId,
  action,
}: RescindActionDialogProps) {
  const t = useT();
  const invalidate = useWorkerInvalidation(workerId, "safety");
  const rescind = useApiMutation({
    scope: { id: `worker-discipline-rescind:${workerId}` },
    resourceName: "Disciplinary action",
    mutationFn: ({ target, reason }: { target: WorkerDisciplinaryActionRow; reason: string }) =>
      rescindDisciplinaryAction({ id: target.id, reason, version: target.version }),
    onSuccess: async () => {
      await invalidate();
      onOpenChange(false);
    },
  });

  return (
    <ReasonDialog
      open={open}
      onOpenChange={onOpenChange}
      title={rescindTitle(action?.level as DisciplinaryLevel | undefined, t)}
      description={t(
        "It stays on the record as rescinded and stops counting toward the next step.",
      )}
      placeholder={t("e.g. Issued against the wrong event")}
      fieldDescription={t(
        "Saved on the rescinded row so an auditor can see why the action was withdrawn.",
      )}
      maxLength={255}
      destructive
      confirmLabel={t("Rescind")}
      cancelLabel={t("Cancel")}
      pending={rescind.isPending}
      onConfirm={(reason) => {
        if (!action) return;
        void rescind.mutateAsync({ target: action, reason }).catch(() => undefined);
      }}
    />
  );
}

function rescindTitle(level: DisciplinaryLevel | undefined, t: TranslateFn): string {
  switch (level) {
    case "Coaching":
      return t("Rescind the coaching?");
    case "VerbalWarning":
      return t("Rescind the verbal warning?");
    case "WrittenWarning":
      return t("Rescind the written warning?");
    case "FinalWarning":
      return t("Rescind the final warning?");
    case "Suspension":
      return t("Rescind the suspension?");
    case "Termination":
      return t("Rescind the termination?");
    default:
      return t("Rescind this action?");
  }
}
