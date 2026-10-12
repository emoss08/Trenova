import { notifyBulkOutcome, type BulkOutcomeMessages } from "@/lib/bulk-outcome";
import type { EDIBulkActionResult } from "@trenova/shared/types/edi";

export function notifyEDIBulkOutcome(result: EDIBulkActionResult, messages: BulkOutcomeMessages) {
  notifyBulkOutcome(result, messages);
}
