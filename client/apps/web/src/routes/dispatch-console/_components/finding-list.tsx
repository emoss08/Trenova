import type { DispatchFinding } from "@/lib/graphql/dispatch-console";
import { cn } from "@trenova/shared/lib/utils";
import { InfoIcon, OctagonXIcon, TriangleAlertIcon } from "lucide-react";

const SEVERITY_RANK: Record<string, number> = { Block: 0, Warn: 1, Info: 2 };

const SEVERITY_ROW: Record<
  string,
  { Icon: typeof InfoIcon; iconClass: string; textClass: string }
> = {
  Block: {
    Icon: OctagonXIcon,
    iconClass: "text-red-600 dark:text-red-400",
    textClass: "text-foreground",
  },
  Warn: {
    Icon: TriangleAlertIcon,
    iconClass: "text-amber-600 dark:text-amber-400",
    textClass: "text-foreground/80",
  },
  Info: {
    Icon: InfoIcon,
    iconClass: "text-muted-foreground",
    textClass: "text-muted-foreground",
  },
};

/**
 * Findings are the console's explanation for why a driver can or cannot take a move.
 * Blocking reasons lead, because that is the one a dispatcher has to act on. Severity
 * reads from the icon alone; the words are saved for the reason itself.
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
    <ul className={cn("flex flex-col gap-1.5", className)}>
      {shown.map((finding) => {
        const row = SEVERITY_ROW[finding.severity] ?? SEVERITY_ROW.Info;
        return (
          <li key={`${finding.code}-${finding.field}`} className="flex items-start gap-2">
            <row.Icon className={cn("mt-px size-3.5 shrink-0", row.iconClass)} aria-hidden />
            <span className={cn("min-w-0 flex-1 text-[11px] leading-snug", row.textClass)}>
              {finding.message}
            </span>
            {finding.regulation ? (
              <span className="border-border bg-muted/50 text-muted-foreground shrink-0 rounded border px-1 py-px font-mono text-[9px] leading-4">
                {finding.regulation}
              </span>
            ) : null}
          </li>
        );
      })}
      {hidden > 0 ? (
        <li className="text-muted-foreground pl-[22px] text-[11px]">+{hidden} more</li>
      ) : null}
    </ul>
  );
}
