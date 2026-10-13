import { formatNumber } from "@trenova/shared/i18n/format";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type {
  EDIInspectionDiagnostic,
  EDIMessage,
  EDIX12Inspection,
} from "@trenova/shared/types/edi";
import type { CardCheckState, CardFact, CardTone } from "../../record-card/record-card";
import { cardDate, cardMoment, money, waitText } from "../../record-card/record-card-values";

type TransactionSet = EDIMessage["transactionSet"];
type Payload = NonNullable<EDIMessage["payloadSnapshot"]>;

/** What a transaction set is, in the words a dispatcher uses for it. */
export function transactionName(set: TransactionSet, t: TranslateFn): string {
  switch (set) {
    case "204":
      return t("Load tender");
    case "210":
      return t("Freight invoice");
    case "214":
      return t("Shipment status");
    case "990":
      return t("Tender response");
    case "997":
      return t("Functional acknowledgment");
    case "999":
      return t("Implementation acknowledgment");
  }
}

/** Where the message stands, as the pill at the card's head says it. */
export type MessageStanding = {
  label: string;
  tone: CardTone;
  mark: "dot" | "check" | "pause";
};

/**
 * Where the message stands, worst news first: one that could not be
 * generated, one the connection gave up on or failed to deliver, one the
 * partner rejected, then how far a good one has come. An inbound message
 * stands as read, or as one that could not be.
 */
export function messageStanding(message: EDIMessage, t: TranslateFn): MessageStanding {
  if (message.status === "Failed") {
    return {
      label: message.direction === "Inbound" ? t("Could not be read") : t("Not generated"),
      tone: "warn",
      mark: "pause",
    };
  }
  if (message.direction === "Inbound") {
    return inboundStanding(message, t);
  }
  switch (message.deliveryStatus) {
    case "DeadLettered":
      return { label: t("Dead-lettered"), tone: "warn", mark: "pause" };
    case "Failed":
      return { label: t("Delivery failed"), tone: "warn", mark: "pause" };
    case "Queued":
      return { label: t("Queued"), tone: "draft", mark: "dot" };
    case "Sending":
      return { label: t("Sending"), tone: "progress", mark: "dot" };
    default:
      break;
  }
  switch (message.ackStatus) {
    case "Rejected":
      return { label: t("Rejected by partner"), tone: "warn", mark: "pause" };
    case "Failed":
      return { label: t("Acknowledgment unreadable"), tone: "warn", mark: "pause" };
    case "Accepted":
      return { label: t("Accepted"), tone: "success", mark: "check" };
    case "Pending":
      return message.deliveryStatus === "Sent"
        ? { label: t("Awaiting acknowledgment"), tone: "progress", mark: "dot" }
        : { label: t("Generated"), tone: "ink", mark: "dot" };
    default:
      return message.deliveryStatus === "Sent"
        ? { label: t("Delivered"), tone: "success", mark: "check" }
        : { label: t("Generated"), tone: "ink", mark: "dot" };
  }
}

/**
 * An inbound message stands as received, unless it is an acknowledgment, which
 * stands as its verdict on what this organization sent: a rejection or an
 * acceptance with errors asks for a look.
 */
function inboundStanding(message: EDIMessage, t: TranslateFn): MessageStanding {
  const ack = message.payloadSnapshot?.functionalAck ?? message.payloadSnapshot?.implementationAck;
  const code = ack
    ? text(ack.transactionAcknowledgmentCode) || text(ack.groupAcknowledgmentCode)
    : "";
  if (code === "E") return { label: t("Accepted with errors"), tone: "warn", mark: "pause" };
  if (code === "P") return { label: t("Partly accepted"), tone: "warn", mark: "pause" };
  if (code && !ACCEPT_CODES.has(code)) {
    return { label: t("Rejected our message"), tone: "warn", mark: "pause" };
  }
  return { label: t("Received"), tone: "success", mark: "check" };
}

/** How a stop along the message's way stands: reached, under way, failed, ahead, or not on its way at all. */
export type StationState = "done" | "now" | "failed" | "ahead" | "skipped";

/** One stop along the way the message travels between this organization and the partner. */
export type JourneyStation = {
  key: "made" | "sent" | "answered" | "received" | "read" | "tender";
  label: string;
  detail: string;
  at: number | null;
  state: StationState;
};

