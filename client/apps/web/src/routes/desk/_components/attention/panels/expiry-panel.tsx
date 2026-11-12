import type { WatchtowerExpiryPanel } from "@/lib/graphql/watchtower";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatFactTime } from "../../watchtower/lanes";
import { PanelBody, PanelCard } from "./panel-parts";

/** Days inside which a credential coming due is a warning rather than a note. */
const SOON_DAYS = 14;

function due(t: TranslateFn, days: number): string {
  if (days < 0) {
    return t("{0, plural, one {Expired # day ago} other {Expired # days ago}}", -days);
  }
  if (days === 0) {
    return t("Expires today");
  }

  return t("{0, plural, one {# day left} other {# days left}}", days);
}

/**
 * One person's credentials coming due, soonest first: each paper, the day it
 * runs out and how long is left, so a renewal is one read and one packet.
 */
export function ExpiryPanel({
  expiry,
  now,
  timezone,
}: {
  expiry: WatchtowerExpiryPanel;
  now: number;
  timezone: string;
}) {
  const t = useT();
  const papers = [...expiry.papers].sort((a, b) => a.daysLeft - b.daysLeft);
  const expired = papers.filter((paper) => paper.daysLeft < 0).length;

  return (
    <PanelCard
      eyebrow={t("Credentials")}
      title={expiry.who}
      badge={
        expired > 0 ? (
          <Badge variant="danger" dot>
            {t("{0, plural, one {# expired} other {# expired}}", expired)}
          </Badge>
        ) : (
          <Badge variant="warning" dot>
            {t("{0, plural, one {# expiring} other {# expiring}}", papers.length)}
          </Badge>
        )
      }
    >
      <PanelBody>
        <ul className="flex flex-col divide-y divide-dsk-b-sub">
          {papers.map((paper) => (
            <li key={paper.name} className="flex items-center gap-3 py-1.5 text-dsk-base">
              <span className="min-w-0 flex-1 truncate text-dsk-fg">{paper.name}</span>
              {paper.expiresAt > 0 && (
                <span className="text-xs text-dsk-subtle tabular-nums">
                  {formatFactTime(t, paper.expiresAt, now, timezone)}
                </span>
              )}
              <Badge
                variant={
                  paper.daysLeft < 0
                    ? "danger"
                    : paper.daysLeft <= SOON_DAYS
                      ? "warning"
                      : "neutral"
                }
              >
                {due(t, paper.daysLeft)}
              </Badge>
            </li>
          ))}
        </ul>
      </PanelBody>
    </PanelCard>
  );
}
