import type { CarrierIntelEventStatus } from "@trenova/graphql/generated/graphql";
import { Badge, type BadgeVariant } from "@trenova/shared/components/ui/badge";
import { cn } from "@trenova/shared/lib/utils";
import { useCarrierIntelLabels } from "./use-carrier-intel-labels";

const STATUS_VARIANTS: Record<CarrierIntelEventStatus, BadgeVariant> = {
  Open: "warning",
  Acknowledged: "info",
  Resolved: "active",
  Dismissed: "secondary",
};

export type EventStatusBadgeProps = {
  status: CarrierIntelEventStatus;
  className?: string;
};

export function EventStatusBadge({ status, className }: EventStatusBadgeProps) {
  const labels = useCarrierIntelLabels();

  return (
    <Badge variant={STATUS_VARIANTS[status]} className={cn("max-h-5", className)}>
      {labels.eventStatus[status]}
    </Badge>
  );
}
