import { SectionPanelLink } from "@/components/section-panel";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import { organization } from "@/lib/queries/organization";
import { useQuery } from "@tanstack/react-query";
import { ActivityIcon, Lock01Icon } from "@trenova/shared/components/icons";
import { Avatar, AvatarFallback } from "@trenova/shared/components/ui/avatar";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatCompactAge, formatUnixDateTimeOrDash } from "@trenova/shared/lib/date";
import { cn, getNameInitials } from "@trenova/shared/lib/utils";
import type { SecurityBriefing } from "@trenova/shared/types/iam";
import type { ReactNode } from "react";
import { SecuritySection } from "../shared";
import { recentActivity, type ActivityTone } from "../activity-copy";
import { securityTabHref, USERS_PAGE } from "./finding-copy";

const RECENT_ACTIVITY_LIMIT = 6;
const NOW_TICK_MS = 60_000;

const TONE_DOT: Record<ActivityTone, string> = {
  success: "bg-success",
  warning: "bg-warning",
  danger: "bg-danger",
  neutral: "bg-foreground-subtle",
};

/** One quiet line for a section with nothing to list. */
function SectionQuiet({ children }: { children: ReactNode }) {
  return <p className="text-muted-foreground py-3 text-xs">{children}</p>;
}

/**
 * The row both lists in the side column are drawn with, so they read as one column: a lead
 * (an avatar or a dot), a line and a quieter line under it, and how long ago at the far end.
 */
function AsideRow({
  lead,
  title,
  detail,
  meta,
  metaTitle,
}: {
  lead: ReactNode;
  title: string;
  detail: string;
  meta: string;
  metaTitle?: string;
}) {
  return (
    <li className="flex items-center gap-3 py-2">
      <span className="flex size-6 shrink-0 items-center justify-center">{lead}</span>
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm">{title}</p>
        {detail !== "" ? (
          <p className="text-foreground-subtle truncate text-xs">{detail}</p>
        ) : null}
      </div>
      <span
        title={metaTitle}
        className="text-foreground-subtle shrink-0 text-xs tabular-nums"
      >
        {meta}
      </span>
    </li>
  );
}

function AsideRowsSkeleton({ rows, label }: { rows: number; label: string }) {
  return (
    <div aria-busy aria-label={label} className="divide-border-subtle divide-y">
      {Array.from({ length: rows }, (_, row) => (
        <div key={row} aria-hidden className="flex items-center gap-3 py-2">
          <Skeleton className="size-6 shrink-0 rounded-md" />
          <div className="flex flex-1 flex-col gap-1.5">
            <Skeleton className="h-3.5 w-2/3" />
            <Skeleton className="h-3 w-1/3" />
          </div>
          <Skeleton className="h-3 w-6" />
        </div>
      ))}
    </div>
  );
}

/**
 * Who has not turned on two-factor, most recently active first, since they are the accounts most
 * in use. The server names at most a page of them; the rest are counted so the list never claims
 * to be complete when it is not.
 */
export function UnenrolledMembersPanel({
  mfa,
  canOpenUsers,
}: {
  mfa: NonNullable<SecurityBriefing["mfa"]>;
  canOpenUsers: boolean;
}) {
  const t = useT();
  const now = useNowSeconds(NOW_TICK_MS);
  const unenrolledCount = Math.max(0, mfa.total - mfa.enrolled);

  if (unenrolledCount === 0) {
    return null;
  }

  const shown = mfa.unenrolled ?? [];
  const hidden = unenrolledCount - shown.length;

  return (
    <SecuritySection
      title={t("Without two-factor")}
      icon={<Lock01Icon />}
      count={unenrolledCount}
      action={
        canOpenUsers ? <SectionPanelLink to={USERS_PAGE}>{t("Users")}</SectionPanelLink> : null
      }
    >
      {mfa.unenrolled === null ? (
        <SectionQuiet>{t("Their names are shown to people who can read users.")}</SectionQuiet>
      ) : (
        <>
          <ul className="divide-border-subtle divide-y">
            {shown.map((member) => (
              <AsideRow
                key={member.userId}
                lead={
                  <Avatar size="sm">
                    <AvatarFallback>{getNameInitials(member.name)}</AvatarFallback>
                  </Avatar>
                }
                title={member.name}
                detail={member.emailAddress}
                meta={
                  member.lastLoginAt === null
                    ? t("Never")
                    : formatCompactAge(now - member.lastLoginAt)
                }
                metaTitle={
                  member.lastLoginAt === null
                    ? t("Never signed in")
                    : t("Last signed in {0}", formatUnixDateTimeOrDash(member.lastLoginAt))
                }
              />
            ))}
          </ul>
          {hidden > 0 ? (
            <p className="text-foreground-subtle border-border-subtle border-t py-2 text-xs">
              {t("{0, plural, one {and # more} other {and # more}}", hidden)}
            </p>
          ) : null}
        </>
      )}
    </SecuritySection>
  );
}

/** The latest sign-ins and the risk checks that stopped one, with the way into the full log. */
export function RecentActivityPanel({
  organizationId,
  canReadAuth,
  canReadRisk,
  canOpenActivity,
}: {
  organizationId: string;
  canReadAuth: boolean;
  canReadRisk: boolean;
  canOpenActivity: boolean;
}) {
  const t = useT();
  const now = useNowSeconds(NOW_TICK_MS);
  const authEventsQuery = useQuery({
    ...organization.authEvents(organizationId),
    enabled: canReadAuth,
  });
  const riskQuery = useQuery({
    ...organization.riskDecisions(organizationId),
    enabled: canReadRisk,
  });

  if (!canReadAuth && !canReadRisk) {
    return null;
  }

  const loading = authEventsQuery.isLoading || riskQuery.isLoading;
  const failed = authEventsQuery.isError || riskQuery.isError;
  const lines = recentActivity(
    t,
    authEventsQuery.data ?? [],
    riskQuery.data ?? [],
    RECENT_ACTIVITY_LIMIT,
  );

  return (
    <SecuritySection
      title={t("Recent activity")}
      icon={<ActivityIcon />}
      action={
        canOpenActivity ? (
          <SectionPanelLink to={securityTabHref("activity")}>{t("View all")}</SectionPanelLink>
        ) : null
      }
    >
      {loading ? (
        <AsideRowsSkeleton rows={RECENT_ACTIVITY_LIMIT} label={t("Loading recent activity")} />
      ) : failed ? (
        <SectionQuiet>{t("Recent activity could not be loaded.")}</SectionQuiet>
      ) : lines.length === 0 ? (
        <SectionQuiet>
          {t("Sign-in and risk events appear here once people authenticate.")}
        </SectionQuiet>
      ) : (
        <ul className="divide-border-subtle divide-y">
          {lines.map((line) => (
            <AsideRow
              key={line.id}
              lead={
                <span aria-hidden className={cn("size-1.5 rounded-full", TONE_DOT[line.tone])} />
              }
              title={line.label}
              detail={line.detail}
              meta={formatCompactAge(now - line.occurredAt)}
              metaTitle={formatUnixDateTimeOrDash(line.occurredAt)}
            />
          ))}
        </ul>
      )}
    </SecuritySection>
  );
}
