import type { SupportAccessGrant } from "@/lib/graphql/support-access";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import { useState } from "react";
import { SupportGrantActions } from "./support-grant-actions";
import { durationLabel, scopeLabel, statusLabel, statusVariant } from "./support-access-labels";

/** One grant: who it lets in, why, for how long, who gave it, and what was done under it. */
export function SupportGrantCard({
  grant,
  canRevoke,
  onRevoke,
}: {
  grant: SupportAccessGrant;
  canRevoke: boolean;
  onRevoke: () => void;
}) {
  const t = useT();
  const [showActions, setShowActions] = useState(false);
  const live = grant.status === "Live";

  return (
    <div className="bg-card border-border flex flex-col rounded-lg border">
      <div className="flex items-start justify-between gap-3 p-4">
        <div className="flex min-w-0 flex-col gap-1">
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-foreground truncate font-medium">{grant.staffEmail}</span>
            <Badge variant={statusVariant(grant.status)} dot>
              {statusLabel(t, grant.status)}
            </Badge>
            <Badge variant={grant.scope === "Full" ? "warning" : "neutral"}>
              {scopeLabel(t, grant.scope)}
            </Badge>
          </div>
          <p className="text-foreground-muted text-sm">{grant.reason}</p>
          <p className="text-foreground-subtle text-xs">
            {t(
              "Granted by {0} on {1} for {2}",
              grant.grantedByName ?? t("an administrator"),
              formatUnixDateTime(grant.grantedAt),
              durationLabel(t, grant.durationHours),
            )}
            {" · "}
            {live
              ? t("Ends {0}", formatUnixDateTime(grant.expiresAt))
              : grant.revokedAt != null
                ? t(
                    "Ended by {0} on {1}: {2}",
                    grant.revokedByName ?? t("an administrator"),
                    formatUnixDateTime(grant.revokedAt),
                    grant.revokeReason ?? "",
                  )
                : t("Ended on its own {0}", formatUnixDateTime(grant.expiresAt))}
          </p>
          {grant.lastEnteredAt != null ? (
            <p className="text-foreground-subtle text-xs">
              {t(
                "Last entered by {0} on {1}",
                grant.lastStaffName ?? grant.staffEmail,
                formatUnixDateTime(grant.lastEnteredAt),
              )}
            </p>
          ) : null}
        </div>
        {live && canRevoke ? (
          <Button type="button" size="sm" variant="destructive" onClick={onRevoke}>
            {t("End access")}
          </Button>
        ) : null}
      </div>
      <div className="border-border-subtle border-t px-4 py-2">
        <Button
          type="button"
          size="sm"
          variant="ghost"
          aria-expanded={showActions}
          onClick={() => setShowActions((open) => !open)}
        >
          {showActions ? t("Hide what support did") : t("Show what support did")}
        </Button>
        {showActions ? <SupportGrantActions grantId={grant.id} /> : null}
      </div>
    </div>
  );
}
