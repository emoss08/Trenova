import { translate } from "@trenova/shared/i18n/runtime";
/**
 * Labels and tones for the driver qualification file. The words here mirror
 * the credential and document areas the file is assembled from, so a reader who
 * knows one knows this.
 */

export type DQFItemStatusValue =
  | "Satisfied"
  | "ExpiringSoon"
  | "Expired"
  | "Missing"
  | "Outstanding"
  | "NotApplicable";

export type DQFSectionValue = "Credentials" | "Documents" | "SafetyHistory" | "DrugAlcohol";

export type DQFTone = "active" | "inactive" | "warning" | "secondary";

export const DQF_SECTION_LABELS: Record<DQFSectionValue, string> = {
  Credentials: "Licences, medical and recurring reviews",
  Documents: "Documents on file",
  SafetyHistory: "Previous employers",
  DrugAlcohol: "Drug and alcohol",
};

export function dqfSectionLabel(section: string): string {
  return DQF_SECTION_LABELS[section as DQFSectionValue] ?? section;
}

/** The order the file reads in: evidence first, then the checks over it. */
export const DQF_SECTION_ORDER: DQFSectionValue[] = [
  "Credentials",
  "Documents",
  "SafetyHistory",
  "DrugAlcohol",
];

export const DQF_ITEM_STATUS_LABELS: Record<DQFItemStatusValue, string> = {
  Satisfied: "On file",
  ExpiringSoon: "Expiring soon",
  Expired: "Expired",
  Missing: "Missing",
  Outstanding: "Outstanding",
  NotApplicable: "Not applicable",
};

export function dqfItemStatusLabel(status: string): string {
  return DQF_ITEM_STATUS_LABELS[status as DQFItemStatusValue] ?? status;
}

export function dqfItemTone(status: string): DQFTone {
  switch (status) {
    case "Satisfied":
      return "active";
    case "ExpiringSoon":
      return "warning";
    case "Expired":
    case "Missing":
    case "Outstanding":
      return "inactive";
    default:
      return "secondary";
  }
}

/**
 * Whether an item stops the file being complete. Expiring soon warns: the
 * document on file is still valid today, and treating it as a gap would make
 * every file incomplete for a month before every renewal.
 */
export function dqfItemBlocks(status: string): boolean {
  return status === "Expired" || status === "Missing" || status === "Outstanding";
}

export const VERIFICATION_STATUS_LABELS: Record<string, string> = {
  Pending: "Not yet requested",
  Requested: "Awaiting response",
  Received: "Response received",
  NoResponse: "No response after follow-up",
  NotApplicable: "Not applicable",
};

export function verificationStatusLabel(status: string): string {
  return VERIFICATION_STATUS_LABELS[status] ?? status;
}

export function verificationTone(status: string): DQFTone {
  switch (status) {
    case "Received":
      return "active";
    case "NoResponse":
    case "NotApplicable":
      return "secondary";
    default:
      return "warning";
  }
}

/**
 * Whether this employer needs no further chasing. An employer who never
 * answers still settles the obligation: the rule asks for a good-faith effort
 * and a record of it, not for an answer nobody can compel.
 */
export function verificationSettled(status: string): boolean {
  return status === "Received" || status === "NoResponse" || status === "NotApplicable";
}

export const VERIFICATION_METHOD_LABELS: Record<string, string> = {
  Email: "Email",
  Fax: "Fax",
  Mail: "Mail",
  Phone: "Phone",
  Portal: "Portal",
  Other: "Other",
};

export function verificationMethodLabel(method: string): string {
  return VERIFICATION_METHOD_LABELS[method] ?? method;
}

/**
 * The three years 49 CFR 391.51(d) requires a file be held past termination.
 * The organisation's own setting can be longer; this is only the default the
 * server falls back to.
 */
export const DEFAULT_DQF_RETENTION_DAYS = 1095;

const SECONDS_IN_DAY = 86400;

/**
 * How long to leave a request unanswered before chasing again. 391.23 sets no
 * interval; it asks for a good-faith effort and a record of it. Two weeks is
 * long enough that a chase is not noise and short enough that the thirty-day
 * investigation window still holds a couple of them.
 */
export const DQF_FOLLOW_UP_INTERVAL_DAYS = 14;

/**
 * Chases on record before a silent employer can honestly be closed as
 * "no response". The number is the evidence an auditor reads.
 */
export const DQF_GOOD_FAITH_FOLLOW_UPS = 2;

export type DQFSectionProgress = {
  section: DQFSectionValue;
  total: number;
  satisfied: number;
  warning: number;
  blocking: number;
};

/** Per-section counts for the file spine. Not-applicable rows count for nothing. */
export function dqfSectionProgress(
  items: readonly { section: string; status: string }[],
): DQFSectionProgress[] {
  return DQF_SECTION_ORDER.map((section) => {
    const progress: DQFSectionProgress = {
      section,
      total: 0,
      satisfied: 0,
      warning: 0,
      blocking: 0,
    };
    for (const item of items) {
      if (item.section !== section || item.status === "NotApplicable") continue;
      progress.total += 1;
      if (item.status === "Satisfied") progress.satisfied += 1;
      else if (item.status === "ExpiringSoon") progress.warning += 1;
      else if (dqfItemBlocks(item.status)) progress.blocking += 1;
    }
    return progress;
  });
}

