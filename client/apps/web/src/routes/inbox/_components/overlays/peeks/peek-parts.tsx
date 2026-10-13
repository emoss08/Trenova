import type { TriageShipmentPeek } from "@/lib/graphql/inbox";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { BadgeVariant } from "@trenova/shared/types/badge";
import { useT } from "@trenova/shared/i18n/use-t";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";

/** Where a shipment is in its life, as a peek says it: delivered, running late, or on its way. */
export type PeekPhase = "delivered" | "late" | "moving";

export function peekPhase(
  shipment: Pick<TriageShipmentPeek, "progress" | "late" | "deliveredAt">,
): PeekPhase {
  if (shipment.deliveredAt !== null || shipment.progress >= 1) {
    return "delivered";
  }

  return shipment.late ? "late" : "moving";
}

const PHASE_TONE: Record<PeekPhase, BadgeVariant> = {
  delivered: "success",
  late: "warning",
  moving: "info",
};

export function phaseLabel(t: TranslateFn, phase: PeekPhase): string {
  switch (phase) {
    case "delivered":
      return t("Delivered");
    case "late":
      return t("Running late");
    case "moving":
      return t("In transit");
  }
}

export function PhaseBadge({ phase }: { phase: PeekPhase }) {
  const t = useT();

  return (
    <Badge variant={PHASE_TONE[phase]} className="gap-1.5">
      <span aria-hidden className="size-1.5 rounded-full bg-current" />
      {phaseLabel(t, phase)}
    </Badge>
  );
}
