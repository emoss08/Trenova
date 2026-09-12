import { useT } from "@trenova/shared/i18n/use-t";
import { usePermission } from "@/hooks/use-permission";
import {
  classifyMembers,
  coverSources,
  groupByTerminal,
  recentStarters,
  summarizeTeam,
  upcomingAnniversaries,
} from "@/lib/my-team";
import { useQuery } from "@tanstack/react-query";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useMemo, useState } from "react";
import { ApprovalCoverPanel, ByTerminalPanel, ComingUpPanel } from "./team-aside";
import { TeamAttention } from "./team-attention";
import { MyTeamEmpty } from "./my-team-empty";
import { MyTeamSkeleton } from "./my-team-skeleton";
import { coverQuery, myTeamQuery } from "./team-queries";
import { TeamRoster } from "./team-roster";
import { TeamSummaryStrip } from "./team-summary";

const EMPTY_DELEGATIONS: never[] = [];

export default function MyTeamConsole() {
  const t = useT();

  const { allowed: canRead } = usePermission(Resource.Worker, Operation.Read);
  const { allowed: canReadCover } = usePermission(Resource.ApprovalDelegation, Operation.Read);
  const userId = useAuthStore((state) => state.user?.id);
  const [includeInactive, setIncludeInactive] = useState(false);
  // One clock for the whole page: tenure, anniversaries and delegation windows
  // must agree with each other, and a re-render must not move "today".
  const [now] = useState(() => Math.floor(Date.now() / 1000));

  const teamQuery = useQuery({ ...myTeamQuery(includeInactive), enabled: canRead });
  const coverAllowed = canRead && canReadCover && Boolean(userId);
  const coverResult = useQuery({ ...coverQuery(userId ?? ""), enabled: coverAllowed });

  const delegations = coverResult.data ?? EMPTY_DELEGATIONS;
  const covers = useMemo(() => coverSources(delegations, now), [delegations, now]);
  const members = teamQuery.data;
  const rows = useMemo(
    () => classifyMembers(members ?? [], userId, covers),
    [members, userId, covers],
  );
  const summary = useMemo(() => summarizeTeam(rows, now), [rows, now]);
  const terminals = useMemo(() => groupByTerminal(rows), [rows]);
  const anniversaries = useMemo(() => upcomingAnniversaries(rows, now), [rows, now]);
  const starters = useMemo(() => recentStarters(rows, now), [rows, now]);
  const coverList = useMemo(() => Array.from(covers.values()), [covers]);

  if (!canRead) return null;

  if (teamQuery.isLoading) {
    return <MyTeamSkeleton showCover={coverAllowed} />;
  }

  if (teamQuery.isError) {
    return (
      <div className="text-destructive rounded-lg border border-dashed p-4 text-sm">
        {t("Your team could not be loaded. {0}", teamQuery.error.message)}
      </div>
    );
  }

  if (rows.length === 0 && !includeInactive) {
    return (
      <div className="flex flex-col gap-4">
        <MyTeamEmpty
          title={t("Nobody reports to you yet")}
          description={
            "A worker joins your team when their record names you as their manager, " +
            "when you manage the terminal they are in, or while a manager has handed " +
            "you their approvals."
          }
        />
        {coverAllowed ? (
          <div className="mx-auto w-full max-w-md">
            <ApprovalCoverPanel covers={coverList} isLoading={coverResult.isLoading} />
          </div>
        ) : null}
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <TeamSummaryStrip summary={summary} now={now} />
      <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_20rem]">
        <div className="flex min-w-0 flex-col gap-4">
          <TeamAttention rows={rows} />
          <TeamRoster
            rows={rows}
            now={now}
            includeInactive={includeInactive}
            onIncludeInactiveChange={setIncludeInactive}
          />
        </div>
        <aside className="flex min-w-0 flex-col gap-4">
          <ByTerminalPanel groups={terminals} total={rows.length} />
          <ComingUpPanel anniversaries={anniversaries} starters={starters} />
          {coverAllowed ? (
            <ApprovalCoverPanel covers={coverList} isLoading={coverResult.isLoading} />
          ) : null}
        </aside>
      </div>
    </div>
  );
}
