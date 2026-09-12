import { attentionReasons, isWatched, type ClassifiedMember } from "@/lib/my-team";
import { Badge } from "@trenova/shared/components/ui/badge";
import { AlertTriangleIcon, ChevronRightIcon, CircleCheckIcon } from "lucide-react";
import { useMemo } from "react";
import { Link } from "react-router";
import { MemberIdentity, memberHref } from "./member-identity";

type TeamAttentionProps = {
  rows: readonly ClassifiedMember[];
};

/**
 * The people a manager has to do something about, worst first. It sits above
 * the roster because the roster answers "who is on my team" and this answers
 * "who do I chase today" — the question the page is opened for.
 */
export function TeamAttention({ rows }: TeamAttentionProps) {
  const flagged = useMemo(
    () =>
      rows
        .map((row) => ({ row, reasons: attentionReasons(row.member) }))
        .filter(({ reasons }) => reasons.some((reason) => reason.severity === "critical"))
        .sort(
          (a, b) =>
            criticalCount(b.reasons) - criticalCount(a.reasons) ||
            a.row.member.name.localeCompare(b.row.member.name),
        ),
    [rows],
  );
  const watched = useMemo(
    () => rows.filter((row) => isWatched(row.member)).map((row) => row.member),
    [rows],
  );

  return (
    <section
      aria-labelledby="team-attention-heading"
      className="bg-card overflow-hidden rounded-lg border"
    >
      <header className="flex items-center justify-between gap-2 border-b px-3 py-2">
        <div className="flex items-center gap-2">
          {flagged.length > 0 ? (
            <AlertTriangleIcon className="text-destructive size-3.5" aria-hidden />
          ) : (
            <CircleCheckIcon className="text-muted-foreground size-3.5" aria-hidden />
          )}
          <h3 id="team-attention-heading" className="text-sm font-medium">
            Needs your attention
          </h3>
          {flagged.length > 0 ? (
            <Badge variant="inactive" className="text-2xs h-4 px-1 tabular-nums">
              {flagged.length}
            </Badge>
          ) : null}
        </div>
        {watched.length > 0 ? (
          <span className="text-muted-foreground text-xs">{watched.length} to keep an eye on</span>
        ) : null}
      </header>

      {flagged.length === 0 ? (
        <div className="px-3 py-3">
          <p className="text-muted-foreground text-sm">
            Everyone is in good standing.
            {watched.length > 0 ? (
              <>
                {" "}
                Worth a look soon:{" "}
                <span className="text-foreground">
                  {watched.map((member) => member.name).join(", ")}
                </span>
                .
              </>
            ) : null}
          </p>
        </div>
      ) : (
        <ul className="divide-y">
          {flagged.map(({ row, reasons }) => (
            <li key={row.member.workerId}>
              <Link
                to={memberHref(row.member.workerId)}
                className="group/row hover:bg-accent grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 px-3 py-2 transition-colors"
              >
                <div className="flex min-w-0 flex-row flex-wrap items-center gap-x-4 gap-y-1">
                  <MemberIdentity member={row.member} size="sm" className="w-auto min-w-0 flex-1" />
                  {/* Every reason, not just the worst: a manager chasing one
                      person wants to deal with all of it in the one trip. */}
                  <span className="flex min-w-0 flex-wrap items-center gap-1">
                    {reasons.map((reason) => (
                      <Badge
                        key={reason.key}
                        variant={reason.severity === "critical" ? "inactive" : "warning"}
                        className="text-2xs h-4 px-1"
                      >
                        {reason.label}
                      </Badge>
                    ))}
                  </span>
                </div>
                <ChevronRightIcon
                  aria-hidden
                  className="text-muted-foreground size-4 transition-transform group-hover/row:translate-x-0.5"
                />
              </Link>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

function criticalCount(reasons: ReturnType<typeof attentionReasons>): number {
  return reasons.filter((reason) => reason.severity === "critical").length;
}
