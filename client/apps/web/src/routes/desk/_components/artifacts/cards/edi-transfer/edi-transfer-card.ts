import { MAPPING_ENTITY_LABELS } from "@/lib/edi/mapping-entity-labels";
import { formatNumber } from "@trenova/shared/i18n/format";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type {
  EDIMappingPreview,
  EDIMappingResolution,
  EDIPartner,
  EDITransfer,
} from "@trenova/shared/types/edi";
import { superseded, type TransferDirection } from "../../record-card/edi-transfer-standing";
import type { CardFact } from "../../record-card/record-card";
import { cardMoment, cardMomentAfter, money } from "../../record-card/record-card-values";

export type { TransferDirection };

/**
 * Which side of the tender an organization is on: the one it was tendered to
 * receives it, even when it also sent it, as a partner's 204 read into the
 * same organization is.
 */
export function transferDirection(
  transfer: EDITransfer,
  organizationId: string | null | undefined,
): TransferDirection {
  return transfer.targetOrganizationId === organizationId ? "inbound" : "outbound";
}

/**
 * The other party, as the reader's own organization knows them. Each side of a
 * Trenova connection keeps a partner record for the other: the sender's lives
 * on the source side, the receiver's on the target side. A partner's 204 is
 * one record on both.
 */
export function counterparty(
  transfer: EDITransfer,
  direction: TransferDirection,
): EDIPartner | null {
  return (direction === "inbound" ? transfer.targetPartner : transfer.sourcePartner) ?? null;
}

/** The tender's slip: what it is called, and whether it changes or cancels an earlier one. */
export function tenderSlip(
  transfer: EDITransfer,
  t: TranslateFn,
): { title: string; purpose: string } {
  const payload = transfer.tenderPayload;
  const purpose =
    payload.purposeCode === "04"
      ? t("Change")
      : payload.purposeCode === "01"
        ? t("Cancellation")
        : "";
  return { title: payload.bol?.trim() || t("Load tender"), purpose };
}

/** How far a gate of the tender's decision came. */
export type GateState = "done" | "now" | "warn" | "failed" | "closed" | "ahead" | "skipped";

/** One gate of the tender's way: received, mapped, decided, its shipment built. */
export type DecisionGate = {
  key: "received" | "mapped" | "decided" | "built";
  label: string;
  detail: string;
  state: GateState;
  at: number | null;
};

/** The partner codes the tender names, each against the record it maps to, unmapped first. */
export type MappingRow = {
  key: string;
  entity: string;
  source: string;
  target: string;
  resolved: boolean;
};

export type MappingLedger = { rows: MappingRow[]; resolved: number; total: number };

function mappingRows(
  transfer: EDITransfer,
  preview: EDIMappingPreview | undefined,
): EDIMappingResolution[] {
  return preview?.all ?? transfer.mappingSnapshot;
}

/**
 * What each partner code the tender names maps to: from the live preview when
 * the tender is still open to a decision, else as it stood when the tender
 * came in. Codes nobody has mapped come first.
 */
export function mappingLedger(
  transfer: EDITransfer,
  preview: EDIMappingPreview | undefined,
): MappingLedger {
  const rows = mappingRows(transfer, preview).map((row, index) => ({
    key: `${row.entityType}-${row.sourceId}-${index}`,
    entity: MAPPING_ENTITY_LABELS[row.entityType],
    source: row.sourceLabel?.trim() || row.sourceId,
    target: row.targetLabel?.trim() ?? "",
    resolved: row.resolved,
  }));
  return {
    rows: [...rows.filter((row) => !row.resolved), ...rows.filter((row) => row.resolved)],
    resolved: rows.filter((row) => row.resolved).length,
    total: rows.length,
  };
}

function receivedGate(
  transfer: EDITransfer,
  direction: TransferDirection,
  t: TranslateFn,
): DecisionGate {
  return {
    key: "received",
    label: direction === "inbound" ? t("Received") : t("Sent"),
    detail:
      direction === "outbound"
        ? t("Tendered from shipment")
        : transfer.inboundMessageId
          ? t("Read from an EDI 204")
          : t("Tendered by a Trenova organization"),
    state: "done",
    at: transfer.submittedAt,
  };
}

