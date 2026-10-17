import { defineLabels } from "@trenova/shared/i18n/labels";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { CardStatusMark, CardTone } from "../../record-card/record-card";
import { money } from "../../record-card/record-card-values";
import type { FieldGroup, FieldRow, ReviewTally, StopGroup } from "./extraction-groups";
import type { ExtractedField, Extraction } from "./extraction-payload";
import type { ShipmentStage } from "./shipment-stage";

/** The mark on the document's stub, the way a dispatcher abbreviates the paper. */
const DOCUMENT_MARKS = defineLabels({
  RateConfirmation: "RC",
  BillOfLading: "BOL",
  ProofOfDelivery: "POD",
  Invoice: "INV",
  LumperReceipt: "LMP",
  ScaleTicket: "SCL",
  DeliveryReceipt: "DR",
});

export function documentMark(kind: string, t: TranslateFn): string {
  return Object.hasOwn(DOCUMENT_MARKS, kind)
    ? DOCUMENT_MARKS[kind as keyof typeof DOCUMENT_MARKS]
    : t("DOC");
}

/** The route's two ends and how many stops it makes, as the head titles the reading. */
export type Lane = { from: string; to: string; stops: number };

function placeOf(stop: StopGroup): string {
  for (const row of stop.rows) {
    if (row.kind === "read" && row.field.stop && row.field.stop.place !== "") {
      return row.field.stop.place;
    }
  }
  return "";
}

/**
 * Where the load goes, from the first stop's city to the last's. Null when fewer than two
 * stops name a place, as on a card saved before stops carried one, so the head falls back to
 * the kind of document.
 */
export function laneOf(groups: readonly FieldGroup[]): Lane | null {
  const stops = groups.find((group) => group.id === "stops")?.stops ?? [];
  const placed = stops.map(placeOf).filter((place) => place !== "");
  if (placed.length < 2) return null;
  return { from: placed[0], to: placed[placed.length - 1], stops: stops.length };
}

function fieldOf(extraction: Pick<Extraction, "fields">, key: string): ExtractedField | null {
  return extraction.fields.find((field) => field.key === key) ?? null;
}

/** A money value as the document wrote it, set as money when it reads as a number. */
export function readMoney(value: string): string {
  const figure = Number(value.replace(/[$,\s]/g, ""));
  return value.trim() !== "" && Number.isFinite(figure) ? money(figure) : value;
}

export type HeadFigure = { amount: string; caption: string; warn: boolean };

/**
 * The figure the head is read for: the rate exactly as the document gave it, with the fuel
 * surcharge named beside it but never added to it, since a document's rate line is as often
 * its total as its linehaul. A rate the reading was looking for and did not find says so; a
 * document that carries no rate, such as a bill of lading, has no figure.
 */
export function headFigure(
  extraction: Pick<Extraction, "fields">,
  tally: Pick<ReviewTally, "missing">,
  t: TranslateFn,
): HeadFigure | null {
  const rate = fieldOf(extraction, "rate");
  const fuel = fieldOf(extraction, "fuelSurcharge");
  if (rate) {
    return {
      amount: readMoney(rate.value),
      caption: fuel ? t("rate as read · fuel {0}", readMoney(fuel.value)) : t("rate as read"),
      warn: rate.needsLook,
    };
  }
  const missing = tally.missing.some((row) => row.missing.key === "rate");
  return missing
    ? { amount: t("Rate not read"), caption: t("check the document"), warn: true }
    : null;
}

/** The line under the title: the load number, who is billed and how many stops. */
export function headSubtitle(
  extraction: Pick<Extraction, "fields">,
  lane: Lane | null,
  t: TranslateFn,
): string {
  const load = fieldOf(extraction, "loadNumber");
  const billTo = fieldOf(extraction, "billTo");
  return [
    load ? t("Load {0}", load.value) : "",
    billTo ? billTo.value : "",
    lane ? t("{0, plural, one {# stop} other {# stops}}", lane.stops) : "",
  ]
    .filter((part) => part !== "")
    .join(" · ");
}

export type HeadStanding = { label: string; tone: CardTone; mark: CardStatusMark };

/**
 * The head's one pill: where the shipment stands once there is one to speak of, and before
 * that how many readings need a person.
 */
export function headStanding(stage: ShipmentStage, checks: number, t: TranslateFn): HeadStanding {
  switch (stage.stage) {
    case "created":
      return { label: t("Shipment created"), tone: "success", mark: "check" };
    case "attached":
      return { label: t("Already a shipment"), tone: "success", mark: "check" };
    case "waiting":
      return { label: t("Drafted, waiting on you"), tone: "progress", mark: "dot" };
    case "creating":
      return { label: t("Creating the shipment"), tone: "progress", mark: "dot" };
    case "drafting":
      return { label: t("Drafting the shipment"), tone: "progress", mark: "dot" };
    default:
      return checks > 0
        ? {
            label: t("{0, plural, one {# to check} other {# to check}}", checks),
            tone: "warn",
            mark: "dot",
          }
        : { label: t("Nothing to check"), tone: "success", mark: "check" };
  }
}

/** Beside the pill: what was not read and what was read clearly. */
export function headFacts(tally: ReviewTally, t: TranslateFn): string[] {
  return [
    tally.missing.length > 0
      ? t("{0, plural, one {# not read} other {# not read}}", tally.missing.length)
      : "",
    tally.read > 0 ? t("{0, plural, one {# read clearly} other {# read clearly}}", tally.read) : "",
  ].filter((fact) => fact !== "");
}

/** The fields a check item stands for, in the order their boxes are drawn on its snippet. */
export function checkFields(rows: readonly FieldRow[]): ExtractedField[] {
  return rows.flatMap((row) => (row.kind === "read" ? [row.field] : []));
}
