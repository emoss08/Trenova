import { notifyBulkOutcome, type BulkOutcomeLabels } from "@/lib/bulk-outcome";
import type { EDIBulkActionResult } from "@trenova/shared/types/edi";

export function notifyEDIBulkOutcome(result: EDIBulkActionResult, labels: BulkOutcomeLabels) {
  notifyBulkOutcome(result, labels);
}
