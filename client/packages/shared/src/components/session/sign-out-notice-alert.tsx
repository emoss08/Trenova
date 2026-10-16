import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { useT } from "@trenova/shared/i18n/use-t";
import type { SignOutNotice } from "@trenova/shared/lib/sign-out-notice";

/** Tells someone on the sign-in page why they were signed out when they did not choose to be. */
export function SignOutNoticeAlert({ notice }: { notice: SignOutNotice }) {
  const t = useT();

  return (
    <Alert variant="info" size="sm" role="status">
      <AlertDescription>
        {notice === "idle"
          ? t("You were signed out after a period of inactivity. Sign in again to continue.")
          : t("Your session has ended. Sign in again to continue.")}
      </AlertDescription>
    </Alert>
  );
}
