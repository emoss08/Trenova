import type { DispatchFinding } from "@/lib/graphql/dispatch-console";
import { Badge } from "@trenova/shared/components/ui/badge";
import { cn } from "@trenova/shared/lib/utils";
import { severityMeta } from "./dispatch-vocabulary";

const SEVERITY_RANK: Record<string, number> = { Block: 0, Warn: 1, Info: 2 };

/**
 * Findings are the console's explanation for why a driver can or cannot take a move.
 * Blocking reasons lead, because that is the one a dispatcher has to act on.
 */
export function FindingList({
  findings,
  className,
  limit,
}: {
  findings: readonly DispatchFinding[];
  className?: string;
  limit?: number;
}) {
  if (findings.length === 0) return null;

  const sorted = [...findings].sort(
    (a, b) => (SEVERITY_RANK[a.severity] ?? 3) - (SEVERITY_RANK[b.severity] ?? 3),
  );
  const shown = limit ? sorted.slice(0, limit) : sorted;
  const hidden = sorted.length - shown.length;

  return (
    <ul className={cn("flex flex-col gap-1", className)}>
      {shown.map((finding) => {
        const meta = severityMeta(finding.severity);
        return (
          <li key={`${finding.code}-${finding.field}`} className="flex items-start gap-1.5">
            <Badge variant={meta.variant} className="mt-0.5 h-4 shrink-0 rounded px-1 text-[9px]">
              {meta.label}
            </Badge>
            <span className="text-[11px] leading-tight text-muted-foreground">
              {finding.message}
              {finding.regulation ? (
                <span className="ml-1 font-mono text-[10px] opacity-70">{finding.regulation}</span>
              ) : null}
            </span>
          </li>
        );
      })}
      {hidden > 0 ? (
        <li className="text-[11px] text-muted-foreground">+{hidden} more</li>
      ) : null}
    </ul>
  );
}
