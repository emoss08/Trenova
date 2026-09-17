import { carrierIntelProviderLabel, groupFindings } from "@/lib/carrier-intelligence";
import type { CarrierIntelFinding } from "@/lib/graphql/carrier-intelligence";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useMemo, type ReactNode } from "react";
import { FindingList } from "./finding-list";
import { StatusDot } from "./status-dot";

export type DecisionSummaryProps = {
  findings: readonly CarrierIntelFinding[];
  ruleLabels: Readonly<Record<string, string>>;
  notFound: boolean;
  provider: string | null | undefined;
  grouped?: boolean;
  renderActions?: (finding: CarrierIntelFinding) => ReactNode;
  className?: string;
};

export function DecisionSummary({
  findings,
  ruleLabels,
  notFound,
  provider,
  grouped = false,
  renderActions,
  className,
}: DecisionSummaryProps) {
  const t = useT();
  const counts = useMemo(() => {
    const byKind = groupFindings(findings);
    return {
      blockers: byKind.blockers.filter((finding) => !finding.overridden).length,
      overridden: byKind.blockers.filter((finding) => finding.overridden).length,
      advisories: byKind.advisories.length,
    };
  }, [findings]);

  const headline = notFound
    ? t("{0} has no record for this carrier", carrierIntelProviderLabel(provider))
    : counts.blockers > 0
      ? t(
          "{0, plural, one {# issue blocks tendering} other {# issues block tendering}}",
          counts.blockers,
        )
      : t("No blocking issues");
  const tone = notFound ? "neutral" : counts.blockers > 0 ? "critical" : "success";
  const extras = notFound
    ? []
    : [
        counts.overridden > 0
          ? t("{0, plural, one {# overridden} other {# overridden}}", counts.overridden)
          : null,
        counts.advisories > 0
          ? t("{0, plural, one {# advisory} other {# advisories}}", counts.advisories)
          : null,
      ].filter((extra): extra is string => extra !== null);

  return (
    <section className={cn("flex flex-col gap-2", className)} aria-label={t("Decision")}>
      <p className="flex items-center gap-2 text-sm font-medium" data-testid="decision-headline">
        <StatusDot tone={tone} />
        <span>{headline}</span>
        {extras.length > 0 ? (
          <span className="text-muted-foreground font-normal">· {extras.join(" · ")}</span>
        ) : null}
      </p>
      {notFound ? (
        <p className="text-muted-foreground text-xs">
          {t(
            "Check the USDOT number on the record, or confirm the carrier is registered with the FMCSA.",
          )}
        </p>
      ) : null}
      <FindingList
        findings={findings}
        ruleLabels={ruleLabels}
        grouped={grouped}
        renderActions={renderActions}
        emptyMessage={notFound ? null : t("Every enabled vetting rule passed for this carrier.")}
      />
    </section>
  );
}
