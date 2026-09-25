import type { AccountingSyncRecordStatus } from "@trenova/graphql/generated/graphql";
import type { FieldFilter } from "@trenova/shared/types/data-table";

export type LedgerBucket = "sent" | "moving" | "held" | "attention";

export const LEDGER_BUCKETS: Record<LedgerBucket, readonly AccountingSyncRecordStatus[]> = {
  sent: ["Synced"],
  moving: ["Queued", "InFlight", "Retrying"],
  held: ["AwaitingApproval"],
  attention: ["Blocked", "DeadLettered"],
};

export function bucketCount(
  counts: readonly { status: AccountingSyncRecordStatus; count: number }[],
  bucket: LedgerBucket,
): number {
  const statuses = LEDGER_BUCKETS[bucket];
  return counts.reduce(
    (total, item) => (statuses.includes(item.status) ? total + item.count : total),
    0,
  );
}

export function bucketFilter(bucket: LedgerBucket): FieldFilter[] {
  return [{ field: "status", operator: "in", value: [...LEDGER_BUCKETS[bucket]] }];
}

export function activeBucket(filters: readonly FieldFilter[]): LedgerBucket | null {
  if (filters.length !== 1) {
    return null;
  }
  const [filter] = filters;
  if (filter.field !== "status" || filter.operator !== "in" || !Array.isArray(filter.value)) {
    return null;
  }
  const values = [...(filter.value as string[])].sort();
  for (const [bucket, statuses] of Object.entries(LEDGER_BUCKETS) as [
    LedgerBucket,
    readonly AccountingSyncRecordStatus[],
  ][]) {
    const expected = [...statuses].sort();
    if (
      expected.length === values.length &&
      expected.every((value, idx) => value === values[idx])
    ) {
      return bucket;
    }
  }
  return null;
}