/**
 * The way the message travels, stop by stop. An outbound message is
 * generated, delivered over the partner's connection and answered by their
 * acknowledgment; an inbound one is received, read and, for a load tender,
 * turned into a tender to approve. Each stop says how it stands and when.
 */
export function messageJourney(message: EDIMessage, t: TranslateFn): JourneyStation[] {
  return message.direction === "Inbound" ? inboundJourney(message, t) : outboundJourney(message, t);
}

function outboundJourney(message: EDIMessage, t: TranslateFn): JourneyStation[] {
  const generated = message.status !== "Failed";
  const made: JourneyStation = {
    key: "made",
    label: t("Generated"),
    detail: generated
      ? t("{0, plural, one {# segment} other {# segments}}", message.segmentCount)
      : t("It could not be generated"),
    at: message.generatedAt,
    state: generated ? "done" : "failed",
  };
  return [made, deliveryStation(message, generated, t), answerStation(message, generated, t)];
}

function deliveryStation(message: EDIMessage, generated: boolean, t: TranslateFn): JourneyStation {
  const attempts = message.deliveryAttempts;
  const station = { key: "sent" as const, label: t("Delivered") };
  if (!generated) {
    return { ...station, detail: t("Never sent"), at: null, state: "skipped" };
  }
  switch (message.deliveryStatus) {
    case "Queued":
      return { ...station, detail: t("Queued to send"), at: null, state: "ahead" };
    case "Sending":
      return {
        ...station,
        detail: t("Sending, attempt {0}", formatNumber(Math.max(attempts, 1))),
        at: message.deliveryLastAttemptAt ?? null,
        state: "now",
      };
    case "Sent":
      return {
        ...station,
        detail: t("{0, plural, one {On the first attempt} other {After # attempts}}", attempts),
        at: message.deliverySentAt ?? null,
        state: "done",
      };
    case "Failed":
      return {
        ...station,
        detail: t("{0, plural, one {# attempt failed} other {# attempts failed}}", attempts),
        at: message.deliveryLastAttemptAt ?? null,
        state: "failed",
      };
    case "DeadLettered":
      return {
        ...station,
        detail: t("Gave up after {0, plural, one {# attempt} other {# attempts}}", attempts),
        at: message.deliveryLastAttemptAt ?? null,
        state: "failed",
      };
    default:
      return message.partner?.kind === "Internal"
        ? { ...station, detail: t("Inside Trenova"), at: message.generatedAt, state: "done" }
        : { ...station, detail: t("Not sent over a connection"), at: null, state: "skipped" };
  }
}

function answerStation(message: EDIMessage, generated: boolean, t: TranslateFn): JourneyStation {
  const station = { key: "answered" as const, label: t("Acknowledged") };
  const at = message.ackReceivedAt ?? null;
  if (!generated) {
    return { ...station, detail: t("Nothing to acknowledge"), at: null, state: "skipped" };
  }
  switch (message.ackStatus) {
    case "Accepted":
      return { ...station, detail: t("Accepted by the partner"), at, state: "done" };
    case "Rejected":
      return { ...station, detail: t("Rejected by the partner"), at, state: "failed" };
    case "Failed":
      return { ...station, detail: t("Its acknowledgment could not be read"), at, state: "failed" };
    case "Pending":
      return {
        ...station,
        detail:
          message.deliveryStatus === "Sent"
            ? t("Waiting on the partner")
            : t("Once it is delivered"),
        at: null,
        state: "ahead",
      };
    default:
      return { ...station, detail: t("None expected"), at: null, state: "skipped" };
  }
}

function inboundJourney(message: EDIMessage, t: TranslateFn): JourneyStation[] {
  const read = message.status !== "Failed";
  const stations: JourneyStation[] = [
    {
      key: "received",
      label: t("Received"),
      detail: message.inboundFileId ? t("In an inbound file") : t("Sent to us directly"),
      at: message.generatedAt,
      state: "done",
    },
    {
      key: "read",
      label: t("Read"),
      detail: read
        ? t("{0, plural, one {# segment} other {# segments}}", message.segmentCount)
        : t("It could not be read"),
      at: read ? message.generatedAt : null,
      state: read ? "done" : "failed",
    },
  ];
  if (message.transactionSet === "204") {
    stations.push(
      message.transferId
        ? {
            key: "tender",
            label: t("Tendered"),
            detail: t("Waiting in inbound tenders"),
            at: null,
            state: "done",
          }
        : {
            key: "tender",
            label: t("Tendered"),
            detail: read ? t("No tender made from it") : t("Nothing to tender"),
            at: null,
            state: "skipped",
          },
    );
  }
  return stations;
}

