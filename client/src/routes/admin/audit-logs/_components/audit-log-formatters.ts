import type { BadgeVariant } from "@/components/ui/badge";
import { formatToUserTimezone } from "@/lib/date";
import type { SelectOption } from "@/types/fields";

type AuditChangeType = "added" | "removed" | "changed";

export type NormalizedAuditChange = {
  path: string;
  from: unknown;
  to: unknown;
  type: AuditChangeType;
  fieldType?: string;
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === "object" && !Array.isArray(value);
}

export function isRecordValue(value: unknown): value is Record<string, unknown> {
  return isRecord(value);
}

function toTrimmedString(value: unknown): string | undefined {
  if (typeof value !== "string") {
    return undefined;
  }

  const normalized = value.trim();
  return normalized.length > 0 ? normalized : undefined;
}

function inferChangeType(from: unknown, to: unknown): AuditChangeType {
  if (from === undefined && to !== undefined) {
    return "added";
  }

  if (from !== undefined && to === undefined) {
    return "removed";
  }

  return "changed";
}

export function normalizeAuditChanges(changes: Record<string, unknown>): NormalizedAuditChange[] {
  return Object.entries(changes)
    .map(([key, value]) => {
      if (isRecord(value)) {
        const from = value.from;
        const to = value.to;
        const path = toTrimmedString(value.path) ?? key;
        const explicitType = toTrimmedString(value.type);
        const type =
          explicitType === "added" || explicitType === "removed" || explicitType === "changed"
            ? explicitType
            : inferChangeType(from, to);

        return {
          path,
          from,
          to,
          type,
          fieldType: toTrimmedString(value.fieldType),
        };
      }

      return {
        path: key,
        from: undefined,
        to: value,
        type: inferChangeType(undefined, value),
      };
    })
    .sort((left, right) => left.path.localeCompare(right.path));
}

const operationLabels = {
  read: "Read",
  create: "Create",
  update: "Update",
  delete: "Delete",
  export: "Export",
  import: "Import",
  approve: "Approve",
  reject: "Reject",
  assign: "Assign",
  unassign: "Unassign",
  archive: "Archive",
  restore: "Restore",
  submit: "Submit",
  cancel: "Cancel",
  duplicate: "Duplicate",
  close: "Close",
  lock: "Lock",
  unlock: "Unlock",
  activate: "Activate",
  reopen: "Reopen",
} as const;

const operationFilterOrder: (keyof typeof operationLabels)[] = [
  "read",
  "create",
  "update",
  "delete",
  "export",
  "import",
  "approve",
  "reject",
  "assign",
  "unassign",
  "archive",
  "restore",
  "submit",
  "cancel",
  "duplicate",
  "close",
  "lock",
  "unlock",
  "activate",
  "reopen",
];

export const auditOperationFilterOptions: SelectOption[] = operationFilterOrder.map((value) => ({
  value,
  label: operationLabels[value],
}));

export function operationLabel(operation: string) {
  const normalized = operation?.toLowerCase?.() || "";
  return operationLabels[normalized as keyof typeof operationLabels] || operation;
}

export function operationVariant(operation: string): BadgeVariant {
  const normalized = operation?.toLowerCase?.() || "";

  switch (normalized) {
    case "create":
      return "active";
    case "update":
      return "info";
    case "archive":
    case "cancel":
    case "reject":
    case "delete":
    case "close":
      return "warning";
    case "restore":
    case "approve":
    case "unlock":
    case "activate":
    case "reopen":
      return "teal";
    case "read":
      return "outline";
    default:
      return "secondary";
  }
}

export function resourceLabel(resource: string) {
  return resource
    .split("_")
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

export function userInitials(name?: string) {
  if (!name) return "U";

  return name
    .split(" ")
    .map((part) => part[0])
    .join("")
    .toUpperCase()
    .slice(0, 2);
}

export function formatFieldLabel(path: string) {
  return path
    .split(/[._]/)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

export function changeTypeLabel(type: AuditChangeType) {
  switch (type) {
    case "added":
      return "Added";
    case "removed":
      return "Removed";
    default:
      return "Changed";
  }
}

export function changeTypeVariant(type: AuditChangeType): BadgeVariant {
  switch (type) {
    case "added":
      return "active";
    case "removed":
      return "warning";
    default:
      return "info";
  }
}

export function formatAuditValue(value: unknown): string {
  if (value === undefined) return "Not set";
  if (value === null) return "null";
  if (typeof value === "string") return value.length === 0 ? "Empty string" : value;
  if (typeof value === "number" || typeof value === "bigint") {
    return String(value);
  }
  if (typeof value === "boolean") {
    return value ? "true" : "false";
  }
  if (Array.isArray(value)) {
    return value.length === 0
      ? "Empty array"
      : `Array with ${value.length} item${value.length === 1 ? "" : "s"}`;
  }
  if (isRecord(value)) {
    const keys = Object.keys(value);
    return keys.length === 0
      ? "Empty object"
      : `Object with ${keys.length} field${keys.length === 1 ? "" : "s"}`;
  }
  return JSON.stringify(value) ?? "";
}

function looksLikeDatePath(path?: string): boolean {
  if (!path) return false;
  const snaked = path.replace(/([a-z])([A-Z])/g, "$1_$2").toLowerCase();
  return /(?:^|[._])(date|time|timestamp|at|eta|etd|start|end)(?:$|[._])/i.test(snaked);
}

export function formatAuditValueWithDates(
  value: unknown,
  path?: string,
): { value: string; transformed: boolean; hint?: string } {
  const isDatePath = looksLikeDatePath(path);

  if (typeof value === "number" && isDatePath) {
    return {
      value: formatToUserTimezone(value, { showTimeZone: true }),
      transformed: true,
      hint: "Unix timestamp",
    };
  }

  if (typeof value === "string" && isDatePath) {
    const asNumber = Number(value);
    if (!Number.isNaN(asNumber) && Number.isFinite(asNumber)) {
      return {
        value: formatToUserTimezone(asNumber, { showTimeZone: true }),
        transformed: true,
        hint: "Unix timestamp",
      };
    }
  }

  return {
    value: formatAuditValue(value),
    transformed: false,
  };
}

export function isSensitiveOmittedValue(value: unknown): boolean {
  if (typeof value !== "string") {
    return false;
  }

  const normalized = value.trim();
  if (!normalized) {
    return false;
  }

  return normalized === "[REDACTED]" || normalized === "****" || /^\*{4,}$/.test(normalized);
}

export function containsSensitiveOmittedData(value: unknown): boolean {
  if (isSensitiveOmittedValue(value)) {
    return true;
  }

  if (typeof value === "string") {
    const normalized = value.trim();
    return normalized.includes("[REDACTED]") || /\*{4,}/.test(normalized);
  }

  if (Array.isArray(value)) {
    return value.some((entry) => containsSensitiveOmittedData(entry));
  }

  if (value && typeof value === "object") {
    return Object.values(value as Record<string, unknown>).some((entry) =>
      containsSensitiveOmittedData(entry),
    );
  }

  return false;
}
