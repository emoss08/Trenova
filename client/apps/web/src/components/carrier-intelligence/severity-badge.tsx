import type { CarrierIntelSeverity } from "@trenova/graphql/generated/graphql";
import { Badge, type BadgeVariant } from "@trenova/shared/components/ui/badge";
import { cn } from "@trenova/shared/lib/utils";
import { useCarrierIntelLabels } from "./use-carrier-intel-labels";

const SEVERITY_VARIANTS: Record<CarrierIntelSeverity, BadgeVariant> = {
  Critical: "inactive",
  High: "orange",
  Medium: "warning",
  Low: "info",
  Info: "secondary",
};

export type SeverityBadgeProps = {
  severity: CarrierIntelSeverity;
  className?: string;
};

export function SeverityBadge({ severity, className }: SeverityBadgeProps) {
  const labels = useCarrierIntelLabels();

  return (
    <Badge variant={SEVERITY_VARIANTS[severity]} className={cn("max-h-5", className)}>
      {labels.severity[severity]}
    </Badge>
  );
}
