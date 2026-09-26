import type { AccountingInboundChangeStatus } from "@trenova/graphql/generated/graphql";
import type { FieldFilter } from "@trenova/shared/types/data-table";

export type InboundBucket = "proposed" | "detected" | "applied" | "ignored";

const BUCKET_STATUS: Record<InboundBucket, AccountingInboundChangeStatus> = {
  proposed: "Proposed",
  detected: "Detected",
  applied: "Applied",
  ignored: "Ignored",
};

export function inboundBucketFilter(bucket: InboundBucket): FieldFilter[] {
  return [{ field: "status", operator: "in", value: [BUCKET_STATUS[bucket]] }];
}

export function activeInboundBucket(filters: readonly FieldFilter[]): InboundBucket | null {
  if (filters.length !== 1) {
    return null;
  }
  const [filter] = filters;
  if (filter.field !== "status") {
    return null;
  }
  const values =
    filter.operator === "in" && Array.isArray(filter.value)
      ? (filter.value as unknown[])
      : filter.operator === "eq"
        ? [filter.value]
        : [];
  if (values.length !== 1) {
    return null;
  }
  const entry = (Object.entries(BUCKET_STATUS) as [InboundBucket, string][]).find(
    ([, status]) => status === values[0],
  );
  return entry ? entry[0] : null;
}