/** Where the envelope rests: the furthest stop the message reached, or failed at. */
export function journeyPosition(stations: readonly JourneyStation[]): number {
  let position = 0;
  stations.forEach((station, index) => {
    if (station.state === "done" || station.state === "now" || station.state === "failed") {
      position = index;
    }
  });
  return position;
}

/** One layer of the X12 envelope the message travels in, from the interchange in. */
export type EnvelopeLayer = {
  key: "isa" | "gs" | "st";
  /** The segment pair that opens and closes the layer. */
  pair: string;
  label: string;
  control: string;
  detail: string;
  /** Whether the closing segment agrees with the opening one; null before the X12 is inspected. */
  check: Extract<CardCheckState, "ok" | "bad"> | null;
  checkText: string;
};

/**
 * The message's X12 envelope from the outside in: the interchange (ISA/IEA),
 * the functional group (GS/GE) and the transaction set (ST/SE), each with its
 * control number and, once the X12 is inspected, whether its trailer closes it
 * with the same control number and count it opened with.
 */
export function envelopeLayers(
  message: EDIMessage,
  inspection: EDIX12Inspection | undefined,
  t: TranslateFn,
): EnvelopeLayer[] {
  const envelope = inspection?.envelope;
  const group = inspection?.groups[0];
  const transaction = inspection?.transactions[0];

  const isaCheck = !envelope
    ? null
    : !envelope.ieaControlNumber
      ? { check: "bad" as const, checkText: t("No IEA trailer found") }
      : envelope.ieaControlNumber !== envelope.isaControlNumber
        ? { check: "bad" as const, checkText: t("IEA closes a different interchange") }
        : envelope.expectedGroups !== envelope.actualGroups
          ? {
              check: "bad" as const,
              checkText: t(
                "IEA counts {0, plural, one {# group} other {# groups}}, found {1}",
                envelope.expectedGroups,
                formatNumber(envelope.actualGroups),
              ),
            }
          : { check: "ok" as const, checkText: t("IEA closes it") };

  const gsCheck = !inspection
    ? null
    : !group
      ? { check: "bad" as const, checkText: t("No functional group found") }
      : !group.geControlNumber
        ? { check: "bad" as const, checkText: t("No GE trailer found") }
        : group.geControlNumber !== group.gsControlNumber
          ? { check: "bad" as const, checkText: t("GE closes a different group") }
          : group.expectedCount !== group.actualCount
            ? {
                check: "bad" as const,
                checkText: t(
                  "GE counts {0, plural, one {# transaction} other {# transactions}}, found {1}",
                  group.expectedCount,
                  formatNumber(group.actualCount),
                ),
              }
            : { check: "ok" as const, checkText: t("GE closes it") };

  const stCheck = !inspection
    ? null
    : !transaction
      ? { check: "bad" as const, checkText: t("No transaction set found") }
      : !transaction.seControlNumber
        ? { check: "bad" as const, checkText: t("No SE trailer found") }
        : transaction.seControlNumber !== transaction.stControlNumber
          ? { check: "bad" as const, checkText: t("SE closes a different transaction") }
          : transaction.expectedSegments !== transaction.actualSegments
            ? {
                check: "bad" as const,
                checkText: t(
                  "SE counts {0, plural, one {# segment} other {# segments}}, found {1}",
                  transaction.expectedSegments,
                  formatNumber(transaction.actualSegments),
                ),
              }
            : { check: "ok" as const, checkText: t("SE closes it") };

  return [
    {
      key: "isa",
      pair: "ISA · IEA",
      label: t("Interchange"),
      control: message.interchangeControlNumber,
      detail: t("X12 {0}", message.x12Version),
      check: isaCheck?.check ?? null,
      checkText: isaCheck?.checkText ?? "",
    },
    {
      key: "gs",
      pair: "GS · GE",
      label: t("Functional group"),
      control: message.groupControlNumber,
      detail: group?.functionalIdCode ? t("Group {0}", group.functionalIdCode) : "",
      check: gsCheck?.check ?? null,
      checkText: gsCheck?.checkText ?? "",
    },
    {
      key: "st",
      pair: "ST · SE",
      label: t("Transaction set"),
      control: message.transactionControlNumber,
      detail: t(
        "{0} · {1, plural, one {# segment} other {# segments}}",
        message.transactionSet,
        message.segmentCount,
      ),
      check: stCheck?.check ?? null,
      checkText: stCheck?.checkText ?? "",
    },
  ];
}

