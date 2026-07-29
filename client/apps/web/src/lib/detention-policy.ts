import type { DetentionPolicy } from "@trenova/shared/types/detention";

/**
 * Fields that identify or scope a policy without changing a single cent of what
 * it charges. Typing a description must not re-run the pricing engine, so these
 * are excluded from the fingerprint below.
 */
const NON_PRICING_FIELDS = new Set<string>([
  "id",
  "organizationId",
  "businessUnitId",
  "version",
  "createdAt",
  "updatedAt",
  "name",
  "code",
  "description",
  "comments",
  "status",
  "priority",
  "specificityScore",
  "isOrgDefault",
  "customerId",
  "locationId",
  "shipmentTypeIds",
  "serviceTypeIds",
  "commodityIds",
  "stopTypes",
  "appointmentStopsOnly",
  "effectiveStartDate",
  "effectiveEndDate",
]);

/**
 * A stable identity for the priced terms of a policy.
 *
 * Used as a query key so the preview re-prices when the math changes and stays
 * put when it does not, and to detect that a backtest result no longer reflects
 * the terms on screen. Anything not explicitly excluded counts, so a new pricing
 * field is covered without touching this list.
 */
export function pricingFingerprint(policy: Partial<DetentionPolicy>): string {
  return JSON.stringify(
    Object.entries(policy)
      .filter(([key]) => !NON_PRICING_FIELDS.has(key))
      .sort(([left], [right]) => left.localeCompare(right)),
  );
}
