import { Alert } from "@trenova/shared/components/ui/alert";
import type { ComponentProps } from "react";

/**
 * The one notice the review pane draws above an item's tabs: billing notes, an exception, a
 * detention or rating hold, an adjusted invoice's rebill. Every one is the same blue, so none
 * reads as louder than another and the biller reads them as one list of things to know.
 */
export function BillingQueueNotice(props: Omit<ComponentProps<typeof Alert>, "variant" | "size">) {
  return <Alert size="sm" variant="info" {...props} />;
}
