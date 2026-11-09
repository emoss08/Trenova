import { translate } from "@trenova/shared/i18n/runtime";
import {
  formatAuditFieldValue,
  formatFieldLabel,
  isRecordValue,
  normalizeAuditChanges,
  operationLabel,
  resourceLabel,
} from "./audit-log-formatters";

/** Who wrote an entry, by the channel it arrived through: the chart's four series. */
export type AuditActorKind = "people" | "agents" | "integrations" | "jobs";

const AGENT_CHANNELS = new Set(["agent_conversation", "agent_run"]);
const INTEGRATION_CHANNELS = new Set(["api", "scim"]);
const JOB_CHANNELS = new Set(["job", "cli", "system"]);

export function actorKindOfChannel(channel: string): AuditActorKind {
  if (AGENT_CHANNELS.has(channel)) return "agents";
  if (INTEGRATION_CHANNELS.has(channel)) return "integrations";
  if (JOB_CHANNELS.has(channel)) return "jobs";
  return "people";
}

export function actorKindOfPrincipal(principalType: string): AuditActorKind {
  switch (principalType) {
    case "agent":
      return "agents";
    case "api_key":
      return "integrations";
    case "system":
      return "jobs";
    default:
      return "people";
  }
}

type SentenceEntry = { operation: string; resource: string; entityRef?: string | null };

/** What happened, as the trail's sentence: the operation done to the record it names. */
export function entrySentence(entry: SentenceEntry): string {
  const subject = entry.entityRef?.trim() || resourceLabel(entry.resource);
  switch (entry.operation?.toLowerCase()) {
    case "create":
      return translate("Created {0}", subject);
    case "update":
      return translate("Updated {0}", subject);
    case "delete":
      return translate("Deleted {0}", subject);
    case "read":
      return translate("Opened {0}", subject);
    case "export":
      return translate("Exported {0}", subject);
    case "import":
      return translate("Imported {0}", subject);
    case "approve":
      return translate("Approved {0}", subject);
    case "reject":
      return translate("Rejected {0}", subject);
    case "assign":
      return translate("Assigned {0}", subject);
    case "unassign":
      return translate("Unassigned {0}", subject);
    case "archive":
      return translate("Archived {0}", subject);
    case "restore":
      return translate("Restored {0}", subject);
    case "submit":
      return translate("Submitted {0}", subject);
    case "cancel":
      return translate("Cancelled {0}", subject);
    case "duplicate":
      return translate("Duplicated {0}", subject);
    case "close":
      return translate("Closed {0}", subject);
    case "lock":
      return translate("Locked {0}", subject);
    case "unlock":
      return translate("Unlocked {0}", subject);
    case "activate":
      return translate("Activated {0}", subject);
    case "reopen":
      return translate("Reopened {0}", subject);
    default:
      return translate("{0} · {1}", operationLabel(entry.operation), subject);
  }
}

/** One changed field as the trail shows it: its label and its value before and after. */
export type ChangeChip = {
  path: string;
  field: string;
  /** Null when the field was added. */
  from: string | null;
  /** Null when the field was removed. */
  to: string | null;
};

const CHIP_VALUE_LIMIT = 40;

function short(value: string): string {
  return value.length > CHIP_VALUE_LIMIT ? `${value.slice(0, CHIP_VALUE_LIMIT - 1)}…` : value;
}

/** The first `limit` changed fields of an entry, and how many more it changed. */
export function changeChips(
  changes: unknown,
  limit: number,
): { chips: ChangeChip[]; more: number } {
  if (!isRecordValue(changes)) return { chips: [], more: 0 };
  const all = normalizeAuditChanges(changes);
  const chips = all.slice(0, limit).map((change) => ({
    path: change.path,
    field: formatFieldLabel(change.path),
    from:
      change.type === "added" ? null : short(formatAuditFieldValue(change.from, change.path).value),
    to:
      change.type === "removed" ? null : short(formatAuditFieldValue(change.to, change.path).value),
  }));
  return { chips, more: Math.max(all.length - chips.length, 0) };
}
