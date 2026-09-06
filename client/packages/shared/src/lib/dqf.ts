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
