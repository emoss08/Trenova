import { cadenceLabel } from "@/lib/schedule-labels";
import type { AssistantThread } from "@/types/assistant";
import { AlertDialog as AlertDialogPrimitive } from "@base-ui/react/alert-dialog";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { useThreadSchedules } from "./thread-schedules";
import "./delete-conversation-card.css";

/**
 * The question asked before a conversation with scheduled requests is
 * deleted, from the Desk's rail or the assistant panel, drawn in the Desk's
 * own hand: the conversation by name, and each request that goes with it,
 * by what it asks and how often. Open while `thread` is set.
 */
export function DeleteConversationCard({
  thread,
  pending = false,
  onConfirm,
  onClose,
}: {
  thread: AssistantThread | null;
  /** The delete is on its way, so it cannot be asked for twice. */
  pending?: boolean;
  onConfirm: (thread: AssistantThread) => void;
  onClose: () => void;
}) {
  const t = useT();

  return (
    <AlertDialogPrimitive.Root open={thread !== null} onOpenChange={(next) => !next && onClose()}>
      <AlertDialogPrimitive.Portal>
        <AlertDialogPrimitive.Backdrop className="dk-dcf-scrim" />
        <AlertDialogPrimitive.Popup className="dk-dcf">
          <AlertDialogPrimitive.Title className="dk-dcf-t">
            {t("Delete “{0}”?", thread?.title || t("Untitled conversation"))}
          </AlertDialogPrimitive.Title>
          {thread && <GoesWithIt threadId={thread.id} />}
          <div className="dk-dcf-f">
            <AlertDialogPrimitive.Close
              render={
                <Button
                  variant="bare"
                  size="bare"
                  className="h-8 rounded-lg px-3 text-sm font-medium text-dsk-muted transition-colors duration-140 hover:bg-dsk-hover hover:text-dsk-fg"
                />
              }
            >
              {t("Cancel")}
            </AlertDialogPrimitive.Close>
            <Button
              variant="bare"
              size="bare"
              className="h-8 rounded-lg bg-danger px-3 text-sm font-medium text-dsk-on-solid transition-colors duration-150 hover:bg-danger-hover disabled:opacity-50"
              disabled={pending}
              onClick={() => thread && onConfirm(thread)}
            >
              {t("Delete")}
            </Button>
          </div>
        </AlertDialogPrimitive.Popup>
      </AlertDialogPrimitive.Portal>
    </AlertDialogPrimitive.Root>
  );
}

/**
 * Each request scheduled in the conversation, by what it asks and how often;
 * a plain line when they could not be read.
 */
function GoesWithIt({ threadId }: { threadId: string }) {
  const t = useT();
  const schedules = useThreadSchedules(threadId);
  if (schedules === null || schedules.length === 0) {
    return (
      <AlertDialogPrimitive.Description className="dk-dcf-d">
        {t("Its scheduled requests go with it.")}
      </AlertDialogPrimitive.Description>
    );
  }

  return (
    <>
      <AlertDialogPrimitive.Description className="dk-dcf-d">
        {t("These schedules go with it:")}
      </AlertDialogPrimitive.Description>
      <ul className="dk-dcf-l">
        {schedules.map((schedule) => (
          <li key={schedule.id} className="dk-dcf-r">
            <DeskIcon name="clock" size={14} />
            <span>
              <b>{schedule.prompt}</b>
              <em>{cadenceLabel(schedule, t)}</em>
            </span>
          </li>
        ))}
      </ul>
    </>
  );
}
