import type {
  WorkerTrainingRecordRow,
  WorkerTrainingSummaryItem,
} from "@/lib/graphql/worker-training";
import type { WorkerTrainingStatus } from "@trenova/shared/types/worker-training";

/** The statuses a record ends in; the rest are an open assignment. */
const CLOSED_STATUSES: ReadonlySet<WorkerTrainingStatus> = new Set([
  "Completed",
  "Failed",
  "Expired",
  "Waived",
  "Cancelled",
]);

/** The record a course's slot stands on, if any. */
export function slotRecord(item: WorkerTrainingSummaryItem): WorkerTrainingRecordRow | null {
  return (item.record as WorkerTrainingRecordRow | null | undefined) ?? null;
}

export function isOpenRecord(record: WorkerTrainingRecordRow | null | undefined): boolean {
  return record?.status === "Assigned" || record?.status === "InProgress";
}

/** The courses with an open assignment, which cannot be assigned again. */
export function openCourseIdsOf(items: readonly WorkerTrainingSummaryItem[]): Set<string> {
  return new Set(
    items.filter((item) => isOpenRecord(slotRecord(item))).map((item) => item.course.id),
  );
}

/**
 * The earlier records folded under the table: closed ones that no course's slot stands on, so a
 * record is never listed twice.
 */
export function earlierRecords(
  records: readonly WorkerTrainingRecordRow[],
  items: readonly WorkerTrainingSummaryItem[],
): WorkerTrainingRecordRow[] {
  const inSlots = new Set(items.map((item) => slotRecord(item)?.id));
  return records.filter(
    (record) =>
      CLOSED_STATUSES.has(record.status as WorkerTrainingStatus) && !inSlots.has(record.id),
  );
}

/** Every training record the tab has read, by id: the summary's slots and the loaded history. */
export function trainingRecordsById(
  items: readonly WorkerTrainingSummaryItem[] | undefined,
  records: readonly WorkerTrainingRecordRow[] | undefined,
): Map<string, WorkerTrainingRecordRow> {
  const byId = new Map<string, WorkerTrainingRecordRow>();
  for (const record of records ?? []) byId.set(record.id, record);
  for (const item of items ?? []) {
    const record = slotRecord(item);
    if (record) byId.set(record.id, record);
  }
  return byId;
}
