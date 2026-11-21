import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { SecurityFindingSeverity } from "@trenova/shared/types/iam";
import type { FindingGroups } from "./finding-copy";

const SIZE = 76; // size-19
const STROKE = 6;
const RADIUS = (SIZE - STROKE) / 2;
const CIRCUMFERENCE = 2 * Math.PI * RADIUS;
const SEGMENT_GAP = 3;

const SEVERITY_STROKE: Record<SecurityFindingSeverity, string> = {
  critical: "var(--danger)",
  warning: "var(--warning)",
  notice: "var(--info)",
  pass: "var(--success)",
};

const SEVERITY_DOT: Record<SecurityFindingSeverity, string> = {
  critical: "bg-danger",
  warning: "bg-warning",
  notice: "bg-info",
  pass: "bg-success",
};

const LEGEND_ORDER: readonly SecurityFindingSeverity[] = ["critical", "warning", "notice", "pass"];

function legendLabel(t: TranslateFn, severity: SecurityFindingSeverity, count: number): string {
  switch (severity) {
    case "critical":
      return t("{0} critical", count);
    case "warning":
      return t("{0} need attention", count);
    case "notice":
      return t("{0} recommended", count);
    case "pass":
      return t("{0} passed", count);
  }
}

/**
 * Every check the briefing ran, one segment each around a ring, worst first and coloured by how
 * it came out, with how many passed in the middle and the count of each outcome beside it.
 */
export function PostureRing({ groups }: { groups: FindingGroups }) {
  const t = useT();
  const severities = [
    ...groups.open.map((finding) => finding.severity),
    ...groups.passed.map((finding) => finding.severity),
  ];
  const total = severities.length;
  const passed = groups.passed.length;
  const counts: Record<SecurityFindingSeverity, number> = {
    critical: groups.critical,
    warning: groups.warning,
    notice: groups.notice,
    pass: passed,
  };
  const slice = total > 0 ? CIRCUMFERENCE / total : CIRCUMFERENCE;
  const gap = total > 1 ? SEGMENT_GAP : 0;

  return (
    <div className="flex items-center gap-5">
      <div className="relative size-19 shrink-0">
        <svg
          role="img"
          aria-label={t("{0} of {1} checks pass", passed, total)}
          width={SIZE}
          height={SIZE}
          viewBox={`0 0 ${SIZE} ${SIZE}`}
          className="-rotate-90"
        >
          <circle
            cx={SIZE / 2}
            cy={SIZE / 2}
            r={RADIUS}
            fill="none"
            stroke="var(--border-subtle)"
            strokeWidth={STROKE}
          />
          {severities.map((severity, index) => (
            <circle
              key={index}
              cx={SIZE / 2}
              cy={SIZE / 2}
              r={RADIUS}
              fill="none"
              stroke={SEVERITY_STROKE[severity]}
              strokeWidth={STROKE}
              strokeDasharray={`${Math.max(0, slice - gap)} ${CIRCUMFERENCE}`}
              strokeDashoffset={-index * slice}
            />
          ))}
        </svg>
        <div
          aria-hidden
          className="absolute inset-0 flex flex-col items-center justify-center leading-none"
        >
          <span className="text-xl font-semibold tabular-nums">{passed}</span>
          <span className="text-foreground-subtle mt-0.5 text-2xs tabular-nums">
            {t("of {0}", total)}
          </span>
        </div>
      </div>
      <ul aria-label={t("Check outcomes")} className="flex flex-col gap-1">
        {LEGEND_ORDER.filter((severity) => counts[severity] > 0).map((severity) => (
          <li key={severity} className="text-muted-foreground flex items-center gap-2 text-xs">
            <span aria-hidden className={cn("size-1.5 rounded-full", SEVERITY_DOT[severity])} />
            {legendLabel(t, severity, counts[severity])}
          </li>
        ))}
      </ul>
    </div>
  );
}
