import { useT } from "@trenova/shared/i18n/use-t";
import type { CarrierIntelRiskLevel } from "@trenova/graphql/generated/graphql";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { BadgeAttrProps } from "@trenova/shared/components/status-badge";
import { cn } from "@trenova/shared/lib/utils";
import { CARRIER_INTEL_RISK_LEVELS } from "@/lib/carrier-intelligence";

export type RiskLevelBadgeProps = {
  level: CarrierIntelRiskLevel | string | null | undefined;
  className?: string;
};

function isRiskLevel(value: string): value is CarrierIntelRiskLevel {
  return (CARRIER_INTEL_RISK_LEVELS as readonly string[]).includes(value);
}

export function RiskLevelBadge({ level, className }: RiskLevelBadgeProps) {
  const t = useT();

  const attributes: Record<CarrierIntelRiskLevel, BadgeAttrProps> = {
    Low: {
      variant: "active",
      text: t("Low risk"),
      description: t("Nothing in the latest vetting points to elevated risk."),
    },
    Moderate: {
      variant: "info",
      text: t("Moderate risk"),
      description: t("Minor findings worth knowing about before tendering."),
    },
    Elevated: {
      variant: "warning",
      text: t("Elevated risk"),
      description: t("Findings that call for a closer look before assigning freight."),
    },
    High: {
      variant: "orange",
      text: t("High risk"),
      description: t("Serious findings. Review the carrier before tendering."),
    },
    VeryHigh: {
      variant: "inactive",
      text: t("Very high risk"),
      description: t("Blocking-level findings. Do not tender without a review."),
    },
    Unknown: {
      variant: "secondary",
      text: t("Risk unknown"),
      description: t("The provider did not return enough data to score this carrier."),
    },
  };

  if (!level) {
    return (
      <Badge
        variant="outline"
        className={cn("max-h-5", className)}
        title={t("This carrier has not been vetted yet.")}
      >
        {t("Not vetted")}
      </Badge>
    );
  }

  const attrs = isRiskLevel(level) ? attributes[level] : attributes.Unknown;

  return (
    <Badge variant={attrs.variant} className={cn("max-h-5", className)} title={attrs.description}>
      {attrs.text}
    </Badge>
  );
}
