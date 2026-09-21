import type { Tone } from "@/components/kpi/tone";
import type { AssistantProposal } from "@/types/assistant";
import { argumentRows, humanizeToolName } from "./proposal-state";

export type ProposalFact = { label: string; value: string };

/**
 * A proposal in an approver's words.
 *
 * `summary` is the whole decision in one sentence and `highlights` are the few
 * values that bear on it. There is deliberately no second tier: a disclosure
 * holding the literal payload turned out to be a wall of bookkeeping — run ids,
 * PULIDs, a raw evidence blob — that restated the sentence above it and pushed
 * the decision off the bottom of the card.
 *
 * Nothing is lost by dropping it. Each presenter declares which arguments its
 * words already account for, and every argument it did not name is appended to
 * `highlights`, so a tool that grows a field server-side still puts it in front
 * of the approver rather than going quiet.
 */
export type ProposalView = {
  title: string;
  summary: string;
  /** Shown as a badge beside the title when the tool reports one. */
  severity: { label: string; tone: Tone } | null;
  highlights: ProposalFact[];
  reversible: boolean;
};

type Args = Record<string, unknown>;

type Presenter = (args: Args) => {
  title: string;
  summary: string;
  severity?: { label: string; tone: Tone } | null;
  highlights?: ProposalFact[];
  /** Argument keys this presenter's words already account for. */
  covered: string[];
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

type DefinitionColumn = {
  ref?: { path?: unknown; field?: unknown };
  agg?: unknown;
  label?: unknown;
};

/**
 * A report definition on a card, in the words an approver can check: the
 * dataset, the columns as "sum of revenue" and a count of filters. The object
 * itself stays off the face; the columns and filters are what the person is
 * approving, and a reader can no more audit a filter tree than a PULID.
 */
function definitionFacts(definition: unknown): ProposalFact[] {
  if (typeof definition !== "object" || definition === null) {
    return [];
  }
  const shape = definition as {
    columns?: unknown;
    filters?: { filters?: unknown; groups?: unknown };
    parameters?: unknown;
  };

  const columns = Array.isArray(shape.columns)
    ? shape.columns.map((column) => columnText((column ?? {}) as DefinitionColumn))
    : [];
  const filterCount = countFilters(shape.filters);
  const parameters = Array.isArray(shape.parameters)
    ? shape.parameters.map((parameter) => text((parameter as { name?: unknown })?.name))
    : [];

  return facts(
    fact("Columns", columns.filter((column) => column !== "").join(", ")),
    filterCount > 0 ? fact("Filters", plural(filterCount, "filter", "filters")) : null,
    fact("Parameters", parameters.filter((name) => name !== "").join(", ")),
  );
}

function columnText(column: DefinitionColumn): string {
  const path = Array.isArray(column.ref?.path) ? column.ref.path.map(text) : [];
  const field = [...path, text(column.ref?.field)].filter((part) => part !== "").join(".");
  const label = text(column.label);
  const agg = text(column.agg);

  if (field === "") {
    return label;
  }
  if (agg === "") {
    return field;
  }
  return `${agg.replaceAll("_", " ")} of ${field}`;
}

function countFilters(group: { filters?: unknown; groups?: unknown } | undefined): number {
  if (typeof group !== "object" || group === null) {
    return 0;
  }
  const own = Array.isArray(group.filters) ? group.filters.length : 0;
  const nested = Array.isArray(group.groups)
    ? group.groups.reduce(
        (sum: number, child) => sum + countFilters(child as { filters?: unknown }),
        0,
      )
    : 0;

  return own + nested;
}

const REPORT_METADATA_KEYS = [
  "name",
  "description",
  "category",
  "tags",
  "visibility",
  "defaultFormat",
  "status",
] as const;

const REPORT_METADATA_LABELS: Record<(typeof REPORT_METADATA_KEYS)[number], string> = {
  name: "Name",
  description: "Description",
  category: "Category",
  tags: "Tags",
  visibility: "Visibility",
  defaultFormat: "Format",
  status: "Status",
};

/** "needs_attention" reads as "Needs attention"; a file format reads as its acronym. */
function reportMetadataValue(key: (typeof REPORT_METADATA_KEYS)[number], value: unknown): string {
  const raw = text(value);
  switch (key) {
    case "tags":
      return list(value).join(", ");
    case "defaultFormat":
      return raw.toUpperCase();
    case "visibility":
    case "status": {
      const words = raw.replaceAll("_", " ").toLowerCase();
      return words === "" ? "" : words[0].toUpperCase() + words.slice(1);
    }
    default:
      return raw;
  }
}

const PRESENTERS: Record<string, Presenter> = {
  create_report: (args) => {
    const name = text(args.name);
    const dataset = text((args.definition as { entity?: unknown } | undefined)?.entity);
    const visibility = text(args.visibility).toLowerCase();

    return {
      title: "Create a report",
      summary: `Save ${name ? `“${name}”` : "a new report"} as a new ${
        visibility === "shared" ? "shared " : ""
      }report${dataset ? ` on the ${dataset} dataset` : ""}.`,
      highlights: facts(
        ...definitionFacts(args.definition),
        fact("Category", text(args.category)),
        fact("Tags", list(args.tags).join(", ")),
        fact("Format", reportMetadataValue("defaultFormat", args.defaultFormat)),
        fact("Description", text(args.description)),
      ),
      covered: [...REPORT_METADATA_KEYS, "definition"],
      reversible: true,
    };
  },

  update_report: (args) => {
    const changed = REPORT_METADATA_KEYS.filter((key) => args[key] !== undefined).map((key) =>
      REPORT_METADATA_LABELS[key].toLowerCase(),
    );
    if (args.definition !== undefined) {
      changed.push("definition");
    }
    const what =
      changed.length === 0
        ? "settings"
        : changed.length === 1
          ? changed[0]
          : `${changed.slice(0, -1).join(", ")} and ${changed[changed.length - 1]}`;

    return {
      title: "Change a report",
      summary: `Change this report's ${what}.`,
      highlights: facts(
        ...REPORT_METADATA_KEYS.map((key) =>
          args[key] === undefined
            ? null
            : fact(REPORT_METADATA_LABELS[key], reportMetadataValue(key, args[key])),
        ),
        ...definitionFacts(args.definition),
      ),
      covered: ["definitionId", ...REPORT_METADATA_KEYS, "definition"],
      reversible: true,
    };
  },

  evaluate_service_failures: (args) => ({
    title: "Check for service failures",
    summary: `Run the late-stop check on this shipment${
      args.force === true ? ", re-checking stops already evaluated" : ""
    }.`,
    highlights: [],
    covered: ["shipmentId", "force"],
    reversible: true,
  }),

  resolve_service_failure: (args) => ({
    title: "Resolve a service failure",
    summary: "Close this service failure with the reason and note below.",
    highlights: facts(
      fact("Reason code", shortRef(text(args.reasonCodeId))),
      fact("Note", text(args.notes)),
    ),
    covered: ["serviceFailureId", "reasonCodeId", "notes"],
    reversible: false,
  }),

  notify_driver: (args) => {
    const priority = text(args.priority);

    return {
      title: "Message the driver",
      summary: `Send driver ${shortRef(text(args.workerId)) || "?"} “${text(args.title)}” in Dash.`,
      severity:
        priority === "critical" || priority === "high"
          ? { label: humanizeEnum(priority[0].toUpperCase() + priority.slice(1)), tone: "warning" }
          : null,
      highlights: facts(fact("Message", text(args.message))),
      covered: ["workerId", "title", "message", "priority", "shipmentId"],
      reversible: false,
    };
  },

  email_customer: (args) => ({
    title: "Email the customer",
    summary: `Send this shipment's customer “${text(args.subject)}” from the organization's letterhead.`,
    highlights: facts(fact("Message", text(args.body))),
    covered: ["shipmentId", "profileId", "subject", "body"],
    reversible: false,
  }),

  send_detention_notice: () => ({
    title: "Send a detention notice",
    summary: "Send the customer the detention notice for this occurrence.",
    highlights: [],
    covered: ["occurrenceId"],
    reversible: false,
  }),

  waive_detention: (args) => ({
    title: "Waive detention",
    summary: `Waive this detention charge as ${midSentence(humanizeEnum(text(args.reason)))}.`,
    highlights: facts(fact("Note", text(args.note))),
    covered: ["occurrenceId", "reason", "note"],
    reversible: false,
  }),

  fork_report: (args) => {
    const key = text(args.reportKey);
    const name = text(args.name);

    return {
      title: "Copy a report",
      summary: `Make a copy of the built-in ${key || "report"} report${
        name ? ` named “${name}”` : ""
      }.`,
      highlights: [],
      covered: ["reportKey", "name"],
      reversible: true,
    };
  },

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
      covered: [
        "to",
        "requestedDocuments",
        "customerName",
        "shipmentProNumber",
        "subject",
        "body",
        "profileId",
      ],
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
      covered: ["billingQueueItemId", "additionalCharges"],
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
      covered: ["shipmentMoveId", "primaryWorkerId", "secondaryWorkerId", "tractorId", "trailerId"],
      reversible: true,
    };
  },

  transition_item_to_in_review: (args) => ({
    title: "Send to review",
    summary: "Move this billing item into review so a biller picks it up.",
    highlights: facts(fact("Reason", text(args.reason))),
    covered: ["billingQueueItemId", "reason"],
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
      // The run id, the subject's key and the evidence blob belong to the audit
      // trail. Severity and category are already the badge and the sentence.
      covered: [
        "runId",
        "subjectId",
        "category",
        "severity",
        "attemptSummary",
        "blastRadius",
        "evidence",
      ],
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
      covered: [
        "shipmentId",
        "documentTypeId",
        "fileName",
        "contentType",
        "fileSize",
        "description",
      ],
      reversible: false,
    };
  },
};

export function presentProposal(proposal: AssistantProposal): ProposalView {
  const args = (proposal.arguments ?? {}) as Args;
  const rows = argumentRows(args).map((row) => ({
    label: humanizeEnum(row.key),
    value: row.value,
    key: row.key,
  }));

  const presenter = PRESENTERS[proposal.toolName];
  if (!presenter) {
    return {
      title: humanizeToolName(proposal.toolName),
      summary: `Run ${humanizeToolName(proposal.toolName).toLowerCase()} with the values below.`,
      severity: null,
      // Nothing is known about an unrecognised tool, so nothing is assumed to
      // be noise: every argument it would send is on the face of the card.
      highlights: rows.map(({ label, value }) => ({ label, value })),
      reversible: false,
    };
  }

  const view = presenter(args);
  const covered = new Set(view.covered);
  const uncovered = rows
    .filter((row) => !covered.has(row.key))
    .map(({ label, value }) => ({ label, value }));

  return {
    title: view.title,
    summary: view.summary,
    severity: view.severity ?? null,
    highlights: [...(view.highlights ?? []).filter((entry) => entry.value !== ""), ...uncovered],
    reversible: view.reversible,
  };
}
