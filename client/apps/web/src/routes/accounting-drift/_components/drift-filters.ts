import { activeTableBucket, tableBucketFilters, type TableBucketSpec } from "@/lib/table-buckets";
import type { FieldFilter } from "@trenova/shared/types/data-table";

export type DriftBucket = "open" | "amount" | "gone" | "settled";

const BUCKETS: Record<DriftBucket, TableBucketSpec> = {
  open: { status: ["Open"] },
  amount: { status: ["Open"], kind: ["AmountMismatch", "CustomerBalanceMismatch"] },
  gone: { status: ["Open"], kind: ["DeletedInProvider", "VoidedInProvider"] },
  settled: { status: ["Resolved", "Dismissed"] },
};

export function driftBucketFilter(bucket: DriftBucket): FieldFilter[] {
  return tableBucketFilters(BUCKETS[bucket]);
}

export function activeDriftBucket(filters: readonly FieldFilter[]): DriftBucket | null {
  return activeTableBucket(filters, BUCKETS);
}
