import { ReasonDialog } from "@/components/reason-dialog";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { closeWorkerSafetyEvent, type WorkerSafetyEventRow } from "@/lib/graphql/worker-safety";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { useWorkerInvalidation } from "../use-worker-invalidation";

export type CloseEventDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  event: WorkerSafetyEventRow | null;
};

/**
 * Closes a safety event with what was done about it. The field starts from the event's current
 * resolution at each opening, and a refetch while it is open never replaces what is typed.
 */
export function CloseEventDialog({ open, onOpenChange, workerId, event }: CloseEventDialogProps) {
  const t = useT();
  const invalidate = useWorkerInvalidation(workerId, "safety");
  const close = useApiMutation({
    scope: { id: `worker-safety-status:${workerId}` },
    resourceName: "Safety event",
    mutationFn: ({ target, resolution }: { target: WorkerSafetyEventRow; resolution: string }) =>
      closeWorkerSafetyEvent({ id: target.id, resolution, version: target.version }),
    onSuccess: async () => {
      await invalidate();
      onOpenChange(false);
    },
  });

  return (
    <ReasonDialog
      open={open}
      onOpenChange={onOpenChange}
      title={t("Close this event?")}
      description={
        event?.pointsExpireAt && event.activePoints > 0
          ? t(
              "It stays on the record. Its points keep counting until {0}.",
              formatUnixDateMedium(event.pointsExpireAt),
            )
          : t("It stays on the record. Say what was done about it.")
      }
      label={t("Resolution")}
      placeholder={t("e.g. Coaching on speed management, no further action")}
      fieldDescription={t(
        "Kept on the event and in the audit log; a closed event cannot be deleted without reopening it first.",
      )}
      defaultReason={event?.resolution ?? ""}
      maxLength={4000}
      confirmLabel={t("Close event")}
      cancelLabel={t("Cancel")}
      pending={close.isPending}
      onConfirm={(resolution) => {
        if (!event) return;
        void close.mutateAsync({ target: event, resolution }).catch(() => undefined);
      }}
    />
  );
}
