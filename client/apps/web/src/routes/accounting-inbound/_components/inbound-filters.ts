import { activeTableBucket, tableBucketFilters, type TableBucketSpec } from "@/lib/table-buckets";
import type { FieldFilter } from "@trenova/shared/types/data-table";

export type InboundBucket = "proposed" | "detected" | "applied" | "ignored";

const BUCKETS: Record<InboundBucket, TableBucketSpec> = {
  proposed: { status: ["Proposed"] },
  detected: { status: ["Detected"] },
  applied: { status: ["Applied"] },
  ignored: { status: ["Ignored"] },
};

export function inboundBucketFilter(bucket: InboundBucket): FieldFilter[] {
  return tableBucketFilters(BUCKETS[bucket]);
}

export function activeInboundBucket(filters: readonly FieldFilter[]): InboundBucket | null {
  return activeTableBucket(filters, BUCKETS);
}
