import { describeBillingSchedule, type BillingSchedule } from "@/lib/billing-schedule";
import { ReceiptTextIcon } from "lucide-react";

/**
 * States the configured schedule as a sentence.
 *
 * The settings above answer three separate questions — cadence, how many
 * invoices, how they read — that used to be one overloaded dropdown. Saying the
 * combination out loud is how an operator can tell they picked what they meant.
 */
export function BillingSchedulePreview({ schedule }: { schedule: BillingSchedule }) {
  return (
    <div className="bg-muted/40 flex items-start gap-3 rounded-lg border p-3">
      <ReceiptTextIcon className="text-muted-foreground mt-0.5 size-4 shrink-0" />
      <div>
        <p className="text-muted-foreground text-xs font-medium">This customer is billed</p>
        <p className="mt-0.5 text-sm">{describeBillingSchedule(schedule)}</p>
      </div>
    </div>
  );
}
