import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Button } from "@trenova/shared/components/ui/button";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatCountdownClock } from "@trenova/shared/lib/date";
import { useRef } from "react";

/**
 * The warning before an idle sign-out. Closing it any way other than Sign out (the
 * button, Escape) is the person saying they are still there.
 */
export function IdleTimeoutDialog({
  deadline,
  onStay,
  onSignOut,
}: {
  /** When the session will be signed out, in epoch milliseconds. */
  deadline: number;
  onStay: () => void;
  onSignOut: () => void;
}) {
  const t = useT();
  const now = useNow("second");
  const stayRef = useRef<HTMLButtonElement>(null);
  const remaining = formatCountdownClock(deadline / 1000 - now);

  return (
    <AlertDialog
      open
      onOpenChange={(open) => {
        if (!open) {
          onStay();
        }
      }}
    >
      <AlertDialogContent initialFocus={stayRef}>
        <AlertDialogHeader>
          <AlertDialogTitle className="text-lg font-semibold">
            {t("Are you still there?")}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {t(
              "You haven't been active for a while. To keep your account secure, you will be signed out when the timer runs out, and anything you haven't saved will be lost.",
            )}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <p
          role="timer"
          aria-label={t("Time left before you are signed out")}
          className="text-center text-2xl font-semibold text-foreground tabular-nums"
        >
          {remaining}
        </p>
        <AlertDialogFooter>
          <Button variant="outline" onClick={onSignOut}>
            {t("Sign out")}
          </Button>
          <Button ref={stayRef} onClick={onStay}>
            {t("Stay signed in")}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