function mappedGate(transfer: EDITransfer, ledger: MappingLedger, t: TranslateFn): DecisionGate {
  const gate: DecisionGate = {
    key: "mapped",
    label: t("Mapped"),
    detail: "",
    state: "done",
    at: null,
  };
  const counted = t("{0} of {1} mapped", formatNumber(ledger.resolved), formatNumber(ledger.total));
  if (transfer.status === "MappingRequired") {
    return { ...gate, state: "warn", detail: counted };
  }
  if (ledger.total === 0) return { ...gate, detail: t("Nothing to map") };
  if (ledger.resolved === ledger.total) {
    return { ...gate, detail: t("All {0} mapped", formatNumber(ledger.total)) };
  }
  return { ...gate, detail: counted };
}

function decidedGate(
  transfer: EDITransfer,
  direction: TransferDirection,
  t: TranslateFn,
): DecisionGate {
  const gate: DecisionGate = {
    key: "decided",
    label: t("Decided"),
    detail: "",
    state: "ahead",
    at: null,
  };
  switch (transfer.status) {
    case "Submitted":
    case "MappingRequired":
      return gate;
    case "PendingApproval":
      return {
        ...gate,
        state: "now",
        detail:
          direction === "inbound" ? t("Awaiting your decision") : t("Awaiting their decision"),
      };
    case "Processing":
    case "Approved":
    case "Failed":
      return { ...gate, state: "done", detail: t("Accepted"), at: transfer.approvedAt ?? null };
    case "Rejected":
      return { ...gate, state: "closed", detail: t("Declined"), at: transfer.rejectedAt ?? null };
    case "Canceled":
      return {
        ...gate,
        state: "closed",
        detail: superseded(transfer) ? t("Replaced by a change") : t("Canceled"),
        at: transfer.canceledAt ?? transfer.processedAt ?? null,
      };
    case "Expired":
      return {
        ...gate,
        state: "closed",
        detail: t("Expired before a decision"),
        at: transfer.processedAt ?? null,
      };
  }
}

function builtGate(transfer: EDITransfer, t: TranslateFn): DecisionGate {
  const gate: DecisionGate = {
    key: "built",
    label: t("Shipment"),
    detail: "",
    state: "ahead",
    at: null,
  };
  switch (transfer.status) {
    case "Approved":
      return {
        ...gate,
        state: "done",
        detail: t("Shipment created"),
        at: transfer.processedAt ?? null,
      };
    case "Processing":
      return {
        ...gate,
        state: "now",
        detail: t("Creating the shipment"),
        at: transfer.processingStartedAt ?? null,
      };
    case "Failed":
      return {
        ...gate,
        state: "failed",
        detail: t("Could not be created"),
        at: transfer.processedAt ?? null,
      };
    case "Rejected":
    case "Canceled":
    case "Expired":
      return { ...gate, state: "skipped" };
    case "Submitted":
    case "MappingRequired":
    case "PendingApproval":
      return gate;
  }
}

/**
 * The tender's way as four gates: it came in or went out, its partner codes
 * were mapped, someone decided on it, and its shipment was built. A tender
 * declined, replaced or expired is closed at the decision; one accepted whose
 * shipment could not be built fails at the last gate.
 */
export function decisionGates(
  transfer: EDITransfer,
  direction: TransferDirection,
  t: TranslateFn,
  preview?: EDIMappingPreview,
): DecisionGate[] {
  return [
    receivedGate(transfer, direction, t),
    mappedGate(transfer, mappingLedger(transfer, preview), t),
    decidedGate(transfer, direction, t),
    builtGate(transfer, t),
  ];
}

/** Why the tender closed where it did, in the words its side wrote down. */
export type ReasonNote = { tone: "bad" | "muted"; label: string; text: string };