/** The worker-panel tab where a section's gaps are fixed; safety history is fixed here. */
export function dqfSectionTab(section: string): string | null {
  switch (section) {
    case "Credentials":
      return "credentials";
    case "Documents":
      return "documents";
    case "DrugAlcohol":
      return "testing";
    default:
      return null;
  }
}

export type VerificationStepAction =
  | "request"
  | "wait"
  | "chase"
  | "close"
  | "drugAlcohol"
  | "done";

export type VerificationNextStep = {
  action: VerificationStepAction;
  label: string;
  /** When the next chase falls due, for a step that is waiting. */
  dueAt?: number;
};

type VerificationLike = {
  status: string;
  requestedAt?: number | null;
  lastFollowUpAt?: number | null;
  followUpCount: number;
  wasDotRegulated: boolean;
  drugAlcoholResponseReceivedAt?: number | null;
};

/**
 * What the office should do next about one previous employer. The step is
 * derived from the dates on the record so the tab can say "chase again" on
 * the day it becomes true rather than waiting for somebody to notice.
 */
export function verificationNextStep(
  verification: VerificationLike,
  now: number,
): VerificationNextStep {
  switch (verification.status) {
    case "Pending":
      return { action: "request", label: translate("Send the request") };
    case "Requested": {
      const lastContact = verification.lastFollowUpAt ?? verification.requestedAt ?? now;
      const dueAt = lastContact + DQF_FOLLOW_UP_INTERVAL_DAYS * SECONDS_IN_DAY;
      if (now < dueAt) {
        return { action: "wait", label: translate("Waiting on the employer"), dueAt };
      }
      if (verification.followUpCount >= DQF_GOOD_FAITH_FOLLOW_UPS) {
        return { action: "close", label: translate("Close as no response") };
      }
      return { action: "chase", label: translate("Chase again") };
    }
    case "Received":
      if (verification.wasDotRegulated && !verification.drugAlcoholResponseReceivedAt) {
        return { action: "drugAlcohol", label: translate("Record the drug and alcohol history") };
      }
      return { action: "done", label: translate("Investigated") };
    default:
      return { action: "done", label: verificationStatusLabel(verification.status) };
  }
}

export type DQFStepKind = "item" | "verification" | "employers" | "purge";

export type DQFNextStep = {
  id: string;
  kind: DQFStepKind;
  label: string;
  detail: string;
  /** The panel tab that fixes it, when one does. */
  tab?: string | null;
  verificationId?: string;
  action?: VerificationStepAction | "addEmployer" | "review";
  blocking: boolean;
};

type DQFFileLike = {
  complete: boolean;
  purgeEligible: boolean;
  items: readonly { section: string; code: string; name: string; status: string }[];
  verifications: readonly (VerificationLike & { id: string; employerName: string })[];
};

const ITEM_STEP_LABELS: Record<string, string> = {
  Missing: "Add",
  Expired: "Renew",
  Outstanding: "Settle",
  ExpiringSoon: "Renew soon",
};

/**
 * The file's work, in the order the office should do it: gaps that stop the
 * file, then the employers to chase, then what merely warns. The safety
 * history line is not a step of its own — its employers are.
 */
export function dqfNextSteps(file: DQFFileLike, now: number): DQFNextStep[] {
  const steps: DQFNextStep[] = [];

  const itemStep = (item: DQFFileLike["items"][number], blocking: boolean): DQFNextStep => ({
    id: `item:${item.section}:${item.code}`,
    kind: "item",
    label: `${ITEM_STEP_LABELS[item.status] ?? "Fix"} ${item.name.toLowerCase()}`,
    detail: dqfItemStatusLabel(item.status),
    tab: dqfSectionTab(item.section),
    blocking,
  });

  for (const item of file.items) {
    if (item.section === "SafetyHistory") continue;
    if (dqfItemBlocks(item.status)) steps.push(itemStep(item, true));
  }

  const history = file.items.find((item) => item.section === "SafetyHistory");
  if (history && dqfItemBlocks(history.status) && file.verifications.length === 0) {
    steps.push({
      id: "employers",
      kind: "employers",
      label: translate("Record the driver's previous employers"),
      detail: translate("Until one is recorded the three-year investigation has not been made."),
      action: "addEmployer",
      blocking: true,
    });
  }

  for (const verification of file.verifications) {
    const next = verificationNextStep(verification, now);
    if (next.action === "done" || next.action === "wait") continue;
    steps.push({
      id: `verification:${verification.id}`,
      kind: "verification",
      label: `${next.label} — ${verification.employerName}`,
      detail: verificationStatusLabel(verification.status),
      verificationId: verification.id,
      action: next.action,
      blocking: true,
    });
  }

  for (const item of file.items) {
    if (item.status === "ExpiringSoon") steps.push(itemStep(item, false));
  }

  if (file.purgeEligible) {
    steps.push({
      id: "purge",
      kind: "purge",
      label: translate("Review the file for purge"),
      detail: translate(
        "Held past its retention window. Purging is a deliberate act, never automatic.",
      ),
      action: "review",
      blocking: false,
    });
  }

  return steps;
}
