import type { CarrierMonitoringEnrollmentRow } from "@/lib/graphql/carrier-monitoring-table";
import type { CarrierMonitoringEnrollmentFilterInput } from "@trenova/graphql/generated/graphql";
import { parseAsStringLiteral } from "nuqs";

export const ENROLLMENT_VIEWS = ["monitored", "attention", "failed", "all"] as const;
export type EnrollmentView = (typeof ENROLLMENT_VIEWS)[number];

export const enrollmentViewParser = parseAsStringLiteral(ENROLLMENT_VIEWS).withDefault("monitored");

export function enrollmentFilterForView(
  view: EnrollmentView,
): CarrierMonitoringEnrollmentFilterInput | null {
  switch (view) {
    case "monitored":
      return { desiredState: "Enrolled" };
    case "attention":
      return { vendorStates: ["PendingAdd", "PendingRemove", "Failed"] };
    case "failed":
      return { vendorStates: ["Failed"] };
    case "all":
      return null;
  }
}

export function monitorableCarrierIds(rows: readonly CarrierMonitoringEnrollmentRow[]): string[] {
  const ids = new Set<string>();
  for (const row of rows) {
    if (row.carrierId) {
      ids.add(row.carrierId);
    }
  }
  return [...ids];
}
