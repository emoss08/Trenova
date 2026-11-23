import type {
  WorkerCredentialRow,
  WorkerCredentialSummary,
  WorkerCredentialSummaryItem,
} from "@/lib/graphql/worker-credential";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { CREDENTIAL_CATEGORY_LABELS } from "@trenova/shared/types/worker-credential";

export type BadgeTone = "success" | "warning" | "danger" | "neutral";

/** Held credentials nobody has checked against the document yet. */
export function unverifiedCount(summary: WorkerCredentialSummary): number {
  return summary.items.filter((item) => item.credential && !item.credential.verifiedAt).length;
}

/** The table's two groups: every required slot (held or not), then the other credentials held. */
export function credentialGroups(summary: WorkerCredentialSummary): {
  required: WorkerCredentialSummaryItem[];
  other: WorkerCredentialSummaryItem[];
} {
  return {
    required: summary.items.filter((item) => item.required),
    other: summary.items.filter((item) => !item.required && item.credential),
  };
}

export function healthTone(health: string): BadgeTone {
  switch (health) {
    case "Valid":
      return "success";
    case "ExpiringSoon":
      return "warning";
    case "Expired":
    case "Missing":
      return "danger";
    default:
      return "neutral";
  }
}

export function healthLabel(t: TranslateFn, health: string): string {
  switch (health) {
    case "Valid":
      return t("Valid");
    case "ExpiringSoon":
      return t("Expiring soon");
    case "Expired":
      return t("Expired");
    default:
      return t("Missing");
  }
}

/** How far off an expiry is, in the words the table and the read pane use. */
export function expiryPhrase(t: TranslateFn, days: number | null | undefined): string {
  if (days == null) return t("No expiry");
  if (days === 0) return t("Today");
  if (days < 0) return t("{0, plural, one {# day ago} other {# days ago}}", -days);
  if (days <= 365) return t("{0, plural, one {in # day} other {in # days}}", days);
  return t("{0, plural, one {in # year} other {in # years}}", Math.floor(days / 365));
}

export function expiryToneClass(health: string): string {
  if (health === "Expired") return "text-danger-foreground";
  if (health === "ExpiringSoon") return "text-warning-foreground";
  return "text-muted-foreground";
}

/** Category, then what is printed on the card, then who issued it. */
export function credentialCaption(
  item: Pick<WorkerCredentialSummaryItem, "credentialType"> & {
    credential?: Pick<WorkerCredentialRow, "number" | "issuingAuthority"> | null;
  },
): { category: string; number: string | null; authority: string | null } {
  const category = item.credentialType.category;
  return {
    category: CREDENTIAL_CATEGORY_LABELS[category as keyof typeof CREDENTIAL_CATEGORY_LABELS] ?? "",
    number: item.credential?.number || null,
    authority: item.credential?.issuingAuthority || null,
  };
}

/** Archived credentials of one type, newest first: the read pane's earlier versions. */
export function earlierVersions(
  credentials: readonly WorkerCredentialRow[],
  credential: Pick<WorkerCredentialRow, "id" | "credentialTypeId">,
): WorkerCredentialRow[] {
  return credentials
    .filter(
      (row) =>
        row.id !== credential.id &&
        row.status === "Archived" &&
        row.credentialTypeId === credential.credentialTypeId,
    )
    .sort((a, b) => (b.archivedAt ?? 0) - (a.archivedAt ?? 0));
}
