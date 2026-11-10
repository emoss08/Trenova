import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { useT } from "@trenova/shared/i18n/use-t";
import { useId, useState } from "react";

/** The longest reason the server keeps with a decision. */
const MAX_REASON = 2000;

/**
 * Declining a change asks why, because the agent reads the reason in its
 * conversation and so does whoever looks after it. The change never runs.
 */
export function DeclineDialog({
  open,
  agentName,
  pending,
  onOpenChange,
  onDecline,
}: {
  open: boolean;
  agentName: string | null;
  pending: boolean;
  onOpenChange: (open: boolean) => void;
  onDecline: (reason: string) => Promise<void>;
}) {
  const t = useT();
  const id = useId();
  const [reason, setReason] = useState("");
  const [error, setError] = useState("");

  const submit = async () => {
    const trimmed = reason.trim();
    if (trimmed === "") {
      setError(t("Say why. The agent and whoever looks after it read this."));
      return;
    }
    await onDecline(trimmed);
    setReason("");
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <form
          className="flex flex-col gap-4"
          onSubmit={(event) => {
            event.preventDefault();
            void submit();
          }}
        >
          <DialogHeader>
            <DialogTitle>{t("Decline this change?")}</DialogTitle>
            <DialogDescription>
              {agentName
                ? t(
                    "A declined change never runs. {0} reads your reason in the conversation, and so does whoever looks after the agent.",
                    agentName,
                  )
                : t(
                    "A declined change never runs. The agent reads your reason, and so does whoever looks after it.",
                  )}
            </DialogDescription>
          </DialogHeader>
          <div className="flex flex-col gap-1.5">
            <label htmlFor={id} className="text-xs font-medium text-foreground-muted">
              {t("Reason")}
            </label>
            <Textarea
              id={id}
              value={reason}
              maxLength={MAX_REASON}
              minRows={3}
              isInvalid={error !== ""}
              aria-describedby={error !== "" ? `${id}-err` : undefined}
              onChange={(event) => {
                setReason(event.target.value);
                setError("");
              }}
            />
            {error !== "" && (
              <span id={`${id}-err`} role="alert" className="text-xs text-danger-foreground">
                {error}
              </span>
            )}
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              {t("Cancel")}
            </Button>
            <Button type="submit" isLoading={pending}>
              {t("Decline")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
