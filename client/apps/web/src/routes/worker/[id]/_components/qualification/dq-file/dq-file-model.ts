import type { DQFFile, DQFItem, EmploymentVerificationRow } from "@/lib/graphql/worker-dqf";
import type { WorkerTarget } from "@/routes/worker/use-worker-state";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import {
  DQF_SECTION_ORDER,
  dqfItemBlocks,
  verificationMethodLabel,
  verificationNextStep,
  verificationSettled,
  type DQFSectionValue,
} from "@trenova/shared/lib/dqf";
import { toSentenceFragment } from "@trenova/shared/lib/utils";

export type SegmentTone = "success" | "warning" | "danger";

export type SpineBlock = {
  section: DQFSectionValue;
  /** One segment per row of the section's panel, in its order. */
  segments: SegmentTone[];
  /** Rows not short of what the file needs: on file, or still in date while expiring. */
  onFile: number;
  outstanding: number;
  warning: number;
};

export function itemTone(status: string): SegmentTone | null {
  if (status === "NotApplicable") return null;
  if (status === "Satisfied") return "success";
  if (status === "ExpiringSoon") return "warning";
  return dqfItemBlocks(status) ? "danger" : "warning";
}

/**
 * Where one previous employer stands for the spine and its row: settled is on file, a request
 * past the investigation deadline is outstanding, anything else is waiting.
 */
export function employerTone(
  verification: Pick<EmploymentVerificationRow, "status">,
  late: boolean,
): SegmentTone {
  if (verificationSettled(verification.status)) return "success";
  return late ? "danger" : "warning";
}

/**
 * The file's spine, read from the same rows its panels list: each section's items, and for the
 * employers section each previous employer (the one safety-history item when none is recorded).
 */
export function dqSpine(
  file: Pick<DQFFile, "items" | "verifications" | "safetyHistoryLate">,
): SpineBlock[] {
  return DQF_SECTION_ORDER.map((section) => {
    let segments: SegmentTone[];
    if (section === "SafetyHistory" && file.verifications.length > 0) {
      segments = file.verifications.map((row) => employerTone(row, file.safetyHistoryLate));
    } else {
      segments = file.items
        .filter((item) => item.section === section)
        .map((item) => itemTone(item.status))
        .filter((tone): tone is SegmentTone => tone !== null);
    }
    const outstanding = segments.filter((tone) => tone === "danger").length;
    return {
      section,
      segments,
      onFile: segments.length - outstanding,
      outstanding,
      warning: segments.filter((tone) => tone === "warning").length,
    };
  });
}

export type EmployerStopKey = "recorded" | "requested" | "chased" | "answered";

export type EmployerStop = {
  key: EmployerStopKey;
  state: "done" | "current" | "ahead";
  danger: boolean;
  meta: string | null;
};

/**
 * One previous employer's request as four stops: not requested, requested, chased, answered,
 * with the dates that prove each. A silent employer closed as no response reaches the end too:
 * the chases on record are the answer the rule asks for.
 */
