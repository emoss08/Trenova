import { useSignOut } from "@/hooks/use-sign-out";
import { supportAccess } from "@/lib/queries/support-access";
import { useQuery } from "@tanstack/react-query";
import { Alert, AlertAction, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import { useState } from "react";

/**
 * Says, on every page of a Trenova support session, that it is one, whose it is and when it
 * ends, so the staff member always knows and every screenshot shows it. A session that is not
 * Trenova support's draws nothing. It is fixed to the bottom of the window in every frame.
 */
export function SupportSessionBanner() {
  const t = useT();
  const signOut = useSignOut();
  const [ending, setEnding] = useState(false);
  const { data: support } = useQuery({
    ...supportAccess.currentSession(),
    staleTime: Number.POSITIVE_INFINITY,
  });

  if (support == null) {
    return null;
  }

  return (
    <Alert
      variant="info"
      size="sm"
      role="status"
      className="fixed inset-x-0 bottom-0 z-40 rounded-none border-x-0 border-b-0"
    >
      <AlertDescription>
        {support.scope === "Full"
          ? t(
              "Trenova support session: {0}, full access, ends {1}",
              support.staffName,
              formatUnixDateTime(support.endsAt),
            )
          : t(
              "Trenova support session: {0}, read-only, ends {1}",
              support.staffName,
              formatUnixDateTime(support.endsAt),
            )}
      </AlertDescription>
      <AlertAction>
        <Button
          type="button"
          size="xs"
          isLoading={ending}
          onClick={async () => {
            if (ending) {
              return;
            }
            setEnding(true);
            try {
              await signOut();
            } finally {
              setEnding(false);
            }
          }}
        >
          {t("End session")}
        </Button>
      </AlertAction>
    </Alert>
  );
}
