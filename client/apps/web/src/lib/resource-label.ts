import { defineLabels } from "@trenova/shared/i18n/labels";
import { toTitleCase } from "@trenova/shared/lib/utils";

/**
 * Resources whose title-cased name reads wrong. Title case turns an
 * abbreviation into a word — worker_pto becomes "Worker Pto" — and this is the
 * label an administrator reads while deciding what an agent may touch.
 */
const RESOURCE_LABELS: Record<string, string> = defineLabels({
  worker_pto: "Worker time off",
  hazardous_material: "Hazardous materials",
  bqi: "Billing queue",
  home_layout_preset: "Home page",
});

/** A permission resource ("billing_queue") in words ("Billing Queue"). */
export function resourceLabel(resource: string): string {
  const key = resource || "general";
  return RESOURCE_LABELS[key] ?? toTitleCase(key.replace(/[_-]+/g, " "));
}
