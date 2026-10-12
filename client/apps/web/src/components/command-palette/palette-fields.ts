import { defineLabels } from "@trenova/shared/i18n/labels";
import { matchFields, type MatchField } from "@trenova/shared/lib/fuzzy-score";
import type { PaletteCommand, PaletteMatch, PalettePage, PaletteRecord } from "./palette-model";

/**
 * The fields each kind of row is matched on, first the title, each weighted
 * by how much a hit there says the row is the one meant. A field the row does
 * not draw carries a `via` label, so a match through it is shown as its own
 * tag rather than lighting up text that is not on screen.
 */
export type RecordFieldKey =
  | "title"
  | "customer"
  | "bol"
  | "lane"
  | "driver"
  | "code"
  | "city"
  | "id"
  | "role"
  | "home"
  | "record"
  | "type";

export type PageFieldKey = "title" | "keyword" | "trail";
export type CommandFieldKey = "title" | "keyword" | "description";

export const TRAIL_SEPARATOR = " › ";

export const WORKER_TYPE_LABELS = defineLabels({
  Employee: "Employee",
  Contractor: "Contractor",
});

export const DRIVER_TYPE_LABELS = defineLabels({
  Local: "Local",
  Regional: "Regional",
  OTR: "Over the road",
  Team: "Team",
});

/** The words a match-source tag opens with, keyed by a field's `via`. */
export const MATCH_SOURCE_LABELS = defineLabels({
  Driver: "Driver",
  Keyword: "Keyword",
});

export const DOCUMENT_KIND_LABELS = defineLabels({
  RateConfirmation: "Rate confirmation",
  BillOfLading: "Bill of lading",
  ProofOfDelivery: "Proof of delivery",
  Invoice: "Invoice",
  LumperReceipt: "Lumper receipt",
  ScaleTicket: "Scale ticket",
  DeliveryReceipt: "Delivery receipt",
});

function labelFrom<T extends Readonly<Record<string, string>>>(
  labels: T,
  value: string | undefined,
): string {
  if (!value) {
    return "";
  }
  return Object.hasOwn(labels, value) ? labels[value as keyof T] : "";
}

/** "Dallas, TX → Atlanta, GA", or whichever end is known. */
export function shipmentLane(record: PaletteRecord): string {
  const origin = record.metadata.origin ?? "";
  const destination = record.metadata.destination ?? "";
  if (origin && destination) {
    return `${origin} → ${destination}`;
  }
  return origin || destination;
}

export function shipmentBol(record: PaletteRecord): string {
  const bol = record.metadata.bol ?? "";
  // i18n-ignore: "BOL 88213" is the token people type to find a load; the row redraws this exact text so the matched ranges line up
  return bol ? `BOL ${bol}` : "";
}

/** "Employee · Over the road" from a worker's type and driver type; an unknown value is left out. */
export function workerRoleText(
  workerType: string | undefined,
  driverType: string | undefined,
): string {
  return [labelFrom(WORKER_TYPE_LABELS, workerType), labelFrom(DRIVER_TYPE_LABELS, driverType)]
    .filter(Boolean)
    .join(" · ");
}

export function workerRole(record: PaletteRecord): string {
  return workerRoleText(record.metadata.workerType, record.metadata.driverType);
}

export function documentKind(record: PaletteRecord): string {
  return labelFrom(DOCUMENT_KIND_LABELS, record.metadata.detectedKind);
}

export function recordFields(record: PaletteRecord): MatchField<RecordFieldKey>[] {
  const meta = record.metadata;
  switch (record.entityType) {
    case "shipment":
      return [
        { key: "title", text: record.title },
        { key: "customer", text: meta.customerName ?? "", weight: 0.92 },
        { key: "bol", text: shipmentBol(record), weight: 0.85 },
        { key: "lane", text: shipmentLane(record), weight: 0.75 },
        { key: "driver", text: meta.driverName ?? "", weight: 0.6, via: "Driver" },
      ];
    case "customer":
      return [
        { key: "title", text: record.title },
        { key: "code", text: meta.code ?? record.subtitle ?? "", weight: 0.95 },
        { key: "city", text: meta.city ?? "", weight: 0.6 },
      ];
    case "worker":
      return [
        { key: "title", text: record.title },
        { key: "id", text: meta.externalId ?? "", weight: 0.9 },
        { key: "role", text: workerRole(record), weight: 0.5 },
        { key: "home", text: meta.city ?? "", weight: 0.5 },
      ];
    case "document":
      return [
        { key: "title", text: record.title },
        { key: "record", text: meta.attachedLabel ?? "", weight: 0.8 },
        { key: "type", text: documentKind(record), weight: 0.75 },
      ];
  }
}

export function pageFields(
  page: PalettePage,
  translate: (text: string) => string,
): MatchField<PageFieldKey>[] {
  return [
    { key: "title", text: translate(page.title) },
    ...page.keywords.map((keyword): MatchField<PageFieldKey> => ({
      key: "keyword",
      text: keyword,
      weight: 0.82,
      via: "Keyword",
    })),
    {
      key: "trail",
      text: page.trail.map((segment) => translate(segment)).join(TRAIL_SEPARATOR),
      weight: 0.62,
    },
  ];
}

export function commandFields(command: PaletteCommand): MatchField<CommandFieldKey>[] {
  return [
    { key: "title", text: command.label },
    ...command.keywords.map((keyword): MatchField<CommandFieldKey> => ({
      key: "keyword",
      text: keyword,
      weight: 0.8,
      via: "Keyword",
    })),
    { key: "description", text: command.description, weight: 0.45 },
  ];
}

export function matchRecord(query: string, record: PaletteRecord): PaletteMatch | null {
  return matchFields(query, recordFields(record));
}
