import { useT } from "@trenova/shared/i18n/use-t";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import type { Anniversary, CoverSource, RecentStarter, TerminalGroup } from "@/lib/my-team";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDate, formatUnixMonthDay } from "@trenova/shared/lib/date";
import { approvalScopeLabel, headcountShare } from "@trenova/shared/lib/org-structure";
import { cn } from "@trenova/shared/lib/utils";
import {
  ArrowUpRightIcon,
  AwardIcon,
  Building2Icon,
  CalendarDaysIcon,
  HandshakeIcon,
  SparklesIcon,
} from "lucide-react";
import { Link } from "react-router";
import { MemberAvatar, memberHref } from "./member-identity";

type ByTerminalProps = {
  groups: readonly TerminalGroup[];
  total: number;
};

export function ByTerminalPanel({ groups, total }: ByTerminalProps) {
  const t = useT();

  return (
    <SectionPanel
      title={t("By terminal")}
      icon={<Building2Icon />}
      help={t("The team by the terminal each person sits in, biggest first. Somebody with no terminal is listed as such rather than dropped.")}
    >
      {groups.length === 0 ? (
        <SectionPanelQuiet>{t("Nobody to count yet.")}</SectionPanelQuiet>
      ) : (
        <ul className="divide-y">
          {groups.map((group) => (
            <li key={group.key} className="flex flex-col gap-1.5 px-3 py-2">
              <div className="flex items-center justify-between gap-2 text-xs">
                <span className="flex min-w-0 items-center gap-2">
                  <span
                    aria-hidden
                    className={cn(
                      "size-2 shrink-0 rounded-full",
                      !group.color && "bg-muted-foreground/40",
                    )}
                    style={group.color ? { backgroundColor: group.color } : undefined}
                  />
                  <span className="truncate font-medium">{group.code}</span>
                  {group.attention > 0 ? (
                    <span className="text-muted-foreground">{t("· {0} flagged", group.attention)}</span>
                  ) : null}
                </span>
                <span className="text-muted-foreground tabular-nums">{group.count}</span>
              </div>
              <div
                role="img"
                aria-label={`${group.code}: ${group.count} of ${total}`}
                className="bg-muted h-1 w-full overflow-hidden rounded-full"
              >
                <div
                  className="bg-brand/50 h-full rounded-full transition-[width] duration-700 ease-out motion-reduce:transition-none"
                  style={{ width: `${headcountShare(group.count, total)}%` }}
                />
              </div>
            </li>
          ))}
        </ul>
      )}
    </SectionPanel>
  );
}

type ComingUpProps = {
  anniversaries: readonly Anniversary[];
  starters: readonly RecentStarter[];
};

export function ComingUpPanel({ anniversaries, starters }: ComingUpProps) {
  const t = useT();

  const empty = anniversaries.length === 0 && starters.length === 0;
  return (
    <SectionPanel
      title={t("Coming up")}
      icon={<CalendarDaysIcon />}
      help={t("Work anniversaries and recent starters inside the window. Only whole years count, and only for people still here.")}
    >
      {empty ? (
        <SectionPanelQuiet>
          {t("No anniversaries in the next 30 days, and nobody started this quarter.")}
        </SectionPanelQuiet>
      ) : (
        <ul className="divide-y">
          {anniversaries.map((item) => (
            <li key={`anniversary-${item.member.workerId}`}>
              <Link
                to={memberHref(item.member.workerId)}
                className="hover:bg-accent flex items-center gap-2.5 px-3 py-2 transition-colors"
              >
                <MemberAvatar member={item.member} size="sm" />
                <span className="flex min-w-0 flex-1 flex-col leading-tight">
                  <span className="truncate text-xs font-medium">{item.member.name}</span>
                  <span className="text-muted-foreground text-2xs flex items-center gap-1">
                    <AwardIcon className="size-3" aria-hidden />
                    {t("{0}{1} on {2}", item.years, item.years === 1 ? "year" : "years", formatUnixMonthDay(item.onDate))}
                  </span>
                </span>
                <span className="text-muted-foreground shrink-0 text-xs tabular-nums">
                  {describeInDays(item.inDays)}
                </span>
              </Link>
            </li>
          ))}
          {starters.map((item) => (
            <li key={`starter-${item.member.workerId}`}>
              <Link
                to={memberHref(item.member.workerId)}
                className="hover:bg-accent flex items-center gap-2.5 px-3 py-2 transition-colors"
              >
                <MemberAvatar member={item.member} size="sm" />
                <span className="flex min-w-0 flex-1 flex-col leading-tight">
                  <span className="truncate text-xs font-medium">{item.member.name}</span>
                  <span className="text-muted-foreground text-2xs flex items-center gap-1">
                    <SparklesIcon className="size-3" aria-hidden />
                    {t("Started {0}", formatUnixDate(item.member.hireDate))}
                  </span>
                </span>
                <span className="text-muted-foreground shrink-0 text-xs tabular-nums">
                  {describeDaysAgo(item.daysAgo)}
                </span>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </SectionPanel>
  );
}

function describeInDays(days: number): string {
  if (days === 0) return "Today";
  if (days === 1) return "Tomorrow";
  return `In ${days}d`;
}

function describeDaysAgo(days: number): string {
  if (days === 0) return "Today";
  if (days === 1) return "Yesterday";
  return `${days}d ago`;
}

type ApprovalCoverProps = {
  covers: readonly CoverSource[];
  isLoading: boolean;
};

export function ApprovalCoverPanel({ covers, isLoading }: ApprovalCoverProps) {
  const t = useT();

  return (
    <SectionPanel
      title={t("Approval cover")}
      icon={<HandshakeIcon />}
      help={t("Delegations that put another manager's approvals in your hands today. A delegation never widens what you could approve on your own.")}
      action={
        <Link
          to="/hr/org-structure"
          className="text-muted-foreground hover:text-foreground flex items-center gap-0.5 text-xs transition-colors"
        >
          {t("Manage")}
          <ArrowUpRightIcon className="size-3" aria-hidden />
        </Link>
      }
    >
      {isLoading ? (
        <div className="flex flex-col gap-2 p-3">
          <Skeleton className="h-4 w-3/4" />
          <Skeleton className="h-4 w-1/2" />
        </div>
      ) : covers.length === 0 ? (
        <SectionPanelQuiet>{t("Nobody has handed you their approvals right now.")}</SectionPanelQuiet>
      ) : (
        <ul className="divide-y">
          {covers.map((cover) => (
            <li key={cover.id} className="flex flex-col gap-1 px-3 py-2">
              <div className="flex items-center justify-between gap-2">
                <span className="truncate text-xs font-medium">{cover.name}</span>
                <Badge variant="secondary">{approvalScopeLabel(cover.scope)}</Badge>
              </div>
              <span className="text-muted-foreground text-2xs">
                {cover.endsAt ? `Until ${formatUnixDate(cover.endsAt)}` : "Until called back"}
                {cover.reason ? ` · ${cover.reason}` : ""}
              </span>
            </li>
          ))}
        </ul>
      )}
    </SectionPanel>
  );
}
