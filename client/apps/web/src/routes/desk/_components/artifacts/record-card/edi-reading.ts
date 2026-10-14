import { formatNumber } from "@trenova/shared/i18n/format";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { EDIMessage } from "@trenova/shared/types/edi";
import type { CardCheckState, CardFact } from "./record-card";
import { cardDate, cardMoment, money, waitText } from "./record-card-values";

type Payload = NonNullable<EDIMessage["payloadSnapshot"]>;

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
/** The acknowledgment codes that accept what they answer, errors and all. */
export const ACCEPT_CODES: ReadonlySet<string> = new Set(["A", "E"]);

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

/** A payload value as text, when it is text. */
export function payloadText(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function amountOf(value: unknown): string | number | null {
  return typeof value === "number" ? value : payloadText(value) || null;
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
  const currency = payloadText(invoice.currencyCode) || "USD";
  const number = payloadText(invoice.invoiceNumber);
  const charges = Array.isArray(invoice.lineCharges) ? (invoice.lineCharges as unknown[]) : [];
  return {
    ...blankReading(),
    headline: number ? t("Invoice {0}", number) : t("Freight invoice"),
    line: placeLine(
      payloadText(invoice.billToName),
      payloadText(invoice.billToCity),
      payloadText(invoice.billToStateCode),
    ),
    state: "info",
    figure: money(amountOf(invoice.totalAmount), currency),
    facts: [
      ...fact("pro", t("PRO"), payloadText(invoice.proNumber)),
      ...fact("bol", t("BOL"), payloadText(invoice.bol)),
      ...fact("date", t("Invoice date"), cardDate(count(invoice.invoiceDate))),
      ...fact("delivered", t("Delivered"), cardDate(count(invoice.deliveryDate))),
    ],
    lines: charges.flatMap((raw, index) => {
      if (!raw || typeof raw !== "object") return [];
      const charge = raw as Record<string, unknown>;
      return [
        {
          key: `${payloadText(charge.code)}-${index}`,
          label: payloadText(charge.description) || payloadText(charge.code) || t("Charge"),
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
  const code = payloadText(response.responseCode);
  const accepted = code === "A";
  const declined = code === "D";
  return {
    ...blankReading(),
    headline: accepted
      ? t("Accepted the tender")
      : declined
        ? t("Declined the tender")
        : t("Answered the tender"),
    line: payloadText(response.rejectionReason),
    state: accepted ? "ok" : declined ? "bad" : "info",
    facts: [
      ...fact("bol", t("BOL"), payloadText(response.bol)),
      ...fact("code", t("Response code"), code),
      ...fact("reason", t("Reason code"), payloadText(response.reasonCode)),
    ],
  };
}

function ackReading(ack: NonNullable<Payload["functionalAck"]>, t: TranslateFn): PlainReading {
  const code =
    payloadText(ack.transactionAcknowledgmentCode) || payloadText(ack.groupAcknowledgmentCode);
  const accepted = ACCEPT_CODES.has(code);
  const partly = code === "P";
  const set = payloadText(ack.originalTransactionSet);
  const control = payloadText(ack.originalTransactionControlNumber);
  const group = payloadText(ack.originalGroupControlNumber);
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