export function employerStops(
  t: TranslateFn,
  verification: EmploymentVerificationRow,
  deadline: { dueAt: number; late: boolean },
  now: number,
): EmployerStop[] {
  const requestedMeta = verification.requestedAt
    ? t(
        "{0} · {1}",
        formatUnixDateMedium(verification.requestedAt),
        verificationMethodLabel(verification.method),
      )
    : null;
  const chases = verification.followUpCount;
  const chasedMeta =
    chases > 0
      ? verification.lastFollowUpAt
        ? t(
            "{0, plural, one {# chase} other {# chases}} · last {1}",
            chases,
            formatUnixDateMedium(verification.lastFollowUpAt),
          )
        : t("{0, plural, one {# chase} other {# chases}}", chases)
      : null;
  const due =
    deadline.dueAt > 0
      ? deadline.late
        ? t("Was due {0}", formatUnixDateMedium(deadline.dueAt))
        : t("Due {0}", formatUnixDateMedium(deadline.dueAt))
      : null;
  const stop = (
    key: EmployerStopKey,
    state: EmployerStop["state"],
    meta: string | null,
    danger = false,
  ): EmployerStop => ({ key, state, danger, meta });

  switch (verification.status) {
    case "Pending":
      return [
        stop("recorded", "current", t("Not sent yet"), deadline.late),
        stop("requested", "ahead", null),
        stop("chased", "ahead", null),
        stop("answered", "ahead", due),
      ];
    case "Requested": {
      const next = verificationNextStep(verification, now);
      if (chases === 0) {
        return [
          stop("recorded", "done", null),
          stop("requested", "current", requestedMeta, deadline.late),
          stop(
            "chased",
            "ahead",
            next.action === "wait" && next.dueAt
              ? t("From {0}", formatUnixDateMedium(next.dueAt))
              : null,
          ),
          stop("answered", "ahead", due),
        ];
      }
      return [
        stop("recorded", "done", null),
        stop("requested", "done", requestedMeta),
        stop("chased", "current", chasedMeta, deadline.late),
        stop("answered", "ahead", due),
      ];
    }
    case "Received":
      return [
        stop("recorded", "done", null),
        stop("requested", "done", requestedMeta),
        stop("chased", "done", chasedMeta ?? t("Not needed")),
        stop(
          "answered",
          "done",
          verification.responseReceivedAt
            ? formatUnixDateMedium(verification.responseReceivedAt)
            : null,
        ),
      ];
    case "NoResponse":
      return [
        stop("recorded", "done", null),
        stop("requested", "done", requestedMeta),
        stop("chased", "done", chasedMeta),
        stop("answered", "done", t("Closed without an answer")),
      ];
    default:
      return [
        stop("recorded", "done", null),
        stop("requested", "done", t("Not needed")),
        stop("chased", "done", t("Not needed")),
        stop("answered", "done", t("Not applicable")),
      ];
  }
}

/** A credential the file names, found by its type's code in the credential summary. */
export type FileCredential = {
  typeId: string;
  credentialId: string | null;
  name: string;
  required: boolean;
  verified: boolean;
};

export type StepAction =
  | { kind: "go"; label: string; target: WorkerTarget }
  | { kind: "request" | "chase"; label: string; verificationId: string };

export type DQStep = { id: string; text: string; action: StepAction | null };

export type StepAbilities = {
  createCredential: boolean;
  createEmployer: boolean;
  updateEmployer: boolean;
};

const CREDENTIALS_VIEW = { tab: "qualification", view: "credentials" } as const;

function credentialStep(
  t: TranslateFn,
  item: DQFItem,
  credential: FileCredential | undefined,
  can: StepAbilities,
): DQStep {
  const name = toSentenceFragment(item.name);
  const id = `item:${item.section}:${item.code}`;
  const open: StepAction = credential?.credentialId
    ? {
        kind: "go",
        label: t("Open credential"),
        target: { ...CREDENTIALS_VIEW, credential: credential.credentialId },
      }
    : { kind: "go", label: t("Open credentials"), target: CREDENTIALS_VIEW };
  const renew: StepAction =
    credential?.credentialId && can.createCredential
      ? {
          kind: "go",
          label: t("Renew"),
          target: {
            ...CREDENTIALS_VIEW,
            form: "renew-credential",
            credential: credential.credentialId,
          },
        }
      : open;
  switch (item.status) {
    case "Missing":
      return {
        id,
        text: t("Add the {0}", name),
        action:
          credential && can.createCredential
            ? {
                kind: "go",
                label: t("Add credential"),
                target: { ...CREDENTIALS_VIEW, form: "add-credential", kind: credential.typeId },
              }
            : open,
      };
    case "Expired":
      return { id, text: t("Renew the expired {0}", name), action: renew };
    case "ExpiringSoon":
      return {
        id,
        text: item.expiresAt
          ? t("Renew the {0} before {1}", name, formatUnixDateMedium(item.expiresAt))
          : t("Renew the {0} soon", name),
        action: renew,
      };
    default:
      return { id, text: t("Settle the {0}", name), action: open };
  }
}

function documentStep(t: TranslateFn, item: DQFItem): DQStep {
  const name = toSentenceFragment(item.name);
  const text =
    item.status === "Missing"
      ? t("Upload the {0}", name)
      : item.status === "Expired"
        ? t("Replace the expired {0}", name)
        : item.status === "ExpiringSoon"
          ? t("Replace the {0} before it expires", name)
          : t("Review the {0}", name);
  return {
    id: `item:${item.section}:${item.code}`,
    text,
    action: {
      kind: "go",
      label: t("Open documents"),
      target: { tab: "files", view: "documents" },
    },
  };
}

