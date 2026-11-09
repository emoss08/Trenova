import type { AuditAlertKind } from "@trenova/graphql/generated/graphql";
import {
  Building02Icon,
  Download01Icon,
  FileX02Icon,
  Fingerprint04Icon,
  Key01Icon,
  Lock01Icon,
  Settings04Icon,
  Trash01Icon,
  UserCheck01Icon,
  type IconComponent,
} from "@trenova/shared/components/icons";
import { defineLabels } from "@trenova/shared/i18n/labels";

/** Each alert rule's short name, for the line under a review's title. */
export const REVIEW_KIND_LABELS = defineLabels({
  RoleChanged: "Role change",
  APIKeyChanged: "API key change",
  MFAReset: "Two-factor reset",
  IdentityConfigChanged: "Sign-in settings change",
  CrossTenantRefused: "Cross-organization request",
  MalwareRefused: "Malware refused",
  LegalHoldChanged: "Legal hold change",
  BulkDelete: "Many deletes",
  BulkExport: "Many exports",
} as const satisfies Record<AuditAlertKind, string>);

export const REVIEW_KIND_ICONS: Record<AuditAlertKind, IconComponent> = {
  RoleChanged: UserCheck01Icon,
  APIKeyChanged: Key01Icon,
  MFAReset: Fingerprint04Icon,
  IdentityConfigChanged: Settings04Icon,
  CrossTenantRefused: Building02Icon,
  MalwareRefused: FileX02Icon,
  LegalHoldChanged: Lock01Icon,
  BulkDelete: Trash01Icon,
  BulkExport: Download01Icon,
};

/** What could lose data or let someone in is a danger; a change of access is a warning. */
export function reviewTone(kind: AuditAlertKind): "danger" | "warning" {
  return kind === "BulkDelete" || kind === "CrossTenantRefused" || kind === "MalwareRefused"
    ? "danger"
    : "warning";
}
