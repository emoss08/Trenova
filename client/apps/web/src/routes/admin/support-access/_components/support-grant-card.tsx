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
  canExtend,
  onRevoke,
  onExtend,
}: {
  grant: SupportAccessGrant;
  canRevoke: boolean;
  canExtend: boolean;
  onRevoke: () => void;
  onExtend: () => void;
}) {
  const t = useT();
  const [showActions, setShowActions] = useState(false);
  const live = grant.status === "Live";

  return (
    <div className="flex flex-col rounded-lg border border-border bg-card">
      <div className="flex items-start justify-between gap-3 p-4">
        <div className="flex min-w-0 flex-col gap-1">
          <div className="flex flex-wrap items-center gap-2">
            <span className="truncate font-medium text-foreground">{grant.staffEmail}</span>
            <Badge variant={statusVariant(grant.status)} dot>
              {statusLabel(t, grant.status)}
            </Badge>
            <Badge variant={grant.scope === "Full" ? "warning" : "neutral"}>
              {scopeLabel(t, grant.scope)}
            </Badge>
            {grant.allowsAssistant ? (
              <Badge variant="neutral">{t("May use the assistant")}</Badge>
            ) : null}
            {live && grant.activeAt != null ? (
              <Badge variant="info" dot>
                {t("In your organization now")}
              </Badge>
            ) : null}
          </div>
          <p className="text-sm text-foreground-muted">{grant.reason}</p>
          <p className="text-xs text-foreground-subtle">
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
          {grant.extendedAt != null ? (
            <p className="text-xs text-foreground-subtle">
              {t(
                "Extended by {0} on {1}",
                grant.extendedByName ?? t("an administrator"),
                formatUnixDateTime(grant.extendedAt),
              )}
              {live ? (
                <>
                  {" · "}
                  {t("Staff enter again to use the added time")}
                </>
              ) : null}
            </p>
          ) : null}
          {grant.lastEnteredAt != null ? (
            <p className="text-xs text-foreground-subtle">
              {t(
                "Last entered by {0} on {1}",
                grant.lastStaffName ?? grant.staffEmail,
                formatUnixDateTime(grant.lastEnteredAt),
              )}
            </p>
          ) : null}
        </div>
        {live && (canExtend || canRevoke) ? (
          <div className="flex shrink-0 items-center gap-2">
            {canExtend ? (
              <Button type="button" size="sm" variant="secondary" onClick={onExtend}>
                {t("Extend")}
              </Button>
            ) : null}
            {canRevoke ? (
              <Button type="button" size="sm" variant="destructive" onClick={onRevoke}>
                {t("End access")}
              </Button>
            ) : null}
          </div>
        ) : null}
      </div>
      <div className="border-t border-border-subtle px-4 py-2">
        <Button
          type="button"
          size="sm"
          variant="ghost"
          aria-expanded={showActions}
          onClick={() => setShowActions((open) => !open)}
        >
          {showActions ? t("Hide what support did") : t("Show what support did")}
        </Button>
        {showActions ? <SupportGrantActions grant={grant} /> : null}
      </div>
    </div>
  );
}
