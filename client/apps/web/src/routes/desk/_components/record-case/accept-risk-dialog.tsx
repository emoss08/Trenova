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

/** The longest reason the server keeps with an accepted risk. */
const MAX_REASON = 500;
const MIN_REASON = 3;

/**
 * Accepting the risk on a record: nothing changes, and the conditions the
 * Watchtower holds about it are marked as needing no action. It asks why,
 * because the reason is what the team reads when the load does arrive late.
 */
export function AcceptRiskDialog({
  open,
  label,
  pending,
  failure,
  onOpenChange,
  onAccept,
}: {
  open: boolean;
  label: string;
  pending: boolean;
  failure: string;
  onOpenChange: (open: boolean) => void;
  onAccept: (reason: string) => Promise<void>;
}) {
  const t = useT();
  const id = useId();
  const [reason, setReason] = useState("");
  const [error, setError] = useState("");

  const submit = async () => {
    const trimmed = reason.trim();
    if (trimmed.length < MIN_REASON) {
      setError(t("Say why, in a sentence the team can read."));
      return;
    }
    await onAccept(trimmed);
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
            <DialogTitle>{t("Accept the risk on {0}?", label)}</DialogTitle>
            <DialogDescription>
              {t(
                "Nothing changes on the record. What the Watchtower holds about it is marked as needing no action and leaves Needs you, with your reason beside it.",
              )}
            </DialogDescription>
          </DialogHeader>
          <div className="flex flex-col gap-1.5">
            <label htmlFor={id} className="text-xs font-medium text-foreground-muted">
              {t("Why is this acceptable?")}
            </label>
            <Textarea
              id={id}
              value={reason}
              maxLength={MAX_REASON}
              minRows={3}
              isInvalid={error !== "" || failure !== ""}
              aria-describedby={error !== "" || failure !== "" ? `${id}-err` : undefined}
              onChange={(event) => {
                setReason(event.target.value);
                setError("");
              }}
            />
            {(error !== "" || failure !== "") && (
              <span id={`${id}-err`} role="alert" className="text-xs text-danger-foreground">
                {error !== "" ? error : failure}
              </span>
            )}
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              {t("Cancel")}
            </Button>
            <Button type="submit" isLoading={pending}>
              {t("Accept the risk")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
