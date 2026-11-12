import type { Tone } from "@/components/kpi/tone";
import { shortDigest } from "@/lib/digest";
import type { AssistantProposal } from "@/types/assistant";
import { formatList } from "@trenova/shared/i18n/format";
import { translate } from "@trenova/shared/i18n/runtime";
import { argumentRows, humanizeToolName, isRecordId, isRecordIdList } from "./proposal-state";
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
 * A PULID is a storage key, not a name, and no part of one is ever shown: a
 * person cannot tell two records apart by six characters of an id any better
 * than by twenty-six. The record itself is named where the change is previewed,
 * from the server's own read of it.
 */
export function isOpaqueId(value: string): boolean {
  return isRecordId(value.trim());
}

/** What an argument names, when it is a name a person would know; nothing when it is an id. */
export function namedRef(value: string): string {
  const trimmed = value.trim();

  return isOpaqueId(trimmed) ? "" : trimmed;
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
  dimension?: { field?: unknown };
  measure?: { field?: unknown; agg?: unknown };
  calc?: unknown;
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
    filters?: unknown;
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
  const field = text(column.dimension?.field ?? column.measure?.field);
  const label = text(column.label);
  const agg = text(column.measure?.agg);

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

function countFilters(filter: unknown): number {
  if (typeof filter !== "object" || filter === null) {
    return 0;
  }
  const node = filter as { and?: unknown; or?: unknown };
  const children = Array.isArray(node.and) ? node.and : Array.isArray(node.or) ? node.or : null;
  if (!children) {
    return 1;
  }

  return children.reduce((sum: number, child) => sum + countFilters(child), 0);
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
  report_create: (args) => {
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

  report_update: (args) => {
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

  location_create: (args) => {
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
        fact("Category", namedRef(text(args.locationCategoryId))),
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

  exception_evaluate_service_failures: (args) => ({
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

  exception_resolve_service_failure: (args) => ({
    title: "Resolve a service failure",
    summary: "Close this service failure with the reason and note below.",
    highlights: facts(
      fact("Reason code", namedRef(text(args.reasonCodeId))),
      fact("Note", text(args.note)),
    ),
    covered: ["serviceFailureId", "reasonCodeId", "note"],
    reversible: false,
  }),

  worker_send_message: (args) => {
    const priority = text(args.priority);
    const driver = namedRef(text(args.workerId));

    return {
      title: "Message the driver",
      summary:
        driver === ""
          ? translate("Send the driver “{0}” in Dash.", text(args.title))
          : translate("Send driver {0} “{1}” in Dash.", driver, text(args.title)),
      severity:
        priority === "critical" || priority === "high"
          ? { label: humanizeEnum(priority[0].toUpperCase() + priority.slice(1)), tone: "warning" }
          : null,
      highlights: facts(fact("Message", text(args.message))),
      covered: ["workerId", "title", "message", "priority", "shipmentId"],
      reversible: false,
    };
  },

  customer_send_email: (args) => ({
    title: "Email the customer",
    summary: translate(
      "Send this shipment's customer “{0}” from the organization's letterhead.",
      text(args.subject),
    ),
    highlights: facts(fact("Message", text(args.body))),
    covered: ["shipmentId", "profileId", "subject", "body"],
    reversible: false,
  }),

  detention_send_notice: () => ({
    title: "Send a detention notice",
    summary: "Send the customer the detention notice for this occurrence.",
    highlights: [],
    covered: ["occurrenceId"],
    reversible: false,
  }),

  detention_approve: (args) => ({
    title: "Approve detention charge",
    summary: "Approve this detention charge as calculated, so its shipment can be invoiced.",
    highlights: facts(fact("Evidence", text(args.evidence))),
    covered: ["occurrenceId", "evidence"],
    reversible: false,
  }),

  detention_waive: (args) => ({
    title: "Waive detention",
    summary: translate(
      "Waive this detention charge as {0}.",
      midSentence(humanizeEnum(text(args.reason))),
    ),
    highlights: facts(fact("Note", text(args.note))),
    covered: ["occurrenceId", "reason", "note"],
    reversible: false,
  }),

  report_fork: (args) => {
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

  shipment_request_documents: (args) => {
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

  billing_correct_charge_codes: (args) => {
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
          namedRef(text(entry.accessorialChargeId)) ||
          translate("Charge {0}", index + 1);
        const amount = text(entry.amount);

        return { label: name, value: amount === "" ? "—" : amount };
      }),
      covered: ["billingQueueItemId", "additionalCharges"],
      reversible: true,
    };
  },

  shipment_reschedule_stop: (args) => ({
    title: "Move the appointment",
    summary: "Move this stop's planned window; the load's trading partners are told of the change.",
    highlights: facts(
      fact("Window opens", text(args.windowStart)),
      fact("Window closes", text(args.windowEnd)),
    ),
    covered: ["shipmentId", "stopId", "windowStart", "windowEnd"],
    reversible: true,
  }),

  receivable_send_payment_reminder: (args) => ({
    title: "Send a payment reminder",
    summary: "Email this customer a reminder of what is past due.",
    highlights: facts(fact("Note", text(args.note))),
    covered: ["customerId", "note"],
    reversible: false,
  }),

  receivable_schedule_follow_up: (args) => ({
    title: "Schedule a follow-up",
    summary: "Put this customer back in front of collections on the day given.",
    highlights: facts(fact("Follow up on", text(args.followUpOn)), fact("Note", text(args.note))),
    covered: ["customerId", "followUpOn", "note"],
    reversible: true,
  }),

  invoice_resolve_dispute: (args) => ({
    title: "Resolve the dispute",
    summary: "Close this invoice dispute with the resolution given.",
    highlights: facts(
      fact("Resolution", humanizeEnum(text(args.resolution))),
      fact("Notes", text(args.resolutionNotes)),
    ),
    covered: ["disputeId", "resolution", "resolutionAdjustmentId", "resolutionNotes"],
    reversible: false,
  }),

  dispatch_assign_move: (args) => {
    const driver = namedRef(text(args.primaryWorkerId));
    const tractor = namedRef(text(args.tractorId));
    const trailer = namedRef(text(args.trailerId));

    return {
      title: "Assign a driver",
      summary:
        driver === ""
          ? tractor
            ? translate("Put the chosen driver on this move with tractor {0}.", tractor)
            : translate("Put the chosen driver on this move.")
          : tractor
            ? translate("Put driver {0} on this move with tractor {1}.", driver, tractor)
            : translate("Put driver {0} on this move.", driver),
      highlights: facts(
        fact("Driver", driver),
        fact("Second driver", namedRef(text(args.secondaryWorkerId))),
        fact("Tractor", tractor),
        fact("Trailer", trailer),
      ),
      covered: ["shipmentMoveId", "primaryWorkerId", "secondaryWorkerId", "tractorId", "trailerId"],
      reversible: true,
    };
  },

  shipment_create: (args) => {
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
        fact("Customer", namedRef(text(draft.customerId))),
        fact("Service type", namedRef(text(draft.serviceTypeId))),
        fact("Pieces", text(draft.pieces)),
        fact("Weight", text(draft.weight) ? translate("{0} lb", text(draft.weight)) : ""),
        fact("Freight charge", text(draft.freightChargeAmount)),
        fact("Source document", namedRef(text(args.sourceDocumentId))),
      ),
      covered: ["shipment", "sourceDocumentId"],
      reversible: false,
    };
  },

  shipment_update: (args) => {
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
        fact("Customer", namedRef(text(args.customerId))),
        fact("Service type", namedRef(text(args.serviceTypeId))),
        fact("Shipment type", namedRef(text(args.shipmentTypeId))),
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

  dispatch_create_tender_from_routing_guide: (args) => ({
    title: "Tender down the routing guide",
    summary: namedRef(text(args.routingGuideId))
      ? translate(
          "Offer this move to carriers down routing guide {0}, in the guide's order at its rates.",
          namedRef(text(args.routingGuideId)),
        )
      : args.routingGuideId
        ? translate(
            "Offer this move to carriers down the chosen routing guide, in the guide's order at its rates.",
          )
        : translate(
            "Offer this move to carriers down the lane's routing guide, in the guide's order at its rates.",
          ),
    highlights: facts(fact("Routing guide", namedRef(text(args.routingGuideId)))),
    covered: ["shipmentMoveId", "routingGuideId"],
    reversible: true,
  }),

  dispatch_create_tender: (args) => {
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
              namedRef(text(line.carrierId)),
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

  edi_apply_connection_draft: (args) => {
    const counted = (label: string, value: unknown) =>
      fact(label, Array.isArray(value) && value.length > 0 ? String(value.length) : "");

    return {
      title: "Save the EDI connection from the guide",
      summary: text(args.ediPartnerId)
        ? translate(
            "Update a trading partner you already have from its revised guide. Only what the guide changed is written, and nothing is turned on.",
          )
        : translate(
            "Set up a trading partner from its guide: the partner, its communication profile, a document profile per document type, control number starts and code mappings, all inactive until you enter the credentials and turn them on.",
          ),
      highlights: facts(
        counted("Settings from the guide", args.values),
        counted("Segment rules", args.requirements),
        counted("Code mappings", args.mappings),
        counted("Gaps noted", args.gaps),
        counted("Your answers", args.resolutions),
        fact("Guide digest", shortDigest(text(args.documentDigest))),
      ),
      covered: [
        "documentId",
        "ediPartnerId",
        "customerId",
        "values",
        "requirements",
        "mappings",
        "gaps",
        "resolutions",
        "documentDigest",
      ],
      reversible: false,
    };
  },

  invoice_post: (args) => {
    const count = list(args.invoiceIds).length;

    return {
      title: translate("{0, plural, one {Post invoice} other {Post invoices}}", count),
      summary: translate(
        "{0, plural, one {Post # draft invoice to the ledger and queue it for the accounting system.} other {Post # draft invoices to the ledger and queue each for the accounting system.}}",
        count,
      ),
      covered: ["invoiceIds"],
      reversible: false,
    };
  },

  invoice_send: (args) => {
    const count = list(args.invoiceIds).length;

    return {
      title: translate("{0, plural, one {Send invoice} other {Send invoices}}", count),
      summary: translate(
        "Email {0, plural, one {# invoice} other {# invoices}} to the customers they bill, as their billing profiles say.",
        count,
      ),
      covered: ["invoiceIds"],
      reversible: false,
    };
  },

  billing_approve_queue_item: (args) => {
    const count = list(args.billingQueueItemIds).length;

    return {
      title: translate(
        "{0, plural, one {Approve billing item} other {Approve billing items}}",
        count,
      ),
      summary: translate(
        "Approve {0, plural, one {# billing item} other {# billing items}}, creating the draft invoice each bills on.",
        count,
      ),
      highlights: facts(fact("Review notes", text(args.reviewNotes))),
      covered: ["billingQueueItemIds", "reviewNotes"],
      reversible: false,
    };
  },

  settlement_approve_driver: (args) => {
    const count = list(args.settlementIds).length;

    return {
      title: translate(
        "{0, plural, one {Approve driver settlement} other {Approve driver settlements}}",
        count,
      ),
      summary: translate(
        "Approve {0, plural, one {# driver settlement} other {# driver settlements}}, committing to pay each driver what it comes to.",
        count,
      ),
      covered: ["settlementIds"],
      reversible: false,
    };
  },

  settlement_post_driver: (args) => {
    const count = list(args.settlementIds).length;

    return {
      title: translate(
        "{0, plural, one {Post driver settlement} other {Post driver settlements}}",
        count,
      ),
      summary: translate(
        "{0, plural, one {Post # approved driver settlement to the ledger, queue it for the accounting system and tell the driver.} other {Post # approved driver settlements to the ledger, queue each for the accounting system and tell each driver.}}",
        count,
      ),
      covered: ["settlementIds"],
      reversible: false,
    };
  },

  settlement_approve_carrier: (args) => {
    const count = list(args.settlementIds).length;

    return {
      title: translate(
        "{0, plural, one {Approve carrier settlement} other {Approve carrier settlements}}",
        count,
      ),
      summary: translate(
        "Approve {0, plural, one {# carrier settlement} other {# carrier settlements}}, committing to pay each carrier what it comes to.",
        count,
      ),
      covered: ["settlementIds"],
      reversible: false,
    };
  },

  settlement_post_carrier: (args) => {
    const count = list(args.settlementIds).length;

    return {
      title: translate(
        "{0, plural, one {Post carrier settlement} other {Post carrier settlements}}",
        count,
      ),
      summary: translate(
        "{0, plural, one {Post # approved carrier settlement to the ledger and queue it for the accounting system.} other {Post # approved carrier settlements to the ledger and queue each for the accounting system.}}",
        count,
      ),
      covered: ["settlementIds"],
      reversible: false,
    };
  },

  billing_start_review: (args) => {
    const count = list(args.billingQueueItemIds).length;

    return {
      title: "Send to review",
      summary: translate(
        "Move {0, plural, one {# billing item} other {# billing items}} into review so a biller picks {0, plural, one {it} other {them}} up.",
        count,
      ),
      covered: ["billingQueueItemIds"],
      reversible: true,
    };
  },

  agent_raise_exception: (args) => {
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
      // The subject's kind and key and the evidence blob belong to the audit
      // trail. Severity and category are already the badge and the sentence.
      covered: [
        "subjectType",
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

  worker_place_dispatch_hold: (args) => ({
    title: "Place a dispatch hold",
    summary: translate(
      "Take this driver off dispatch. They get no new freight until someone lifts the hold.",
    ),
    highlights: facts(fact("Reason", text(args.reason))),
    covered: ["workerId", "reason"],
    reversible: false,
  }),

  worker_request_credential_renewal: (args) => {
    const papers = Array.isArray(args.credentialIds) ? args.credentialIds.length : 0;

    return {
      title: "Ask for renewed papers",
      summary:
        papers > 0
          ? translate(
              "Ask the driver, in one message, to renew {0, plural, one {# credential} other {# credentials}}. Nothing on file changes.",
              papers,
            )
          : translate("Ask the driver to renew their papers. Nothing on file changes."),
      highlights: facts(fact("Message", text(args.note))),
      covered: ["workerId", "credentialIds", "note"],
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
  // A key that names ids ("credentialIds") is said as what it counts.
  const rows = argumentRows(args).map((row) => ({
    label: humanizeEnum(
      isRecordIdList(args[row.key]) ? row.key.replace(/(?<=[a-z])Ids$/u, "") : row.key,
    ),
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