/** A stop a load tender names, as the plain reading lists it. */
export type ReadingStop = { key: string; kind: string; place: string; window: string };

/** A charge line an invoice carries, as the plain reading lists it. */
export type ReadingLine = { key: string; label: string; amount: string };

/** What the message says, decoded from its payload into a headline and the facts behind it. */
export type PlainReading = {
  headline: string;
  /** Where or to whom, under the headline. */
  line: string;
  /** How the headline reads: a delivery or an acceptance is good news, a delay or rejection is not. */
  state: CardCheckState;
  /** A figure set beside the headline, such as an invoice's total. */
  figure: string;
  facts: CardFact[];
  stops: ReadingStop[];
  lines: ReadingLine[];
  /** Where an acknowledgment found fault, segment by segment. */
  faults: ReadingLine[];
};

const STATUS_WORDS: Record<string, (t: TranslateFn) => string> = {
  X3: (t) => t("Arrived at pickup"),
  AF: (t) => t("Departed pickup"),
  X6: (t) => t("En route to delivery"),
  X1: (t) => t("Arrived at delivery"),
  D1: (t) => t("Delivered"),
  CD: (t) => t("Departed delivery"),
  A3: (t) => t("Shipment delayed"),
  SD: (t) => t("Shipment delayed"),
  A7: (t) => t("Shipment canceled"),
};

const DELAY_CODES = new Set(["A3", "SD", "A7"]);
const ACCEPT_CODES = new Set(["A", "E"]);

/**
 * What the message says, read from its decoded payload rather than its
 * segments: a 214's status at its stop, a 204's load and stops, a 210's bill
 * and charges, a 990's answer, a 997 or 999's verdict on what it answers.
 * Null when no payload was kept.
 */
export function plainReading(message: EDIMessage, t: TranslateFn): PlainReading | null {
  const payload = message.payloadSnapshot;
  if (!payload) return null;
  switch (message.transactionSet) {
    case "214":
      return payload.shipmentStatus ? statusReading(payload.shipmentStatus, t) : null;
    case "204": {
      const tender = payload.loadTender ?? payload.shipment;
      return tender ? tenderReading(tender, payload, t) : null;
    }
    case "210":
      return payload.invoice ? invoiceReading(payload.invoice, t) : null;
    case "990":
      return payload.tenderResponse ? responseReading(payload.tenderResponse, t) : null;
    case "997":
    case "999": {
      const ack = payload.functionalAck ?? payload.implementationAck;
      return ack ? ackReading(ack, t) : null;
    }
  }
}

function blankReading(): Omit<PlainReading, "headline" | "line" | "state"> {
  return { figure: "", facts: [], stops: [], lines: [], faults: [] };
}

function fact(key: string, label: string, value: string | null | undefined): CardFact[] {
  const shown = value?.trim();
  return shown ? [{ key, label, value: shown }] : [];
}

