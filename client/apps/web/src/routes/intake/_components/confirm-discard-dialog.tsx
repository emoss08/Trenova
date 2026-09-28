import { errorCopy } from "@trenova/shared/components/errors/error-copy";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { useT } from "@trenova/shared/i18n/use-t";
import { describeError } from "@trenova/shared/lib/error-presentation";
import { Trash2Icon } from "lucide-react";
import { useState, type ReactNode } from "react";

/**
 * Asks before something is thrown away, and stays until the answer is in: it
 * closes when the discard succeeds and, when it fails, says why in place so
 * the person can try again or keep what they have.
 */
export function ConfirmDiscardDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel,
  failureTitle,
  onConfirm,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description: ReactNode;
  confirmLabel: string;
  /** Headlines the reason the discard failed: "The stack was not discarded". */
  failureTitle: string;
  onConfirm: () => Promise<unknown>;
}) {
  const t = useT();
  const [pending, setPending] = useState(false);
  const [failure, setFailure] = useState<unknown>(null);

  const changeOpen = (next: boolean) => {
    if (pending) {
      return;
    }
    if (!next) {
      setFailure(null);
    }
    onOpenChange(next);
  };

  const confirm = async () => {
    setPending(true);
    setFailure(null);
    try {
      await onConfirm();
      setPending(false);
      onOpenChange(false);
    } catch (error) {
      setPending(false);
      setFailure(error);
    }
  };

  const failureCopy = failure === null ? null : errorCopy(t, describeError(failure));

  return (
    <AlertDialog open={open} onOpenChange={changeOpen}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogMedia className="bg-danger-subtle text-destructive">
            <Trash2Icon />
          </AlertDialogMedia>
          <AlertDialogTitle>{title}</AlertDialogTitle>
          <AlertDialogDescription>{description}</AlertDialogDescription>
        </AlertDialogHeader>
        {failureCopy !== null && (
          <Alert variant="destructive" size="sm">
            <AlertTitle>{failureTitle}</AlertTitle>
            <AlertDescription>{failureCopy.description}</AlertDescription>
          </Alert>
        )}
        <AlertDialogFooter>
          <AlertDialogCancel disabled={pending}>{t("Keep it")}</AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            onClick={() => void confirm()}
            isLoading={pending}
            loadingText={t("Discarding")}
          >
            {failure === null ? confirmLabel : t("Try again")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
