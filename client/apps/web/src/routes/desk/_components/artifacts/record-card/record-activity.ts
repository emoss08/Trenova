import type { ActivityEntry, RecordActivityPage } from "@/lib/graphql/record-activity";
import { recordActivityQuery } from "@/lib/queries/record-activity";
import { normalizeAuditChanges } from "@/routes/admin/audit-logs/_components/audit-log-formatters";
import { defineLabels } from "@trenova/shared/i18n/labels";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { toTitleCase } from "@trenova/shared/lib/utils";
import { useInfiniteQuery } from "@tanstack/react-query";
import { usePagedRows, type PagedRows } from "./paged-rows";

/** How many changed fields an entry names before it counts the rest. */
const NAMED_FIELDS = 3;

/** What was done to a record, in the past tense the activity reads in. */
const DONE_LABELS = defineLabels({
  create: "Created",
  update: "Updated",
  delete: "Deleted",
  export: "Exported",
  import: "Imported",
  approve: "Approved",
  reject: "Rejected",
  assign: "Assigned",
  unassign: "Unassigned",
  archive: "Archived",
  restore: "Restored",
  submit: "Submitted",
  cancel: "Canceled",
  duplicate: "Duplicated",
  close: "Closed",
  lock: "Locked",
  unlock: "Unlocked",
  activate: "Activated",
  reopen: "Reopened",
});

/** Short forms that keep their capitals in a sentence. */
const INITIALISMS = new Set([
  "bol",
  "po",
  "pro",
  "id",
  "dot",
  "mc",
  "scac",
  "vin",
  "edi",
  "ach",
  "ein",
]);

/**
 * A field path as a sentence names it, from its last part: "poNumber" is "PO
 * number" and "bill_to_customer_id" is "bill to customer ID".
 */
export function fieldName(path: string): string {
  const last = path.split(".").filter(Boolean).at(-1) ?? path;
  return last
    .replace(/([a-z0-9])([A-Z])/gu, "$1 $2")
    .replace(/_/gu, " ")
    .toLowerCase()
    .split(" ")
    .filter(Boolean)
    .map((word) => (INITIALISMS.has(word) ? word.toUpperCase() : word))
    .join(" ");
}

/**
 * One entry as a sentence: what was done, and for a change, the fields it
 * touched, the first few by name and the rest counted.
 */
export function activitySentence(
  entry: Pick<ActivityEntry, "operation" | "changes">,
  t: TranslateFn,
): string {
  const operation = entry.operation.toLowerCase();
  if (operation === "update" && entry.changes && typeof entry.changes === "object") {
    const fields = [
      ...new Set(
        normalizeAuditChanges(entry.changes as Record<string, unknown>).map((change) =>
          fieldName(change.path),
        ),
      ),
    ];
    if (fields.length > NAMED_FIELDS) {
      return t(
        "Changed {0} and {1, plural, one {# more} other {# more}}",
        fields.slice(0, NAMED_FIELDS).join(", "),
        fields.length - NAMED_FIELDS,
      );
    }
    if (fields.length > 0) {
      return t("Changed {0}", fields.join(", "));
    }
  }
  return Object.hasOwn(DONE_LABELS, operation)
    ? DONE_LABELS[operation as keyof typeof DONE_LABELS]
    : toTitleCase(operation);
}

/** Who did it: the person, or the system when no one did. */
export function activityActor(
  entry: Pick<ActivityEntry, "user" | "principalType">,
  t: TranslateFn,
): string {
  const name = entry.user?.name?.trim();
  if (name) return name;
  switch (entry.principalType) {
    case "api_key":
      return t("An API key");
    case "capture_device":
      return t("A capture device");
    default:
      return t("Trenova");
  }
}

/** What was done to a record, newest first, a page at a time, read only when `enabled`. */
export function useRecordActivity({
  resourceId,
  pageSize,
  enabled,
}: {
  resourceId: string;
  pageSize: number;
  enabled: boolean;
}): PagedRows<ActivityEntry> {
  return usePagedRows<ActivityEntry, RecordActivityPage>(
    useInfiniteQuery({
      ...recordActivityQuery(resourceId, pageSize),
      enabled,
      retry: false,
    }),
  );
}
