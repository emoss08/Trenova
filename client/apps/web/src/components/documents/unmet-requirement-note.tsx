import { useT } from "@trenova/shared/i18n/use-t";
import { unmetRequirementStanding } from "@/lib/document-review";
import { cn } from "@trenova/shared/lib/utils";
import type { ShipmentBillingRequirement } from "@trenova/shared/types/shipment";

type UnmetRequirementNoteProps = {
  requirement: Pick<ShipmentBillingRequirement, "satisfied" | "ineligibleDocuments">;
  className?: string;
};

export function UnmetRequirementNote({ requirement, className }: UnmetRequirementNoteProps) {
  const t = useT();
  const standing = unmetRequirementStanding(requirement);

  if (!standing) {
    return null;
  }

  const message = {
    rejected: t("Uploaded copy was rejected"),
    expired: t("Uploaded copy has expired"),
    pending_review: t("Uploaded copy is awaiting review"),
    inactive: t("Uploaded copy is archived"),
  }[standing];

  return <p className={cn("text-2xs text-danger-foreground", className)}>{message}</p>;
}
