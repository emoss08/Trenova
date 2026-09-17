import type { Tone } from "@/components/kpi/tone";
import type { AssistantProposal } from "@/types/assistant";
import { argumentRows, humanizeToolName } from "./proposal-state";

export type ProposalFact = { label: string; value: string };

/**
 * A proposal in an approver's words.
 *
 * `summary` is the whole decision in one sentence; everything else is support.
 * `details` carries every argument that would be sent, including the ones the
 * sentence already named, so the card can hide them behind a disclosure without
 * hiding anything: an approver who wants the literal payload can always get it,
 * and one who does not is never made to read an identifier to answer yes or no.
 */
export type ProposalView = {
  title: string;
  summary: string;
  /** Shown as a badge beside the title when the tool reports one. */
  severity: { label: string; tone: Tone } | null;
  /** The one or two values worth reading without expanding. */
  highlights: ProposalFact[];
  details: ProposalFact[];
  reversible: boolean;
};

type Args = Record<string, unknown>;

type Presenter = (args: Args) => {
  title: string;
  summary: string;
  severity?: { label: string; tone: Tone } | null;
  highlights?: ProposalFact[];
  reversible: boolean;
};

function text(value: unknown): string {
  if (value === null || value === undefined) return "";
  if (typeof value === "string") return value.trim();
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  return "";
}

function list(value: unknown): string[] {
  return Array.isArray(value) ? value.map(text).filter((entry) => entry !== "") : [];
}

function fact(label: string, value: string): ProposalFact | null {
  return value === "" ? null : { label, value };
}

function facts(...entries: (ProposalFact | null)[]): ProposalFact[] {
  return entries.filter((entry): entry is ProposalFact => entry !== null);
}

function plural(count: number, singular: string, pluralWord: string): string {
  return `${count} ${count === 1 ? singular : pluralWord}`;
}

/**
 * A PULID is a storage key, not a name. Putting one in a sentence asks a person
 * to match 26 characters by eye to decide whether a change is the one they
 * meant, so the reference is shortened to its tail, which is what distinguishes
 * two ids anyway, and the full value stays in the details.
 */
const PULID = /^[a-z]{2,8}_[0-9A-HJKMNP-TV-Z]{26}$/;

export function isOpaqueId(value: string): boolean {
  return PULID.test(value.trim());
}

export function shortRef(value: string): string {
  const trimmed = value.trim();

  return isOpaqueId(trimmed) ? `…${trimmed.slice(-6)}` : trimmed;
}

/** Splits `MissingBOL` into `Missing BOL` without breaking the initialism. */
function humanizeEnum(value: string): string {
  return value
    .replace(/([a-z0-9])([A-Z])/gu, "$1 $2")
    .replace(/([A-Z]+)([A-Z][a-z])/gu, "$1 $2")
    .trim();
}

/**
 * Lowercases a humanized enum for use mid-sentence without flattening an
 * initialism: "MissingBOL" reads as "missing BOL", never "missing bol".
 */
function midSentence(value: string): string {
  return value
    .split(" ")
    .map((word) => (/^[A-Z0-9]{2,}$/u.test(word) ? word : word.toLowerCase()))
    .join(" ");
}

function severityOf(value: string): { label: string; tone: Tone } | null {
  switch (value.trim().toLowerCase()) {
    case "critical":
    case "high":
      return { label: humanizeEnum(value), tone: "danger" };
    case "medium":
      return { label: humanizeEnum(value), tone: "warning" };
    case "low":
      return { label: humanizeEnum(value), tone: "muted" };
    default:
      return null;
  }
}

const PRESENTERS: Record<string, Presenter> = {
  request_missing_docs: (args) => {
    const to = list(args.to);
    const documents = list(args.requestedDocuments);
    const who = text(args.customerName) || to.join(", ") || "the customer";
    const pro = text(args.shipmentProNumber);

    return {
      title: "Email the customer",
      summary: `Ask ${who} for ${plural(documents.length, "document", "documents")}${
        pro ? ` on shipment ${pro}` : ""
      }.`,
      highlights: facts(
        fact("To", to.join(", ")),
        fact("Asking for", documents.join(", ")),
        fact("Subject", text(args.subject)),
      ),
      reversible: false,
    };
  },

  correct_charge_code: (args) => {
    const charges = Array.isArray(args.additionalCharges) ? args.additionalCharges : [];

    return {
      title: "Correct charge codes",
      summary: `Replace the additional charges on this billing item with ${plural(
        charges.length,
        "charge",
        "charges",
      )}.`,
      highlights: charges.slice(0, 4).map((charge, index) => {
        const entry = (charge ?? {}) as Args;
        const name =
          text(entry.description) ||
          shortRef(text(entry.accessorialChargeId)) ||
          `Charge ${index + 1}`;
        const amount = text(entry.amount);

        return { label: name, value: amount === "" ? "—" : amount };
      }),
      reversible: true,
    };
  },

  assign_move: (args) => {
    const driver = shortRef(text(args.primaryWorkerId));
    const tractor = shortRef(text(args.tractorId));
    const trailer = shortRef(text(args.trailerId));

    return {
      title: "Assign a driver",
      summary: `Put driver ${driver || "?"} on this move${
        tractor ? ` with tractor ${tractor}` : ""
      }.`,
      highlights: facts(
        fact("Driver", driver),
        fact("Second driver", shortRef(text(args.secondaryWorkerId))),
        fact("Tractor", tractor),
        fact("Trailer", trailer),
      ),
      reversible: true,
    };
  },

  transition_item_to_in_review: (args) => ({
    title: "Send to review",
    summary: "Move this billing item into review so a biller picks it up.",
    highlights: facts(fact("Reason", text(args.reason))),
    reversible: true,
  }),

  flag_for_manual_review: (args) => {
    const category = humanizeEnum(text(args.category));
    const affected = Number(args.blastRadius ?? 0);

    return {
      title: "Flag for review",
      summary: category
        ? `Record a ${midSentence(category)} case for a person to resolve.`
        : "Record a case for a person to resolve.",
      severity: severityOf(text(args.severity)),
      highlights: facts(
        fact("What the agent found", text(args.attemptSummary)),
        affected > 1 ? fact("Records affected", String(affected)) : null,
      ),
      reversible: false,
    };
  },

  attach_document_to_bqi: (args) => {
    const file = text(args.fileName);

    return {
      title: "Attach a document",
      summary: `Attach ${file || "a document"} to this shipment's billing item.`,
      highlights: facts(
        fact("File", file),
        fact("Type", text(args.contentType)),
        fact("Note", text(args.description)),
      ),
      reversible: false,
    };
  },
};

export function presentProposal(proposal: AssistantProposal): ProposalView {
  const args = (proposal.arguments ?? {}) as Args;
  const details = argumentRows(args).map((row) => ({
    label: humanizeEnum(row.key),
    value: row.value,
  }));

  const presenter = PRESENTERS[proposal.toolName];
  if (!presenter) {
    return {
      title: humanizeToolName(proposal.toolName),
      summary: `Run ${humanizeToolName(proposal.toolName).toLowerCase()} with the values below.`,
      severity: null,
      // Nothing is known about an unrecognised tool, so nothing is hidden:
      // every argument is on the face of the card rather than behind a click.
      highlights: details,
      details: [],
      reversible: false,
    };
  }

  const view = presenter(args);

  return {
    title: view.title,
    summary: view.summary,
    severity: view.severity ?? null,
    highlights: (view.highlights ?? []).filter((entry) => entry.value !== ""),
    details,
    reversible: view.reversible,
  };
}
