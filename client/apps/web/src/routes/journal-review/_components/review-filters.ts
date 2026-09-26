import type { JournalReviewStatus } from "@/hooks/use-journal-review-labels";
import { activeTableBucket, tableBucketFilters, type TableBucketSpec } from "@/lib/table-buckets";
import type { FieldFilter } from "@trenova/shared/types/data-table";
import type { StatusPhase } from "@trenova/shared/lib/status-phase";

export type JournalReviewBucket = "awaiting" | "ready";

const BUCKETS: Record<JournalReviewBucket, TableBucketSpec> = {
  awaiting: { status: ["Pending"] },
  ready: { status: ["Approved"] },
};

export function journalReviewBucketFilter(bucket: JournalReviewBucket): FieldFilter[] {
  return tableBucketFilters(BUCKETS[bucket]);
}

export function activeJournalReviewBucket(
  filters: readonly FieldFilter[],
): JournalReviewBucket | null {
  return activeTableBucket(filters, BUCKETS);
}

export function journalReviewPhase(status: string): StatusPhase {
  return (status as JournalReviewStatus) === "Approved" ? "queued" : "awaiting";
}
