import { usePermission } from "@/hooks/use-permission";
import { fetchMyTeam, MY_TEAM_KEY, type TeamMemberRow } from "@/lib/graphql/org-structure";
import { useQuery } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { safetyRatingLabel, safetyRatingTone } from "@trenova/shared/lib/csa";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useMemo, useState } from "react";
import { Link } from "react-router";

function complianceTone(value: string): "active" | "inactive" | "warning" | "secondary" {
  switch (value) {
    case "Compliant":
      return "active";
    case "NonCompliant":
      return "inactive";
    case "Pending":
      return "warning";
    default:
      return "secondary";
  }
}

function trainingTone(value: string): "active" | "inactive" | "warning" | "secondary" {
  switch (value) {
    case "Current":
      return "active";
    case "Overdue":
    case "Expired":
      return "inactive";
    case "DueSoon":
      return "warning";
    default:
      return "secondary";
  }
}

export default function MyTeamConsole() {
  const { allowed: canRead } = usePermission(Resource.Worker, Operation.Read);
  const [includeInactive, setIncludeInactive] = useState(false);

  const teamQuery = useQuery({
    queryKey: [MY_TEAM_KEY, includeInactive],
    queryFn: ({ signal }) => fetchMyTeam(includeInactive, { signal }),
    enabled: canRead,
  });

  const members = useMemo(() => teamQuery.data ?? [], [teamQuery.data]);
  const direct = useMemo(() => members.filter((row) => row.direct), [members]);
  const throughTerminal = useMemo(() => members.filter((row) => !row.direct), [members]);

  if (!canRead) return null;

  if (teamQuery.isLoading) {
    return (
      <div className="flex flex-col gap-3">
        <Skeleton className="h-20 w-full" />
        <Skeleton className="h-40 w-full" />
      </div>
    );
  }

  const needsAttention = members.filter(
    (row) =>
      row.complianceStatus === "NonCompliant" ||
      row.trainingHealth === "Overdue" ||
      row.trainingHealth === "Expired" ||
      row.safetyRating === "AtRisk",
  );

  return (
    <div className="flex flex-col gap-4">
      <section className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <Figure label="On your team" value={String(members.length)} />
        <Figure
          label="Direct reports"
          value={String(direct.length)}
          detail="They name you themselves"
        />
        <Figure
          label="Through a terminal"
          value={String(throughTerminal.length)}
          detail="You run their fleet code"
        />
        <Figure
          label="Needing attention"
          value={String(needsAttention.length)}
          tone={needsAttention.length > 0 ? "critical" : undefined}
        />
      </section>

      <div className="flex justify-end">
        <Button
          size="xs"
          variant={includeInactive ? "default" : "outline"}
          onClick={() => setIncludeInactive((current) => !current)}
        >
          {includeInactive ? "Hiding nobody" : "Show people who have left"}
        </Button>
      </div>

      {members.length === 0 ? (
        <p className="text-muted-foreground rounded-md border border-dashed p-4 text-sm">
          Nobody reports to you yet. A worker joins your team when their record names you as their
          manager, or when you manage the terminal they are in.
        </p>
      ) : (
        <TeamList title="Direct reports" rows={direct} empty="Nobody reports to you directly." />
      )}

      {throughTerminal.length > 0 ? (
        <TeamList title="Through a terminal you run" rows={throughTerminal} empty="Nobody." />
      ) : null}
    </div>
  );
}

function TeamList({ title, rows, empty }: { title: string; rows: TeamMemberRow[]; empty: string }) {
  return (
    <section className="rounded-lg border p-4">
      <h3 className="text-sm font-medium">{title}</h3>
      {rows.length === 0 ? (
        <p className="text-muted-foreground mt-3 rounded-md border border-dashed p-3 text-xs">
          {empty}
        </p>
      ) : (
        <ul className="mt-3 flex flex-col gap-1.5">
          {rows.map((row) => (
            <li
              key={row.workerId}
              className="flex flex-wrap items-center justify-between gap-2 border-t pt-1.5 text-xs first:border-t-0 first:pt-0"
            >
              <Link
                to={`/hr/workers?entityId=${row.workerId}&modType=edit`}
                className="flex min-w-0 flex-wrap items-center gap-2 hover:underline"
              >
                {row.fleetColor ? (
                  <span
                    className="size-2 shrink-0 rounded-full"
                    style={{ backgroundColor: row.fleetColor }}
                  />
                ) : null}
                <span className="font-medium">{row.name}</span>
                {row.positionTitle ? (
                  <span className="text-muted-foreground truncate">{row.positionTitle}</span>
                ) : null}
                {row.fleetCode ? (
                  <span className="text-muted-foreground">{row.fleetCode}</span>
                ) : null}
              </Link>
              <span className="flex shrink-0 flex-wrap items-center gap-1.5">
                {row.status !== "Active" ? <Badge variant="inactive">Left</Badge> : null}
                <Badge variant={complianceTone(row.complianceStatus)}>{row.complianceStatus}</Badge>
                <Badge variant={trainingTone(row.trainingHealth)}>{row.trainingHealth}</Badge>
                <Badge variant={safetyRatingTone(row.safetyRating)}>
                  {safetyRatingLabel(row.safetyRating)}
                </Badge>
                <span className="text-muted-foreground">since {formatUnixDate(row.hireDate)}</span>
              </span>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

function Figure({
  label,
  value,
  detail,
  tone,
}: {
  label: string;
  value: string;
  detail?: string;
  tone?: "critical";
}) {
  return (
    <div className="rounded-lg border px-3 py-2">
      <p className="text-muted-foreground text-[11px]">{label}</p>
      <p
        className={
          tone === "critical"
            ? "text-lg font-semibold tabular-nums text-red-600 dark:text-red-400"
            : "text-lg font-semibold tabular-nums"
        }
      >
        {value}
      </p>
      {detail ? <p className="text-muted-foreground text-[11px]">{detail}</p> : null}
    </div>
  );
}