function text(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function amountOf(value: unknown): string | number | null {
  return typeof value === "number" ? value : text(value) || null;
}

function count(value: unknown): number | null {
  return typeof value === "number" && Number.isFinite(value) ? value : null;
}

function placeLine(...parts: (string | null | undefined)[]): string {
  return parts
    .map((part) => part?.trim() ?? "")
    .filter(Boolean)
    .join(", ");
}

function statusReading(
  status: NonNullable<Payload["shipmentStatus"]>,
  t: TranslateFn,
): PlainReading {
  const code = status.statusCode?.trim() ?? "";
  const words = STATUS_WORDS[code];
  const late = status.lateMinutes ?? 0;
  return {
    ...blankReading(),
    headline: words ? words(t) : code ? t("Status {0}", code) : t("Status update"),
    line: placeLine(
      status.locationName,
      placeLine(status.city, [status.stateCode, status.postalCode].filter(Boolean).join(" ")),
    ),
    state: DELAY_CODES.has(code) || late > 0 ? "warn" : code === "D1" ? "ok" : "info",
    figure: cardMoment(status.eventDate),
    facts: [
      ...fact(
        "code",
        t("Status code"),
        code && status.statusReasonCode
          ? t("{0} · reason {1}", code, status.statusReasonCode)
          : code,
      ),
      ...fact(
        "stop",
        t("Stop"),
        status.stopType && status.stopSequence
          ? t("{0}, stop {1}", status.stopType, formatNumber(status.stopSequence))
          : status.stopType,
      ),
      ...fact("pro", t("PRO"), status.proNumber),
      ...fact("bol", t("BOL"), status.bol),
      ...fact("appointment", t("Appointment"), status.appointmentNumber),
      ...fact("equipment", t("Equipment"), status.equipmentNumber),
      ...fact("late", t("Late by"), late > 0 ? waitText(late) : ""),
      ...fact("reason", t("Reason"), status.reasonDescription),
      ...fact("failure", t("Service failure"), status.serviceFailureNumber),
    ],
  };
}

function tenderReading(
  tender: NonNullable<Payload["loadTender"]>,
  payload: Payload,
  t: TranslateFn,
): PlainReading {
  const update = payload.purposeCode === "04";
  const stops = (tender.moves ?? [])
    .flatMap((move) => move.stops ?? [])
    .map((stop, index) => ({
      key: `${stop.locationId}-${index}`,
      kind: stop.type,
      place:
        placeLine(
          stop.locationName ?? stop.locationLabel,
          stop.locationCity,
          stop.locationStateCode,
        ) || t("Unnamed stop"),
      window: cardMoment(stop.scheduledWindowStart),
    }));
  const first = stops[0];
  const last = stops.at(-1);
  return {
    ...blankReading(),
    headline: update ? t("Updated load tender") : t("Load tender"),
    line:
      first && last && first !== last
        ? t("{0} to {1}", first.place, last.place)
        : (first?.place ?? ""),
    state: "info",
    figure:
      amountOf(tender.totalChargeAmount) === null ? "" : money(amountOf(tender.totalChargeAmount)),
    facts: [
      ...fact("bol", t("BOL"), tender.bol),
      ...fact("customer", t("Customer"), tender.customerLabel),
      ...fact("weight", t("Weight"), tender.weight ? t("{0} lb", formatNumber(tender.weight)) : ""),
      ...fact("pieces", t("Pieces"), tender.pieces ? formatNumber(tender.pieces) : ""),
      ...fact("service", t("Service"), tender.serviceTypeLabel),
    ],
    stops,
  };
}

function invoiceReading(invoice: NonNullable<Payload["invoice"]>, t: TranslateFn): PlainReading {
  const currency = text(invoice.currencyCode) || "USD";
  const number = text(invoice.invoiceNumber);
  const charges = Array.isArray(invoice.lineCharges) ? (invoice.lineCharges as unknown[]) : [];
  return {
    ...blankReading(),
    headline: number ? t("Invoice {0}", number) : t("Freight invoice"),
    line: placeLine(
      text(invoice.billToName),
      text(invoice.billToCity),
      text(invoice.billToStateCode),
    ),
    state: "info",
    figure: money(amountOf(invoice.totalAmount), currency),
    facts: [
      ...fact("pro", t("PRO"), text(invoice.proNumber)),
      ...fact("bol", t("BOL"), text(invoice.bol)),
      ...fact("date", t("Invoice date"), cardDate(count(invoice.invoiceDate))),
      ...fact("delivered", t("Delivered"), cardDate(count(invoice.deliveryDate))),
    ],
    lines: charges.flatMap((raw, index) => {
      if (!raw || typeof raw !== "object") return [];
      const charge = raw as Record<string, unknown>;
      return [
        {
          key: `${text(charge.code)}-${index}`,
          label: text(charge.description) || text(charge.code) || t("Charge"),
          amount: money(amountOf(charge.amount), currency),
        },
      ];
    }),
  };
}

function responseReading(
  response: NonNullable<Payload["tenderResponse"]>,
  t: TranslateFn,
): PlainReading {
  const code = text(response.responseCode);
  const accepted = code === "A";
  const declined = code === "D";
  return {
    ...blankReading(),
    headline: accepted
      ? t("Accepted the tender")
      : declined
        ? t("Declined the tender")
        : t("Answered the tender"),
    line: text(response.rejectionReason),
    state: accepted ? "ok" : declined ? "bad" : "info",
    facts: [
      ...fact("bol", t("BOL"), text(response.bol)),
      ...fact("code", t("Response code"), code),
      ...fact("reason", t("Reason code"), text(response.reasonCode)),
    ],
  };
}

function ackReading(ack: NonNullable<Payload["functionalAck"]>, t: TranslateFn): PlainReading {
  const code = text(ack.transactionAcknowledgmentCode) || text(ack.groupAcknowledgmentCode);
  const accepted = ACCEPT_CODES.has(code);
  const partly = code === "P";
  const set = text(ack.originalTransactionSet);
  const control = text(ack.originalTransactionControlNumber);
  const group = text(ack.originalGroupControlNumber);
  const received = count(ack.receivedTransactionSetCount);
  const acceptedCount = count(ack.acceptedTransactionSetCount);
  return {
    ...blankReading(),
    headline:
      code === "E"
        ? t("Accepted with errors")
        : accepted
          ? t("Accepted")
          : partly
            ? t("Partly accepted")
            : code
              ? t("Rejected")
              : t("Acknowledgment"),
    line:
      set && control
        ? group
          ? t("Answers {0} transaction {1} in group {2}", set, control, group)
          : t("Answers {0} transaction {1}", set, control)
        : "",
    state: code === "E" || partly ? "warn" : accepted ? "ok" : code ? "bad" : "info",
    figure:
      received !== null && acceptedCount !== null
        ? t("{0} of {1} accepted", formatNumber(acceptedCount), formatNumber(received))
        : "",
    facts: [...fact("code", t("Acknowledgment code"), code)],
    faults: (ack.diagnostics ?? []).map((diagnostic, index) => ({
      key: `${diagnostic.segmentId ?? ""}-${diagnostic.errorCode ?? ""}-${index}`,
      label:
        diagnostic.segmentId && diagnostic.elementPosition
          ? t("{0}, element {1}", diagnostic.segmentId, formatNumber(diagnostic.elementPosition))
          : (diagnostic.segmentId ?? t("Envelope")),
      amount:
        [
          diagnostic.message?.trim(),
          diagnostic.errorCode ? t("code {0}", diagnostic.errorCode) : "",
        ]
          .filter(Boolean)
          .join(" · ") || t("No detail given"),
    })),
  };
}

/** One element of a segment on the tape: its value and what the X12 dictionary calls it. */
export type TapeElement = { position: number; value: string; label: string; empty: boolean };

/** One segment as the tape draws it, with whatever the validator said about it. */
export type TapeSegment = {
  index: number;
  id: string;
  name: string;
  /** Envelope segments frame the document; the body carries what it says. */
  role: "envelope" | "body";
  elements: TapeElement[];
  malformed: boolean;
  marks: EDIInspectionDiagnostic[];
};

const ENVELOPE_SEGMENTS = new Set(["ISA", "IEA", "GS", "GE", "ST", "SE"]);

/**
 * The inspected X12 as tape rows, envelope segments apart from the body, with
 * each diagnostic pinned to the segment it names: by its index when that
 * segment carries its id, else to the first segment with its id. A diagnostic
 * that names no segment the tape holds is loose and is told above the tape.
 */
export function tapeSegments(inspection: EDIX12Inspection): {
  rows: TapeSegment[];
  loose: EDIInspectionDiagnostic[];
} {
  const rows: TapeSegment[] = inspection.segments.map((segment) => ({
    index: segment.index,
    id: segment.segmentId,
    name: segment.name,
    role: ENVELOPE_SEGMENTS.has(segment.segmentId) ? "envelope" : "body",
    elements: segment.elements.map((element) => ({
      position: element.position,
      value: element.value,
      label: element.label,
      empty: element.empty,
    })),
    malformed: segment.malformed,
    marks: [],
  }));
  const byIndex = new Map(rows.map((row) => [row.index, row]));
  const loose: EDIInspectionDiagnostic[] = [];
  for (const diagnostic of inspection.diagnostics) {
    const segmentId = diagnostic.segmentId?.trim() ?? "";
    const indexed = byIndex.get(diagnostic.segmentIndex);
    const target =
      segmentId && indexed?.id === segmentId
        ? indexed
        : segmentId
          ? rows.find((row) => row.id === segmentId)
          : undefined;
    if (target) {
      target.marks.push(diagnostic);
    } else {
      loose.push(diagnostic);
    }
  }
  return { rows, loose };
}

/** A run of the tape as drawn: a segment, or the segments folded away between two that are shown. */
export type TapeRun =
  | { kind: "segment"; row: TapeSegment }
  | { kind: "fold"; key: string; count: number };

/**
 * The tape folded to its head: the first segments, and past them only the
 * ones the validator marked, each run of hidden segments between folded into
 * a count. Unfolded, every segment shows.
 */
export function foldTape(rows: readonly TapeSegment[], head: number, unfolded: boolean): TapeRun[] {
  if (unfolded || rows.length <= head) {
    return rows.map((row) => ({ kind: "segment", row }));
  }
  const runs: TapeRun[] = [];
  let hidden = 0;
  rows.forEach((row, index) => {
    if (index < head || row.marks.length > 0 || row.malformed) {
      if (hidden > 0) {
        runs.push({ kind: "fold", key: `fold-${index}`, count: hidden });
        hidden = 0;
      }
      runs.push({ kind: "segment", row });
    } else {
      hidden += 1;
    }
  });
  if (hidden > 0) {
    runs.push({ kind: "fold", key: "fold-end", count: hidden });
  }
  return runs;
}

/** How many errors and warnings the validator found across the message. */
export function diagnosticTally(diagnostics: readonly { severity: string }[]): {
  errors: number;
  warnings: number;
} {
  let errors = 0;
  let warnings = 0;
  for (const diagnostic of diagnostics) {
    if (diagnostic.severity === "Error") errors += 1;
    else if (diagnostic.severity === "Warning") warnings += 1;
  }
  return { errors, warnings };
}

/** A record the message names, as the card lists it. */
export type MessageLink = {
  key: "shipment" | "invoice" | "ack" | "source" | "file";
  entity: "shipment" | "invoice" | "edi_message" | "edi_inbound_file";
  id: string;
  title: string;
  line: string;
};

/**
 * The records the message names: the load and invoice it carries, the
 * acknowledgment that answered it, the message an acknowledgment answers, and
 * the inbound file it arrived in. A load or invoice is named by the number
 * the payload gives it.
 */
export function messageLinks(message: EDIMessage, t: TranslateFn): MessageLink[] {
  const payload = message.payloadSnapshot;
  const status = payload?.shipmentStatus;
  const tender = payload?.loadTender ?? payload?.shipment;
  const invoice = payload?.invoice;
  const ack = payload?.functionalAck ?? payload?.implementationAck;
  const links: MessageLink[] = [];

  if (message.shipmentId) {
    const pro = text(status?.proNumber) || text(invoice?.proNumber);
    const bol = text(status?.bol) || text(tender?.bol) || text(invoice?.bol);
    links.push({
      key: "shipment",
      entity: "shipment",
      id: message.shipmentId,
      title: pro || bol || t("Shipment"),
      line: pro && bol ? t("Shipment · BOL {0}", bol) : t("Shipment"),
    });
  }
  if (message.invoiceId) {
    const number = text(invoice?.invoiceNumber);
    links.push({
      key: "invoice",
      entity: "invoice",
      id: message.invoiceId,
      title: number || t("Invoice"),
      line: t("Invoice"),
    });
  }
  if (message.ackMessageId) {
    links.push({
      key: "ack",
      entity: "edi_message",
      id: message.ackMessageId,
      title: t("Its acknowledgment"),
      line: message.ackReceivedAt
        ? t("Received {0}", cardMoment(message.ackReceivedAt))
        : t("EDI message"),
    });
  }
  const source = text(ack?.sourceMessageId);
  if (source) {
    const set = text(ack?.originalTransactionSet);
    const control = text(ack?.originalTransactionControlNumber);
    links.push({
      key: "source",
      entity: "edi_message",
      id: source,
      title: t("The message it answers"),
      line: set && control ? t("{0} · ST {1}", set, control) : t("EDI message"),
    });
  }
  if (message.inboundFileId) {
    links.push({
      key: "file",
      entity: "edi_inbound_file",
      id: message.inboundFileId,
      title: t("The file it arrived in"),
      line: t("Inbound file"),
    });
  }
  return links;
}
