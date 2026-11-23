import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { useRef, useState, type ReactNode } from "react";

export type ConfirmDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description: ReactNode;
  /** Anything the decision needs beyond its sentence, under the description. */
  children?: ReactNode;
  confirmLabel: string;
  /** The words on the button that backs out; "Cancel" unless that would read as the action. */
  cancelLabel?: string;
  /** `destructive` for a step that removes or ends something; `default` for one that does not. */
  variant?: "default" | "destructive";
  /** A request the caller runs itself is in flight; the dialog holds while it is. */
  pending?: boolean;
  /**
   * Runs the step. When it returns a promise the confirm holds until it settles and refuses a
   * second press; the dialog closes when it succeeds and stays open on a refusal, which the
   * mutation has already reported. When it returns nothing the caller closes the dialog.
   */
  onConfirm: () => Promise<unknown> | undefined;
};

/** Asks before a step that changes the record and cannot simply be taken back. */
export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  children,
  confirmLabel,
  cancelLabel,
  variant = "destructive",
  pending = false,
  onConfirm,
}: ConfirmDialogProps) {
  const t = useT();
  const inFlight = useRef(false);
  const [running, setRunning] = useState(false);
  const busy = pending || running;

  const confirm = () => {
    if (inFlight.current || pending) return;
    const result = onConfirm();
    if (!result) return;
    inFlight.current = true;
    setRunning(true);
    void result
      .then(
        () => onOpenChange(false),
        () => undefined,
      )
      .finally(() => {
        inFlight.current = false;
        setRunning(false);
      });
  };

  return (
    <AlertDialog open={open} onOpenChange={(next) => !busy && onOpenChange(next)}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{title}</AlertDialogTitle>
          <AlertDialogDescription>{description}</AlertDialogDescription>
        </AlertDialogHeader>
        {children}
        <AlertDialogFooter>
          <AlertDialogCancel size="sm" disabled={busy}>
            {cancelLabel ?? t("Cancel")}
          </AlertDialogCancel>
          <Button type="button" size="sm" variant={variant} isLoading={busy} onClick={confirm}>
            {confirmLabel}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
