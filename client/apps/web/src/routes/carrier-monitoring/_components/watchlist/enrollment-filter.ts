import type { CarrierMonitoringEnrollmentRow } from "@/lib/graphql/carrier-monitoring-table";

export function monitorableCarrierIds(rows: readonly CarrierMonitoringEnrollmentRow[]): string[] {
  const ids = new Set<string>();
  for (const row of rows) {
    if (row.carrierId) {
      ids.add(row.carrierId);
    }
  }
  return [...ids];
}
