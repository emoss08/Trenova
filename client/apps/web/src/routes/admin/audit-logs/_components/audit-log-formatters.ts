import { translate } from "@trenova/shared/i18n/runtime";
import type { BadgeVariant } from "@trenova/shared/components/ui/badge";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import type { SelectOption } from "@trenova/shared/types/fields";
import { defineLabels, sourceLabels, translateLabel } from "@trenova/shared/i18n/labels";
import {
  formatMinor,
  formatMinorAmount,
  isMinorAmountField,
  withoutMinorSuffix,
} from "@trenova/shared/lib/money";

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

function isEmptyValue(value: unknown): boolean {
  return value === undefined || value === null;
}

/**
 * How a field changed, from the server's diff (`created`, `updated`, `deleted` in
 * shared/jsonutils): an update from or to null reads as filled in or cleared, not as a value.
 */
function changeTypeOf(serverType: string | undefined, from: unknown, to: unknown): AuditChangeType {
  if (serverType === "created") return "added";
  if (serverType === "deleted") return "removed";
  if (isEmptyValue(from) && !isEmptyValue(to)) return "added";
  if (!isEmptyValue(from) && isEmptyValue(to)) return "removed";
  return "changed";
}

export function normalizeAuditChanges(changes: Record<string, unknown>): NormalizedAuditChange[] {
  return Object.entries(changes)
    .map(([key, value]) => {
      if (isRecord(value)) {
        const from = value.from;
        const to = value.to;
        const path = toTrimmedString(value.path) ?? key;
        const type = changeTypeOf(toTrimmedString(value.type), from, to);

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
        type: changeTypeOf(undefined, undefined, value),
      };
    })
    .sort((left, right) => left.path.localeCompare(right.path));
}

const operationLabels = defineLabels({
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
} as const);

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
  label: sourceLabels(operationLabels)[value],
}));

export function operationLabel(operation: string) {
  const normalized = operation?.toLowerCase?.() || "";
  return operationLabels[normalized as keyof typeof operationLabels] || operation;
}

export function operationVariant(operation: string): BadgeVariant {
  const normalized = operation?.toLowerCase?.() || "";

  switch (normalized) {
    case "create":
      return "success";
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
      return "info";
    case "read":
      return "neutral";
    default:
      return "neutral";
  }
}

/** resourceSourceLabel is a resource key in English words, for data built once at import. */
export function resourceSourceLabel(resource: string) {
  return resource
    .split("_")
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

export function resourceLabel(resource: string) {
  return translateLabel(resourceSourceLabel(resource));
}

/** auditEntryTitle names an entry with no comment by what was done to what. */
export function auditEntryTitle(operation: string, resource: string) {
  return translate("{0} · {1}", resourceLabel(resource), operationLabel(operation));
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
  const words = withoutMinorSuffix(path)
    .replace(/([a-z0-9])([A-Z])/g, "$1 $2")
    .replace(/([A-Z]+)([A-Z][a-z])/g, "$1 $2")
    .split(/[\s._]+/)
    .filter(Boolean)
    .map((word) => (/^[A-Z0-9]+$/.test(word) && word.length > 1 ? word : word.toLowerCase()));
  if (words.length === 0) return path;
  return [words[0].charAt(0).toUpperCase() + words[0].slice(1), ...words.slice(1)].join(" ");
}

export function formatAuditValue(value: unknown): string {
  if (value === undefined) return translate("Not set");
  if (value === null) return "null";
  if (typeof value === "string") return value.length === 0 ? translate("Empty string") : value;
  if (typeof value === "number" || typeof value === "bigint") {
    return String(value);
  }
  if (typeof value === "boolean") {
    return value ? "true" : "false";
  }
  if (Array.isArray(value)) {
    return value.length === 0
      ? translate("Empty array")
      : translate("{0, plural, one {Array with # item} other {Array with # items}}", value.length);
  }
  if (isRecord(value)) {
    const keys = Object.keys(value);
    return keys.length === 0
      ? translate("Empty object")
      : translate(
          "{0, plural, one {Object with # field} other {Object with # fields}}",
          keys.length,
        );
  }
  return JSON.stringify(value) ?? "";
}

function looksLikeDatePath(path?: string): boolean {
  if (!path) return false;
  const snaked = path.replace(/([a-z])([A-Z])/g, "$1_$2").toLowerCase();
  return /(?:^|[._])(date|time|timestamp|at|eta|etd|start|end)(?:$|[._])/i.test(snaked);
}

/**
 * A changed field's value as a person reads it: an instant as a date, money in minor units as
 * an amount (with its currency when the record names one), anything else as it was stored.
 */
export function formatAuditFieldValue(
  value: unknown,
  path?: string,
  currency?: string | null,
): { value: string; transformed: boolean; hint?: string } {
  if (typeof value === "number" && isMinorAmountField(path)) {
    return {
      value: currency ? formatMinor(value, currency) : formatMinorAmount(value),
      transformed: true,
      hint: translate("Minor units"),
    };
  }

  const isDatePath = looksLikeDatePath(path);

  if (typeof value === "number" && isDatePath) {
    return {
      value: formatToUserTimezone(value, { showTimeZone: true }),
      transformed: true,
      hint: translate("Unix timestamp"),
    };
  }

  if (typeof value === "string" && isDatePath) {
    const asNumber = Number(value);
    if (!Number.isNaN(asNumber) && Number.isFinite(asNumber)) {
      return {
        value: formatToUserTimezone(asNumber, { showTimeZone: true }),
        transformed: true,
        hint: translate("Unix timestamp"),
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

  return normalized.startsWith("[REDACTED]") || /^\*{4,}$/.test(normalized);
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

export const AUDIT_CHANNEL_LABELS = defineLabels({
  web: "Web app",
  api: "API key",
  portal: "Portal",
  agent_conversation: "Agent, in a conversation",
  agent_run: "Agent, on its own",
  support: "Trenova support",
  scim: "SCIM directory",
  job: "Background job",
  cli: "Command line",
  system: "System",
} as const);

export function channelLabel(channel: string): string {
  return (AUDIT_CHANNEL_LABELS as Record<string, string>)[channel] ?? channel;
}

const CURRENCY_CODE = /^[A-Z]{3}$/;

/** The currency a record's saved state names (`currencyCode`), after the change or before it. */
export function recordCurrency(currentState: unknown, previousState: unknown): string | null {
  for (const state of [currentState, previousState]) {
    const code = isRecord(state) ? state.currencyCode : undefined;
    if (typeof code === "string" && CURRENCY_CODE.test(code)) return code;
  }
  return null;
}
