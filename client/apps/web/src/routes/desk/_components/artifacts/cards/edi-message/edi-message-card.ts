import { formatNumber } from "@trenova/shared/i18n/format";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type {
  EDIInspectionDiagnostic,
  EDIMessage,
  EDIX12Inspection,
} from "@trenova/shared/types/edi";
import type { CardCheckState, CardTone } from "../../record-card/record-card";
import { ACCEPT_CODES, payloadText } from "../../record-card/edi-reading";
import { cardMoment } from "../../record-card/record-card-values";

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
    ? payloadText(ack.transactionAcknowledgmentCode) || payloadText(ack.groupAcknowledgmentCode)
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

/**
 * Why the message's way stopped where it did: the stop it failed at, what to
 * call the reason, the attempt it gave up on and when, and the words the
 * connection or the partner's system sent back.
 */
export type WayHalt = {
  station: JourneyStation["key"];
  label: string;
  attempt: number | null;
  at: number | null;
  text: string;
};

/**
 * What stopped an outbound message, when something did and said why: a
 * delivery that failed or was dead-lettered, or an acknowledgment that
 * rejected it or could not be read. A message still on its way, or one whose
 * failure left no words, has nothing to say here.
 */
export function wayHalt(message: EDIMessage, t: TranslateFn): WayHalt | null {
  const delivery = message.deliveryLastError?.trim() ?? "";
  if (delivery && message.deliveryStatus === "DeadLettered") {
    return deliveryHalt(message, t("Why delivery gave up"), delivery);
  }
  if (delivery && message.deliveryStatus === "Failed") {
    return deliveryHalt(message, t("Why the last attempt failed"), delivery);
  }
  const answer = message.ackLastError?.trim() ?? "";
  if (answer && message.ackStatus === "Rejected") {
    return answerHalt(message, t("Why the partner rejected it"), answer);
  }
  if (answer && message.ackStatus === "Failed") {
    return answerHalt(message, t("Why its acknowledgment could not be read"), answer);
  }
  return null;
}

function deliveryHalt(message: EDIMessage, label: string, text: string): WayHalt {
  return {
    station: "sent",
    label,
    attempt: message.deliveryAttempts > 0 ? message.deliveryAttempts : null,
    at: message.deliveryLastAttemptAt ?? null,
    text,
  };
}

function answerHalt(message: EDIMessage, label: string, text: string): WayHalt {
  return { station: "answered", label, attempt: null, at: message.ackReceivedAt ?? null, text };
}

/**
 * Where along the way a stop's node sits, for pointing at it: the first stop
 * rests on the start edge, the last on the end edge, and any between them at
 * the middle of its column, as the share of the way's width that middle is.
 */
export type WayAnchor = { edge: "start" | "end" } | { share: number };

export function wayAnchor(index: number, count: number): WayAnchor | null {
  if (index < 0 || index >= count) {
    return null;
  }
  if (index === 0) {
    return { edge: "start" };
  }
  if (index === count - 1) {
    return { edge: "end" };
  }
  return { share: (index + 0.5) / count };
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
    const pro = payloadText(status?.proNumber) || payloadText(invoice?.proNumber);
    const bol = payloadText(status?.bol) || payloadText(tender?.bol) || payloadText(invoice?.bol);
    links.push({
      key: "shipment",
      entity: "shipment",
      id: message.shipmentId,
      title: pro || bol || t("Shipment"),
      line: pro && bol ? t("Shipment · BOL {0}", bol) : t("Shipment"),
    });
  }
  if (message.invoiceId) {
    const number = payloadText(invoice?.invoiceNumber);
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
  const source = payloadText(ack?.sourceMessageId);
  if (source) {
    const set = payloadText(ack?.originalTransactionSet);
    const control = payloadText(ack?.originalTransactionControlNumber);
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
