import type { EDIBulkActionResult } from "@/types/edi";
import { toast } from "sonner";

type EDIBulkOutcomeLabels = {
  entity: string;
  verbPast: string;
  skipped?: number;
};

export function notifyEDIBulkOutcome(
  result: EDIBulkActionResult,
  { entity, verbPast, skipped = 0 }: EDIBulkOutcomeLabels,
) {
  const skippedSuffix = skipped > 0 ? ` (${skipped} ineligible skipped)` : "";
  if (result.failed.length === 0) {
    toast.success(`${verbPast} ${result.succeeded.length} ${entity}(s)${skippedSuffix}`);
    return;
  }

  const description = result.failed
    .slice(0, 3)
    .map((failure) => failure.error)
    .join("; ");
  if (result.succeeded.length === 0) {
    toast.error(`All ${result.failed.length} selected ${entity}(s) failed${skippedSuffix}`, {
      description,
    });
    return;
  }
  toast.warning(
    `${verbPast} ${result.succeeded.length} ${entity}(s); ${result.failed.length} failed${skippedSuffix}`,
    { description },
  );
}
