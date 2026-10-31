import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import type { SignOutNotice } from "@trenova/shared/lib/sign-out-notice";

function noticeText(notice: SignOutNotice, t: TranslateFn): string {
  switch (notice) {
    case "idle":
      return t("You were signed out after a period of inactivity. Sign in again to continue.");
    case "elsewhere":
      return t("You signed out in another tab. Sign in again to continue.");
    case "expired":
      return t("Your session has ended. Sign in again to continue.");
  }
}

/** Tells someone on the sign-in page why they were signed out when they did not choose to be. */
export function SignOutNoticeAlert({ notice }: { notice: SignOutNotice }) {
  const t = useT();

  return (
    <Alert variant="info" size="sm" role="status">
      <AlertDescription>{noticeText(notice, t)}</AlertDescription>
    </Alert>
  );
}