export function reasonNote(transfer: EDITransfer, t: TranslateFn): ReasonNote | null {
  const rejection = transfer.rejectionReason?.trim() ?? "";
  const failure = transfer.failureReason?.trim() ?? "";
  if (transfer.status === "Rejected" && rejection) {
    return { tone: "muted", label: t("Why it was declined"), text: rejection };
  }
  if (transfer.status === "Failed" && failure) {
    return { tone: "bad", label: t("Why the shipment was not created"), text: failure };
  }
  if (transfer.status === "Canceled" && failure) {
    return { tone: "muted", label: t("Why it was canceled"), text: failure };
  }
  return null;
}

function text(value: unknown): string | null {
  if (typeof value === "number") return String(value);
  return typeof value === "string" && value.trim() !== "" ? value.trim() : null;
}

function loadLine(
  pieces: number | null | undefined,
  weight: number | null | undefined,
  t: TranslateFn,
): string {
  return [
    pieces ? t("{0, plural, one {# piece} other {# pieces}}", pieces) : "",
    weight ? t("{0} lb", formatNumber(weight)) : "",
  ]
    .filter(Boolean)
    .join(" · ");
}

/** Whether the location a stop names is mapped to one of ours. */
export type StopMapping = { state: "mapped" | "missing"; label: string };

export type RouteStop = {
  key: string;
  /** Whether the stop takes freight on or puts it off. */
  role: "pick" | "drop";
  kind: string;
  place: string;
  line: string;
  window: string;
  load: string;
  mapping: StopMapping | null;
};

function stopKind(type: string, t: TranslateFn): string {
  switch (type) {
    case "Pickup":
      return t("Pickup");
    case "Delivery":
      return t("Delivery");
    case "SplitPickup":
      return t("Split pickup");
    case "SplitDelivery":
      return t("Split delivery");
    default:
      return type;
  }
}

/**
 * The tender's stops in order across its moves: what each is, where, its
 * window, what it loads or unloads, and whether the partner's location code is
 * mapped to one of ours.
 */
export function tenderRoute(
  transfer: EDITransfer,
  preview: EDIMappingPreview | undefined,
  t: TranslateFn,
): { stops: RouteStop[]; distance: string } {
  const locations = mappingRows(transfer, preview).filter((row) => row.entityType === "Location");
  const moves = transfer.tenderPayload.moves ?? [];
  const stops = moves.flatMap((move, moveIndex) =>
    (move.stops ?? []).map((stop) => {
      const mapped = locations.find((row) => row.sourceId === stop.locationId);
      const end = stop.scheduledWindowEnd ?? null;
      return {
        key: `${moveIndex}-${stop.sequence}`,
        role: stop.type.endsWith("Delivery") ? ("drop" as const) : ("pick" as const),
        kind: stopKind(stop.type, t),
        place:
          stop.locationLabel?.trim() ||
          stop.locationName?.trim() ||
          stop.locationCode?.trim() ||
          t("Unnamed stop"),
        line: [
          stop.locationAddressLine1,
          stop.locationCity,
          [stop.locationStateCode, stop.locationPostalCode].filter(Boolean).join(" "),
        ]
          .filter(Boolean)
          .join(", "),
        window:
          stop.scheduledWindowStart && end
            ? `${cardMoment(stop.scheduledWindowStart)} – ${cardMomentAfter(stop.scheduledWindowStart, end)}`
            : cardMoment(stop.scheduledWindowStart),
        load: loadLine(stop.pieces, stop.weight, t),
        mapping: mapped
          ? mapped.resolved
            ? {
                state: "mapped" as const,
                label: mapped.targetLabel?.trim() || mapped.sourceLabel?.trim() || mapped.sourceId,
              }
            : { state: "missing" as const, label: t("Not mapped") }
          : null,
      };
    }),
  );
  const miles = moves.reduce((sum, move) => sum + (move.distance ?? 0), 0);
  return { stops, distance: miles > 0 ? t("{0} mi", formatNumber(Math.round(miles))) : "" };
}

export type FreightLine = { key: string; label: string; load: string };

