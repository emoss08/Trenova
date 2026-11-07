import { useT } from "@trenova/shared/i18n/use-t";
import type { DispatchFinding } from "@/lib/graphql/dispatch-console";
import { cn } from "@trenova/shared/lib/utils";
import { AlertTriangleIcon, InfoCircleIcon, XOctagonIcon } from "@trenova/shared/components/icons";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { severityMeta } from "./dispatch-vocabulary";

const SEVERITY_RANK: Record<string, number> = { Block: 0, Warn: 1, Info: 2 };

const SEVERITY_ROW: Record<
  string,
  { Icon: typeof InfoCircleIcon; iconClass: string; textClass: string }
> = {
  Block: {
    Icon: XOctagonIcon,
    iconClass: "text-danger-foreground",
    textClass: "text-foreground",
  },
  Warn: {
    Icon: AlertTriangleIcon,
    iconClass: "text-warning-foreground",
    textClass: "text-foreground/80",
  },
  Info: {
    Icon: InfoCircleIcon,
    iconClass: "text-muted-foreground",
    textClass: "text-muted-foreground",
  },
};

/**
 * Findings are the console's explanation for why a driver can or cannot take a move.
 * Blocking reasons lead, because that is the one a dispatcher has to act on. Severity
 * reads from the icon for the eye and is spoken before the reason for a screen reader.
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
  const t = useT();

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
            <span className={cn("min-w-0 flex-1 text-xs leading-snug", row.textClass)}>
              <span className="sr-only">{t("{0}:", t(severityMeta(finding.severity).label))} </span>
              {finding.message}
            </span>
            {finding.regulation ? (
              <span className="border-border bg-muted/50 text-muted-foreground shrink-0 rounded-md border px-1 py-px font-mono text-3xs leading-4">
                {finding.regulation}
              </span>
            ) : null}
          </li>
        );
      })}
      {hidden > 0 ? (
        <li className="text-muted-foreground pl-[22px] text-xs">{t("+{0} more", hidden)}</li>
      ) : null}
    </ul>
  );
}

/** The identity of a finding across drivers: the same rule on the same field, said the same way. */
export function findingKey(finding: DispatchFinding): string {
  return `${finding.code}|${finding.field}|${finding.message}`;
}

/** Findings most severe first, the order every finding surface reads in. */
function bySeverity(findings: readonly DispatchFinding[]): DispatchFinding[] {
  return [...findings].sort(
    (a, b) => (SEVERITY_RANK[a.severity] ?? 3) - (SEVERITY_RANK[b.severity] ?? 3),
  );
}

/**
 * Findings in one line: the most severe, cut to fit, with how many more there are. Its
 * tooltip, on hover or focus, holds every finding in full, so a long explanation never
 * takes over the row it describes.
 */
export function FindingSummary({
  findings,
  className,
}: {
  findings: readonly DispatchFinding[];
  className?: string;
}) {
  const t = useT();

  if (findings.length === 0) return null;

  const sorted = bySeverity(findings);
  const [lead] = sorted;
  const row = SEVERITY_ROW[lead.severity] ?? SEVERITY_ROW.Info;
  const more = sorted.length - 1;

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <span
            tabIndex={0}
            className={cn(
              "ui-focus-ring flex min-w-0 cursor-help items-center gap-1.5 rounded-sm text-2xs",
              className,
            )}
          />
        }
      >
        <row.Icon className={cn("size-3 shrink-0", row.iconClass)} aria-hidden />
        <span className={cn("min-w-0 truncate", row.textClass)}>
          <span className="sr-only">{t("{0}:", t(severityMeta(lead.severity).label))} </span>
          {lead.message}
        </span>
        {more > 0 ? (
          <span className="text-muted-foreground shrink-0 tabular-nums">{t("+{0}", more)}</span>
        ) : null}
      </TooltipTrigger>
      <TooltipContent className="max-w-80">
        <ul className="flex flex-col gap-1.5">
          {sorted.map((finding) => (
            <li key={findingKey(finding)} className="leading-snug">
              {finding.regulation
                ? t(
                    "{0}: {1} ({2})",
                    t(severityMeta(finding.severity).label),
                    finding.message,
                    finding.regulation,
                  )
                : t("{0}: {1}", t(severityMeta(finding.severity).label), finding.message)}
            </li>
          ))}
        </ul>
      </TooltipContent>
    </Tooltip>
  );
}
