import type { Tone } from "@/components/kpi/tone";
import type { AssistantProposal } from "@/types/assistant";
import { formatList } from "@trenova/shared/i18n/format";
import { translate } from "@trenova/shared/i18n/runtime";
import { argumentRows, humanizeToolName } from "./proposal-state";
import { defineLabels, translateLabel } from "@trenova/shared/i18n/labels";

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
  return translateLabel(
    value
      .replace(/([a-z0-9])([A-Z])/gu, "$1 $2")
      .replace(/([A-Z]+)([A-Z][a-z])/gu, "$1 $2")
      .trim(),
  );
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
    filterCount > 0
      ? fact("Filters", translate("{0, plural, one {# filter} other {# filters}}", filterCount))
      : null,
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
  switch (agg) {
    case "count":
      return translate("count of {0}", field);
    case "count_distinct":
      return translate("count distinct of {0}", field);
    case "sum":
      return translate("sum of {0}", field);
    case "avg":
      return translate("avg of {0}", field);
    case "min":
      return translate("min of {0}", field);
    case "max":
      return translate("max of {0}", field);
    default:
      return translate("{0} ({1})", field, agg);
  }
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

const REPORT_METADATA_LABELS: Record<(typeof REPORT_METADATA_KEYS)[number], string> = defineLabels({
  name: "Name",
  description: "Description",
  category: "Category",
  tags: "Tags",
  visibility: "Visibility",
  defaultFormat: "Format",
  status: "Status",
});

function reportMetadataNoun(key: (typeof REPORT_METADATA_KEYS)[number]): string {
  switch (key) {
    case "name":
      return translate("name");
    case "description":
      return translate("description");
    case "category":
      return translate("category");
    case "tags":
      return translate("tags");
    case "visibility":
      return translate("visibility");
    case "defaultFormat":
      return translate("format");
    case "status":
      return translate("status");
  }
}

function createReportSummary(name: string, shared: boolean, dataset: string): string {
  if (name) {
    if (shared) {
      return dataset
        ? translate("Save “{0}” as a new shared report on the {1} dataset.", name, dataset)
        : translate("Save “{0}” as a new shared report.", name);
    }
    return dataset
      ? translate("Save “{0}” as a new report on the {1} dataset.", name, dataset)
      : translate("Save “{0}” as a new report.", name);
  }
  if (shared) {
    return dataset
      ? translate("Save a new shared report on the {0} dataset.", dataset)
      : translate("Save a new shared report.");
  }
  return dataset
    ? translate("Save a new report on the {0} dataset.", dataset)
    : translate("Save a new report.");
}

function createLocationSummary(name: string, place: string): string {
  if (name) {
    return place
      ? translate(
          "Add “{0}” to your locations in {1}, so a stop can be booked there. Its code is assigned when it is saved.",
          name,
          place,
        )
      : translate(
          "Add “{0}” to your locations, so a stop can be booked there. Its code is assigned when it is saved.",
          name,
        );
  }
  return place
    ? translate(
        "Add a new location to your locations in {0}, so a stop can be booked there. Its code is assigned when it is saved.",
        place,
      )
    : translate(
        "Add a new location to your locations, so a stop can be booked there. Its code is assigned when it is saved.",
      );
}

function forkReportSummary(key: string, name: string): string {
  if (key) {
    return name
      ? translate("Make a copy of the built-in {0} report named “{1}”.", key, name)
      : translate("Make a copy of the built-in {0} report.", key);
  }
  return name
    ? translate("Make a copy of the built-in report named “{0}”.", name)
    : translate("Make a copy of the built-in report.");
}

function requestDocsSummary(who: string, count: number, pro: string): string {
  if (who) {
    return pro
      ? translate(
          "Ask {1} for {0, plural, one {# document} other {# documents}} on shipment {2}.",
          count,
          who,
          pro,
        )
      : translate("Ask {1} for {0, plural, one {# document} other {# documents}}.", count, who);
  }
  return pro
    ? translate(
        "Ask the customer for {0, plural, one {# document} other {# documents}} on shipment {1}.",
        count,
        pro,
      )
    : translate("Ask the customer for {0, plural, one {# document} other {# documents}}.", count);
}

