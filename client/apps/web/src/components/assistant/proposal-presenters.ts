import type { AssistantProposal } from "@/types/assistant";
import { argumentRows, humanizeToolName } from "./proposal-state";

export type ProposalFact = { label: string; value: string };

/**
 * A proposal in an approver's words: what would happen, the fields that
 * decide it, and whether it can be undone. Every argument the presenter did
 * not name is still listed, so nothing that would run is hidden.
 */
export type ProposalView = {
  title: string;
  summary: string;
  facts: ProposalFact[];
  longText: ProposalFact | null;
  reversible: boolean;
};

type Args = Record<string, unknown>;

type Presenter = (args: Args) => Omit<ProposalView, "facts"> & {
  facts: ProposalFact[];
  /** Argument keys the presenter already covered. */
  covered: string[];
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

const PRESENTERS: Record<string, Presenter> = {
  request_missing_docs: (args) => {
    const to = list(args.to);
    const documents = list(args.requestedDocuments);
    const customer = text(args.customerName);
    const pro = text(args.shipmentProNumber);
    const who = customer || to.join(", ") || "the customer";
    const about = pro ? ` about shipment ${pro}` : "";

    return {
      title: "Request missing documents",
      summary: `Email ${who}${about} asking for ${plural(documents.length, "document", "documents")}.`,
      facts: facts(
        fact("To", to.join(", ")),
        fact("Subject", text(args.subject)),
        fact("Documents", documents.join(", ")),
      ),
      longText: fact("Message", text(args.body)),
      reversible: false,
      covered: [
        "to",
        "requestedDocuments",
        "customerName",
        "shipmentProNumber",
        "subject",
        "body",
        "profileId",
      ],
    };
  },

  correct_charge_code: (args) => {
    const item = text(args.billingQueueItemId);
    const charges = Array.isArray(args.additionalCharges) ? args.additionalCharges : [];
    const chargeFacts = charges.map((charge, index) => {
      const entry = (charge ?? {}) as Args;
      const name =
        text(entry.description) || text(entry.accessorialChargeId) || `Charge ${index + 1}`;
      const amount = text(entry.amount);
      return { label: `Charge ${index + 1}`, value: amount ? `${name} · ${amount}` : name };
    });

    return {
      title: "Correct charge codes",
      summary: `Replace the additional charges on billing item ${item || "?"} with ${plural(charges.length, "charge", "charges")}.`,
      facts: facts(fact("Billing item", item), ...chargeFacts),
      longText: null,
      reversible: true,
      covered: ["billingQueueItemId", "additionalCharges"],
    };
  },

  assign_move: (args) => {
    const move = text(args.shipmentMoveId);
    const driver = text(args.primaryWorkerId);
    const second = text(args.secondaryWorkerId);
    const tractor = text(args.tractorId);
    const trailer = text(args.trailerId);
    const equipment = [tractor ? `tractor ${tractor}` : "", trailer ? `trailer ${trailer}` : ""]
      .filter(Boolean)
      .join(" and ");

    return {
      title: "Assign move",
      summary: `Put driver ${driver || "?"} on move ${move || "?"}${equipment ? ` with ${equipment}` : ""}.`,
      facts: facts(
        fact("Move", move),
        fact("Driver", driver),
        fact("Second driver", second),
        fact("Tractor", tractor),
        fact("Trailer", trailer),
      ),
      longText: null,
      reversible: true,
      covered: ["shipmentMoveId", "primaryWorkerId", "secondaryWorkerId", "tractorId", "trailerId"],
    };
  },

  transition_item_to_in_review: (args) => {
    const item = text(args.billingQueueItemId);
    return {
      title: "Move billing item to review",
      summary: `Mark billing item ${item || "?"} as in review so a person picks it up.`,
      facts: facts(fact("Billing item", item), fact("Reason", text(args.reason))),
      longText: null,
      reversible: true,
      covered: ["billingQueueItemId", "reason"],
    };
  },

  flag_for_manual_review: (args) => {
    const category = text(args.category);
    const severity = text(args.severity);
    return {
      title: "Flag for manual review",
      summary: `Raise a ${severity ? severity.toLowerCase() + " " : ""}${category ? category.replace(/([a-z])([A-Z])/g, "$1 $2").toLowerCase() : "case"} for a person to handle.`,
      facts: facts(
        fact("Subject", text(args.subjectId)),
        fact("Category", category),
        fact("Severity", severity),
        fact("Records affected", text(args.blastRadius)),
      ),
      longText: fact("What the agent tried", text(args.attemptSummary)),
      reversible: false,
      covered: [
        "runId",
        "subjectId",
        "category",
        "severity",
        "attemptSummary",
        "blastRadius",
        "evidence",
      ],
    };
  },

  attach_document_to_bqi: (args) => {
    const file = text(args.fileName);
    return {
      title: "Attach document to billing item",
      summary: `Attach ${file || "a document"} to the billing item for shipment ${text(args.shipmentId) || "?"}.`,
      facts: facts(
        fact("Shipment", text(args.shipmentId)),
        fact("File", file),
        fact("Type", text(args.contentType)),
        fact("Document type", text(args.documentTypeId)),
      ),
      longText: fact("Description", text(args.description)),
      reversible: false,
      covered: [
        "shipmentId",
        "documentTypeId",
        "fileName",
        "contentType",
        "fileSize",
        "description",
      ],
    };
  },
};

export function presentProposal(proposal: AssistantProposal): ProposalView {
  const args = (proposal.arguments ?? {}) as Args;
  const presenter = PRESENTERS[proposal.toolName];

  if (!presenter) {
    return {
      title: humanizeToolName(proposal.toolName),
      summary: `Run ${proposal.toolName} with the parameters below.`,
      facts: argumentRows(args).map((row) => ({ label: row.key, value: row.value })),
      longText: null,
      reversible: false,
    };
  }

  const view = presenter(args);
  const covered = new Set(view.covered);
  const extra = argumentRows(args)
    .filter((row) => !covered.has(row.key))
    .map((row) => ({ label: row.key, value: row.value }));

  return {
    title: view.title,
    summary: view.summary,
    facts: [...view.facts, ...extra],
    longText: view.longText,
    reversible: view.reversible,
  };
}
