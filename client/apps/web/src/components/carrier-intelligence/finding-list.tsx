import { useT } from "@trenova/shared/i18n/use-t";
import type { CarrierIntelFinding } from "@/lib/graphql/carrier-intelligence";
import { groupFindings } from "@/lib/carrier-intelligence";
import { Badge } from "@trenova/shared/components/ui/badge";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import {
  BanIcon,
  BellIcon,
  CircleCheckIcon,
  HelpCircleIcon,
  HourglassIcon,
  ShieldOffIcon,
  TriangleAlertIcon,
  type LucideIcon,
} from "lucide-react";
import { useId, useMemo, type ReactNode } from "react";
import { SeverityBadge } from "./severity-badge";
import { useCarrierIntelLabels } from "./use-carrier-intel-labels";

export type FindingListProps = {
  findings: readonly CarrierIntelFinding[];
  ruleLabels?: Readonly<Record<string, string>>;
  renderActions?: (finding: CarrierIntelFinding) => ReactNode;
  emptyMessage?: string;
  className?: string;
};

type FindingGroupProps = {
  id: string;
  title: string;
  description: string;
  icon: LucideIcon;
  tone: "blocker" | "advisory" | "notice";
  findings: CarrierIntelFinding[];
  ruleLabels?: Readonly<Record<string, string>>;
  renderActions?: (finding: CarrierIntelFinding) => ReactNode;
};

const TONE_CLASSES: Record<FindingGroupProps["tone"], string> = {
  blocker: "text-red-700 dark:text-red-400",
  advisory: "text-yellow-700 dark:text-yellow-400",
  notice: "text-muted-foreground",
};

function FindingFlags({ finding }: { finding: CarrierIntelFinding }) {
  const t = useT();

  return (
    <>
      {finding.unverifiable ? (
        <Badge
          variant="secondary"
          className="max-h-5"
          title={t(
            "The provider did not supply the data this rule needs, so it could not be checked.",
          )}
        >
          <HelpCircleIcon aria-hidden />
          {t("Unverifiable")}
        </Badge>
      ) : null}
      {finding.unconfirmed ? (
        <Badge
          variant="purple"
          className="max-h-5"
          title={t("A blocking change is waiting for a confirming refresh before it takes effect.")}
        >
          <HourglassIcon aria-hidden />
          {t("Unconfirmed")}
        </Badge>
      ) : null}
      {finding.overridden ? (
        <Badge
          variant="teal"
          className="max-h-5"
          title={t("An approved override lets this carrier through while it lasts.")}
        >
          <ShieldOffIcon aria-hidden />
          {finding.overrideExpiresAt
            ? t("Overridden until {0}", formatUnixDateTimeMedium(finding.overrideExpiresAt))
            : t("Overridden")}
        </Badge>
      ) : null}
    </>
  );
}

function FindingGroup({
  id,
  title,
  description,
  icon: Icon,
  tone,
  findings,
  ruleLabels,
  renderActions,
}: FindingGroupProps) {
  const labels = useCarrierIntelLabels();

  if (findings.length === 0) {
    return null;
  }

  const headingId = `${id}-heading`;

  return (
    <section aria-labelledby={headingId} data-finding-group={tone} className="flex flex-col">
      <header className="flex items-center gap-2 px-3 py-2">
        <Icon className={cn("size-3.5", TONE_CLASSES[tone])} aria-hidden />
        <h4 id={headingId} className="text-sm font-medium">
          {title}
        </h4>
        <Badge variant="outline" className="tabular-nums">
          {findings.length}
        </Badge>
        <span className="text-muted-foreground hidden text-xs sm:inline">{description}</span>
      </header>
      <ul className="divide-y border-t">
        {findings.map((finding) => {
          const ruleLabel = ruleLabels?.[finding.code];
          const actions = renderActions?.(finding);
          return (
            <li
              key={`${finding.action}-${finding.code}`}
              className={cn(
                "flex flex-col gap-1.5 px-3 py-2.5 sm:flex-row sm:items-start sm:justify-between",
                finding.overridden && "bg-muted/40",
              )}
            >
              <div className="flex min-w-0 flex-col gap-1">
                <div className="flex flex-wrap items-center gap-1.5">
                  <SeverityBadge severity={finding.severity} />
                  <span className="text-sm font-medium">{ruleLabel ?? finding.code}</span>
                  <span className="text-muted-foreground text-xs">
                    {labels.section[finding.category]}
                  </span>
                  <FindingFlags finding={finding} />
                </div>
                <p
                  className={cn(
                    "text-sm",
                    finding.overridden ? "text-muted-foreground" : "text-foreground",
                  )}
                >
                  {finding.message}
                </p>
                {ruleLabel ? (
                  <span className="text-muted-foreground font-mono text-2xs">{finding.code}</span>
                ) : null}
              </div>
              {actions ? <div className="flex shrink-0 items-center gap-1.5">{actions}</div> : null}
            </li>
          );
        })}
      </ul>
    </section>
  );
}

export function FindingList({
  findings,
  ruleLabels,
  renderActions,
  emptyMessage,
  className,
}: FindingListProps) {
  const t = useT();
  const baseId = useId();
  const grouped = useMemo(() => groupFindings(findings), [findings]);
  const total = grouped.blockers.length + grouped.advisories.length + grouped.notices.length;

  if (total === 0) {
    return (
      <div
        className={cn(
          "text-muted-foreground flex items-center gap-2 rounded-lg border border-dashed px-3 py-4 text-sm",
          className,
        )}
      >
        <CircleCheckIcon className="size-4 text-green-600" aria-hidden />
        {emptyMessage ?? t("No findings. Every enabled rule passed on the latest vetting.")}
      </div>
    );
  }

  return (
    <div
      className={cn("bg-card flex flex-col divide-y overflow-hidden rounded-lg border", className)}
    >
      <FindingGroup
        id={`${baseId}-blockers`}
        title={t("Blockers")}
        description={t("Stop tendering until resolved or overridden.")}
        icon={BanIcon}
        tone="blocker"
        findings={grouped.blockers}
        ruleLabels={ruleLabels}
        renderActions={renderActions}
      />
      <FindingGroup
        id={`${baseId}-advisories`}
        title={t("Advisories")}
        description={t("Worth a look before assigning freight.")}
        icon={TriangleAlertIcon}
        tone="advisory"
        findings={grouped.advisories}
        ruleLabels={ruleLabels}
        renderActions={renderActions}
      />
      <FindingGroup
        id={`${baseId}-notices`}
        title={t("Notices")}
        description={t("Recorded for awareness only.")}
        icon={BellIcon}
        tone="notice"
        findings={grouped.notices}
        ruleLabels={ruleLabels}
        renderActions={renderActions}
      />
    </div>
  );
}