/** What the tender moves and for whom: its customer, service, weight and temperature, and its commodities. */
export function tenderFreight(
  transfer: EDITransfer,
  t: TranslateFn,
): { facts: CardFact[]; commodities: FreightLine[] } {
  const payload = transfer.tenderPayload;
  const min = payload.temperatureMin;
  const max = payload.temperatureMax;
  const temperature =
    min !== null && min !== undefined && max !== null && max !== undefined
      ? t("{0}–{1} °F", formatNumber(min), formatNumber(max))
      : (min ?? max) !== null && (min ?? max) !== undefined
        ? t("{0} °F", formatNumber((min ?? max) as number))
        : "";
  const facts: [string, string, string | null | undefined][] = [
    ["customer", t("Customer"), payload.customerLabel],
    ["service", t("Service"), payload.serviceTypeLabel],
    ["shipmentType", t("Shipment type"), payload.shipmentTypeLabel],
    ["weight", t("Weight"), payload.weight ? t("{0} lb", formatNumber(payload.weight)) : ""],
    ["pieces", t("Pieces"), payload.pieces ? formatNumber(payload.pieces) : ""],
    ["temperature", t("Temperature"), temperature],
  ];
  return {
    facts: facts.flatMap(([key, label, value]) =>
      value?.trim() ? [{ key, label, value: value.trim() }] : [],
    ),
    commodities: (payload.commodities ?? []).map((commodity, index) => ({
      key: `${commodity.commodityId}-${index}`,
      label:
        commodity.commodityLabel?.trim() ||
        commodity.commodityName?.trim() ||
        commodity.commodityDescription?.trim() ||
        t("Commodity"),
      load: loadLine(commodity.pieces, commodity.weight, t),
    })),
  };
}

export type ChargeLine = { key: string; label: string; amount: string };

/** The tender's rate as a ledger: the freight, each accessorial or the other charges, and the total. */
export function tenderCharges(
  transfer: EDITransfer,
  t: TranslateFn,
): { lines: ChargeLine[]; total: string } {
  const payload = transfer.tenderPayload;
  const freight = text(payload.freightChargeAmount);
  const other = text(payload.otherChargeAmount);
  const total = text(payload.totalChargeAmount);
  const accessorials = (payload.additionalCharges ?? []).map((charge, index) => ({
    key: `${charge.accessorialCode || charge.accessorialChargeId}-${index}`,
    label:
      charge.accessorialLabel?.trim() ||
      charge.accessorialDescription?.trim() ||
      charge.accessorialCode?.trim() ||
      t("Accessorial charge"),
    amount: money(text(charge.amount)),
  }));
  return {
    lines: [
      ...(freight ? [{ key: "freight", label: t("Freight"), amount: money(freight) }] : []),
      ...accessorials,
      ...(accessorials.length === 0 && other
        ? [{ key: "other", label: t("Other charges"), amount: money(other) }]
        : []),
    ],
    total: total ? money(total) : "",
  };
}

/** A record the tender names that opens in the card's place. */
export type TransferLink = {
  key: "message" | "source" | "built";
  entity: "edi_message" | "shipment";
  id: string;
  title: string;
  line: string;
};

/**
 * The records on the reader's side the tender names: the 204 it was read from,
 * the shipment a tender we sent offers, and the shipment an accepted tender
 * built. A shipment on the other organization's side is not ours to open.
 */
export function transferLinks(
  transfer: EDITransfer,
  direction: TransferDirection,
  t: TranslateFn,
): TransferLink[] {
  const links: TransferLink[] = [];
  if (transfer.inboundMessageId) {
    links.push({
      key: "message",
      entity: "edi_message",
      id: transfer.inboundMessageId,
      title: t("Load tender 204"),
      line: t("The EDI message it was read from"),
    });
  }
  if (direction === "outbound" && transfer.sourceShipmentId) {
    links.push({
      key: "source",
      entity: "shipment",
      id: transfer.sourceShipmentId,
      title: t("Shipment tendered"),
      line: t("Our shipment this tender offers"),
    });
  }
  if (direction === "inbound" && transfer.targetShipmentId) {
    links.push({
      key: "built",
      entity: "shipment",
      id: transfer.targetShipmentId,
      title: t("Shipment it built"),
      line: t("Created when the tender was accepted"),
    });
  }
  return links;
}