function createShipmentSummary(bol: string, stops: number, fromDocument: boolean): string {
  if (bol) {
    return fromDocument
      ? translate(
          "Enter a new shipment for BOL {1} with {0, plural, one {# stop} other {# stops}}, read from an uploaded document.",
          stops,
          bol,
        )
      : translate(
          "Enter a new shipment for BOL {1} with {0, plural, one {# stop} other {# stops}}.",
          stops,
          bol,
        );
  }
  return fromDocument
    ? translate(
        "Enter a new shipment with {0, plural, one {# stop} other {# stops}}, read from an uploaded document.",
        stops,
      )
    : translate("Enter a new shipment with {0, plural, one {# stop} other {# stops}}.", stops);
}

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
      summary: createReportSummary(name, visibility === "shared", dataset),
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
    const changed = REPORT_METADATA_KEYS.filter((key) => args[key] !== undefined).map(
      reportMetadataNoun,
    );
    if (args.definition !== undefined) {
      changed.push(translate("definition"));
    }

    return {
      title: "Change a report",
      summary:
        changed.length === 0
          ? translate("Change this report's settings.")
          : translate("Change this report's {0}.", formatList(changed)),
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

  create_location: (args) => {
    const name = text(args.name);
    const street = [text(args.addressLine1), text(args.addressLine2)]
      .filter((part) => part !== "")
      .join(", ");
    const place = [text(args.city), [text(args.state), text(args.postalCode)].join(" ").trim()]
      .filter((part) => part !== "")
      .join(", ");

    return {
      title: "Create a location",
      summary: createLocationSummary(name, place),
      highlights: facts(
        fact("Name", name),
        fact("Address", [street, place].filter((part) => part !== "").join(", ")),
        fact("Category", shortRef(text(args.locationCategoryId))),
      ),
      covered: [
        "name",
        "addressLine1",
        "addressLine2",
        "city",
        "state",
        "postalCode",
        "locationCategoryId",
      ],
      reversible: true,
    };
  },

  evaluate_service_failures: (args) => ({
    title: "Check for service failures",
    summary:
      args.force === true
        ? translate(
            "Run the late-stop check on this shipment, re-checking stops already evaluated.",
          )
        : translate("Run the late-stop check on this shipment."),
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
      summary: translate(
        "Send driver {0} “{1}” in Dash.",
        shortRef(text(args.workerId)) || "?",
        text(args.title),
      ),
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
    summary: translate(
      "Send this shipment's customer “{0}” from the organization's letterhead.",
      text(args.subject),
    ),
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

  approve_detention: (args) => ({
    title: "Approve detention charge",
    summary: "Approve this detention charge as calculated, so its shipment can be invoiced.",
    highlights: facts(fact("Evidence", text(args.evidence))),
    covered: ["occurrenceId", "evidence"],
    reversible: false,
  }),

  waive_detention: (args) => ({
    title: "Waive detention",
    summary: translate(
      "Waive this detention charge as {0}.",
      midSentence(humanizeEnum(text(args.reason))),
    ),
    highlights: facts(fact("Note", text(args.note))),
    covered: ["occurrenceId", "reason", "note"],
    reversible: false,
  }),

  fork_report: (args) => {
    const key = text(args.reportKey);
    const name = text(args.name);

    return {
      title: "Copy a report",
      summary: forkReportSummary(key, name),
      highlights: [],
      covered: ["reportKey", "name"],
      reversible: true,
    };
  },

  request_missing_docs: (args) => {
    const to = list(args.to);
    const documents = list(args.requestedDocuments);
    const who = text(args.customerName) || to.join(", ");
    const pro = text(args.shipmentProNumber);

    return {
      title: "Email the customer",
      summary: requestDocsSummary(who, documents.length, pro),
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
      summary: translate(
        "Replace the additional charges on this billing item with {0, plural, one {# charge} other {# charges}}.",
        charges.length,
      ),
      highlights: charges.slice(0, 4).map((charge, index) => {
        const entry = (charge ?? {}) as Args;
        const name =
          text(entry.description) ||
          shortRef(text(entry.accessorialChargeId)) ||
          translate("Charge {0}", index + 1);
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
      summary: tractor
        ? translate("Put driver {0} on this move with tractor {1}.", driver || "?", tractor)
        : translate("Put driver {0} on this move.", driver || "?"),
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

  create_shipment: (args) => {
    const draft = (args.shipment ?? {}) as Args;
    const moves = Array.isArray(draft.moves) ? (draft.moves as Args[]) : [];
    const stops = moves.reduce(
      (count, move) => count + (Array.isArray(move.stops) ? move.stops.length : 0),
      0,
    );
    const bol = text(draft.bol);

    return {
      title: "Enter a shipment",
      summary: createShipmentSummary(bol, stops, Boolean(args.sourceDocumentId)),
      highlights: facts(
        fact("Customer", shortRef(text(draft.customerId))),
        fact("Service type", shortRef(text(draft.serviceTypeId))),
        fact("Pieces", text(draft.pieces)),
        fact("Weight", text(draft.weight) ? translate("{0} lb", text(draft.weight)) : ""),
        fact("Freight charge", text(draft.freightChargeAmount)),
        fact("Source document", shortRef(text(args.sourceDocumentId))),
      ),
      covered: ["shipment", "sourceDocumentId"],
      reversible: false,
    };
  },

  update_shipment: (args) => {
    const changed = Object.keys(args).filter((key) => key !== "shipmentId");

    return {
      title: "Change a shipment",
      summary:
        changed.length === 0
          ? translate("Change nothing on this shipment.")
          : translate(
              "Change {0} on this shipment.",
              changed.map((key) => midSentence(humanizeEnum(key))).join(", "),
            ),
      highlights: facts(
        fact("Customer", shortRef(text(args.customerId))),
        fact("Service type", shortRef(text(args.serviceTypeId))),
        fact("Shipment type", shortRef(text(args.shipmentTypeId))),
        fact("BOL", text(args.bol)),
        fact("Pieces", text(args.pieces)),
        fact("Weight", text(args.weight) ? translate("{0} lb", text(args.weight)) : ""),
        fact(
          "Temperature",
          [text(args.temperatureMin), text(args.temperatureMax)].filter(Boolean).join(" to "),
        ),
      ),
      covered: [
        "shipmentId",
        "customerId",
        "serviceTypeId",
        "shipmentTypeId",
        "tractorTypeId",
        "trailerTypeId",
        "bol",
        "pieces",
        "weight",
        "temperatureMin",
        "temperatureMax",
      ],
      reversible: true,
    };
  },

  tender_move_to_routing_guide: (args) => ({
    title: "Tender down the routing guide",
    summary: args.routingGuideId
      ? translate(
          "Offer this move to carriers down routing guide {0}, in the guide's order at its rates.",
          shortRef(text(args.routingGuideId)),
        )
      : translate(
          "Offer this move to carriers down the lane's routing guide, in the guide's order at its rates.",
        ),
    highlights: facts(fact("Routing guide", shortRef(text(args.routingGuideId)))),
    covered: ["shipmentMoveId", "routingGuideId"],
    reversible: true,
  }),

  tender_move_to_carriers: (args) => {
    const lines = Array.isArray(args.lines) ? (args.lines as Args[]) : [];
    const sequential = text(args.mode) === "SpotSequential";

    return {
      title: "Tender to carriers",
      summary: sequential
        ? translate(
            "Offer this move to {0, plural, one {# carrier} other {# carriers}} one after another at the rates below.",
            lines.length,
          )
        : translate(
            "Offer this move to {0, plural, one {# carrier} other {# carriers}} all at once at the rates below.",
            lines.length,
          ),
      highlights: facts(
        ...lines.map((line, index) =>
          fact(
            translate("Carrier {0}", index + 1),
            [
              shortRef(text(line.carrierId)),
              text(line.rate)
                ? `${text(line.rate)}${text(line.rateMethod) === "PerMile" ? "/mi" : ""}`
                : "",
            ]
              .filter(Boolean)
              .join(" at "),
          ),
        ),
        fact("Insurance warnings", args.overrideInsuranceWarnings === true ? "overridden" : ""),
      ),
      covered: ["shipmentMoveId", "mode", "lines", "overrideInsuranceWarnings"],
      reversible: true,
    };
  },

  post_invoices: (args) => {
    const count = list(args.invoiceIds).length;

    return {
      title: "Post invoices",
      summary: translate(
        "Post {0, plural, one {# draft invoice} other {# draft invoices}} to the ledger and queue each for the accounting system.",
        count,
      ),
      covered: ["invoiceIds"],
      reversible: false,
    };
  },

  send_invoices: (args) => {
    const count = list(args.invoiceIds).length;

    return {
      title: "Send invoices",
      summary: translate(
        "Email {0, plural, one {# invoice} other {# invoices}} to the customers they bill, as their billing profiles say.",
        count,
      ),
      covered: ["invoiceIds"],
      reversible: false,
    };
  },

  approve_billing_queue_items: (args) => {
    const count = list(args.billingQueueItemIds).length;

    return {
      title: "Approve billing items",
      summary: translate(
        "Approve {0, plural, one {# billing item} other {# billing items}}, creating the draft invoice each bills on.",
        count,
      ),
      highlights: facts(fact("Review notes", text(args.reviewNotes))),
      covered: ["billingQueueItemIds", "reviewNotes"],
      reversible: false,
    };
  },

  approve_driver_settlements: (args) => {
    const count = list(args.settlementIds).length;

    return {
      title: "Approve driver settlements",
      summary: translate(
        "Approve {0, plural, one {# driver settlement} other {# driver settlements}}, committing to pay each driver what it comes to.",
        count,
      ),
      covered: ["settlementIds"],
      reversible: false,
    };
  },

  post_driver_settlements: (args) => {
    const count = list(args.settlementIds).length;

    return {
      title: "Post driver settlements",
      summary: translate(
        "{0, plural, one {Post # approved driver settlement to the ledger, queue it for the accounting system and tell the driver.} other {Post # approved driver settlements to the ledger, queue each for the accounting system and tell each driver.}}",
        count,
      ),
      covered: ["settlementIds"],
      reversible: false,
    };
  },

  approve_carrier_settlements: (args) => {
    const count = list(args.settlementIds).length;

    return {
      title: "Approve carrier settlements",
      summary: translate(
        "Approve {0, plural, one {# carrier settlement} other {# carrier settlements}}, committing to pay each carrier what it comes to.",
        count,
      ),
      covered: ["settlementIds"],
      reversible: false,
    };
  },

  post_carrier_settlements: (args) => {
    const count = list(args.settlementIds).length;

    return {
      title: "Post carrier settlements",
      summary: translate(
        "{0, plural, one {Post # approved carrier settlement to the ledger and queue it for the accounting system.} other {Post # approved carrier settlements to the ledger and queue each for the accounting system.}}",
        count,
      ),
      covered: ["settlementIds"],
      reversible: false,
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
        ? translate("Record a {0} case for a person to resolve.", midSentence(category))
        : translate("Record a case for a person to resolve."),
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
      summary: file
        ? translate("Attach {0} to this shipment's billing item.", file)
        : translate("Attach a document to this shipment's billing item."),
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
      summary: translate("Run this tool with the values below."),
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

/**
 * Whether the client words this tool's proposals itself. One it does not is
 * summed up as "Run … with the values below", which says nothing a surface
 * holding the tool's own preview could not say better.
 */
export function hasPresenter(toolName: string): boolean {
  return Object.hasOwn(PRESENTERS, toolName);
}