function drugAlcoholStep(t: TranslateFn, item: DQFItem): DQStep {
  const name = toSentenceFragment(item.name);
  return {
    id: `item:${item.section}:${item.code}`,
    text: item.status === "Missing" ? t("Record the {0}", name) : t("Settle the {0}", name),
    action: {
      kind: "go",
      label: t("Open drug and alcohol"),
      target: { tab: "qualification", view: "drug-and-alcohol" },
    },
  };
}

function itemStep(
  t: TranslateFn,
  item: DQFItem,
  credentials: ReadonlyMap<string, FileCredential>,
  can: StepAbilities,
): DQStep {
  switch (item.section) {
    case "Credentials":
      return credentialStep(t, item, credentials.get(item.code), can);
    case "Documents":
      return documentStep(t, item);
    default:
      return drugAlcoholStep(t, item);
  }
}

/**
 * The file's work, in the order the office should do it: gaps that stop the file, the employers
 * to record, send, chase or close, credentials nobody has verified, then what only warns. Each
 * step carries the one action that does it, or takes the reader to where it is done.
 */
export function dqNextSteps(
  t: TranslateFn,
  input: {
    file: Pick<DQFFile, "items" | "verifications" | "purgeEligible">;
    credentials: ReadonlyMap<string, FileCredential>;
    now: number;
    can: StepAbilities;
  },
): DQStep[] {
  const { file, credentials, now, can } = input;
  const steps: DQStep[] = [];

  for (const item of file.items) {
    if (item.section === "SafetyHistory") continue;
    if (dqfItemBlocks(item.status)) steps.push(itemStep(t, item, credentials, can));
  }

  const history = file.items.find((item) => item.section === "SafetyHistory");
  if (history && dqfItemBlocks(history.status) && file.verifications.length === 0) {
    steps.push({
      id: "employers",
      text: t("Record the previous employers from the three years before hire"),
      action: can.createEmployer
        ? {
            kind: "go",
            label: t("Add employer"),
            target: { tab: "qualification", view: "dq-file", form: "add-employer" },
          }
        : null,
    });
  }

  for (const verification of file.verifications) {
    const next = verificationNextStep(verification, now);
    const id = `verification:${verification.id}`;
    const name = verification.employerName;
    const edit: StepAction | null = can.updateEmployer
      ? {
          kind: "go",
          label: t("Edit employer"),
          target: {
            tab: "qualification",
            view: "dq-file",
            form: "edit-employer",
            employer: verification.id,
          },
        }
      : null;
    switch (next.action) {
      case "request":
        steps.push({
          id,
          text: t("Send the safety history request to {0}", name),
          action: can.updateEmployer
            ? { kind: "request", label: t("Send the request"), verificationId: verification.id }
            : null,
        });
        break;
      case "chase":
        steps.push({
          id,
          text: t("Chase {0} again, or record a good-faith effort and close it out", name),
          action: can.updateEmployer
            ? { kind: "chase", label: t("Chase again"), verificationId: verification.id }
            : null,
        });
        break;
      case "close":
        steps.push({
          id,
          text: t("Close {0} as no response; the chases on record are the good-faith effort", name),
          action: edit && { ...edit, label: t("Close it out") },
        });
        break;
      case "drugAlcohol":
        steps.push({
          id,
          text: t("Record the drug and alcohol history {0} sent", name),
          action: edit,
        });
        break;
      default:
        break;
    }
  }

  for (const credential of credentials.values()) {
    if (!credential.required || !credential.credentialId || credential.verified) continue;
    steps.push({
      id: `verify:${credential.credentialId}`,
      text: t("Verify the {0}", toSentenceFragment(credential.name)),
      action: {
        kind: "go",
        label: t("Open credential"),
        target: { ...CREDENTIALS_VIEW, credential: credential.credentialId },
      },
    });
  }

  for (const item of file.items) {
    if (item.status === "ExpiringSoon") steps.push(itemStep(t, item, credentials, can));
  }

  if (file.purgeEligible) {
    steps.push({
      id: "purge",
      text: t("Review the file for purge; it is held past its retention window"),
      action: null,
    });
  }

  return steps;
}
