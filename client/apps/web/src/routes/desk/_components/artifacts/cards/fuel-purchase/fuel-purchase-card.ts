import type { DeskFuelPurchase, DeskFuelPurchaseCard } from "@/lib/graphql/fuel-desk-cards";
import { computeFuelTotal } from "@/lib/fuel-purchase";
import { defineLabels } from "@trenova/shared/i18n/labels";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { referenceKindAccessLabel } from "@trenova/shared/lib/reference-access";
import { parseAmountToMinor } from "@trenova/shared/lib/money";
import {
  FUEL_CARD_PROVIDER_LABELS,
  IFTA_FUEL_TYPE_LABELS,
  type IftaFuelType,
} from "@trenova/shared/types/fuel-ifta-enums";
import type { CardCheckState, CardStatusMark, CardTone } from "../../record-card/record-card";
import {
  cardDay,
  cardTime,
  cardWeekday,
  minorMoney,
  money,
} from "../../record-card/record-card-values";

/** The mark on a purchase's stub: the fuel, the way a pump abbreviates it. */
const FUEL_MARKS: Record<IftaFuelType, string> = defineLabels({
  Diesel: "DSL",
  Gasoline: "GAS",
  Gasohol: "GSH",
  Propane: "LPG",
  CNG: "CNG",
  LNG: "LNG",
  Ethanol: "ETH",
  Methanol: "MTH",
  E85: "E85",
  M85: "M85",
  A55: "A55",
  Biodiesel: "BIO",
  Electricity: "EV",
  Hydrogen: "H2",
  DEF: "DEF",
  Reefer: "RFR",
  Other: "FUEL",
});

export function fuelMark(fuelType: IftaFuelType): string {
  return FUEL_MARKS[fuelType];
}

/** Where the fuel was bought: the vendor's city and the jurisdiction it counts in. */
export function purchasePlace(purchase: DeskFuelPurchase, t: TranslateFn): string {
  const city = purchase.vendorCity?.trim() ?? "";
  const code = purchase.jurisdiction.code;
  return city ? t("{0}, {1}", city, code) : code;
}

export function purchaseTitle(purchase: DeskFuelPurchase, t: TranslateFn): string {
  return purchase.vendor?.trim() || t("Fuel purchase");
}

/** A quantity as the receipt reads it, with its unit. */
export function quantityText(purchase: DeskFuelPurchase, t: TranslateFn): string {
  return purchase.quantityUnit === "Litre"
    ? t("{0} L", purchase.quantity)
    : t("{0} gal", purchase.quantity);
}

export function priceText(purchase: DeskFuelPurchase): string {
  if (!purchase.unitPrice) return "";
  return formatUnitPrice(purchase.unitPrice, purchase.currencyCode);
}

/** A price per unit to the four places the pump prints it. */
export function formatUnitPrice(price: string, currency: string): string {
  const figure = Number(price);
  if (!Number.isFinite(figure)) return price;
  return new Intl.NumberFormat(undefined, {
    style: "currency",
    currency: currency || "USD",
    minimumFractionDigits: 4,
    maximumFractionDigits: 4,
  }).format(figure);
}

export function totalText(purchase: DeskFuelPurchase): string {
  return money(purchase.totalAmount, purchase.currencyCode);
}

export function quarterLabel(year: number, quarter: number, t: TranslateFn): string {
  return t("Q{0} {1}", quarter, year);
}

/** How far a total is from quantity × price, in minor units; null when there is no price to check. */
export function totalDifference(purchase: DeskFuelPurchase): {
  computed: string;
  differenceMinor: number;
} | null {
  const computed = computeFuelTotal(purchase.quantity, purchase.unitPrice);
  if (computed === null) return null;
  const total = parseAmountToMinor(purchase.totalAmount);
  const expected = parseAmountToMinor(computed);
  if (total === null || expected === null) return null;
  return { computed, differenceMinor: total - expected };
}

export type PurchaseCheck = {
  key: string;
  state: CardCheckState;
  title: string;
  detail: string;
};

function totalCheck(purchase: DeskFuelPurchase, t: TranslateFn): PurchaseCheck | null {
  const difference = totalDifference(purchase);
  if (!difference) return null;
  const formula = t("{0} × {1}", quantityText(purchase, t), priceText(purchase));
  if (difference.differenceMinor === 0) {
    return {
      key: "total",
      state: "ok",
      title: t("Total matches quantity × price"),
      detail: t("{0} = {1}", formula, totalText(purchase)),
    };
  }
  const gap = minorMoney(Math.abs(difference.differenceMinor), purchase.currencyCode);
  const computed = money(difference.computed, purchase.currencyCode);
  return {
    key: "total",
    state: "warn",
    title: t("Total differs from quantity × price"),
    detail:
      difference.differenceMinor > 0
        ? t("{0} = {1}, {2} less than the {3} entered", formula, computed, gap, totalText(purchase))
        : t(
            "{0} = {1}, {2} more than the {3} entered",
            formula,
            computed,
            gap,
            totalText(purchase),
          ),
  };
}

/** Miles since the previous fill and what this fill puts the truck's mileage at. */
export function milesSincePrevious(card: DeskFuelPurchaseCard): {
  miles: number;
  mpg: number | null;
} | null {
  const odometer = card.purchase.odometer;
  const previous = card.previousFill?.odometer;
  if (odometer === null || previous === null || previous === undefined) return null;
  const miles = odometer - previous;
  const gallons = Number(card.purchase.gallons);
  const mpg = miles > 0 && gallons > 0 ? Math.round((miles / gallons) * 10) / 10 : null;
  return { miles, mpg };
}

function odometerCheck(card: DeskFuelPurchaseCard, t: TranslateFn): PurchaseCheck | null {
  const { purchase, previousFill } = card;
  if (!card.ifta.countsForIfta) return null;
  if (purchase.odometer === null) {
    return {
      key: "odometer",
      state: "warn",
      title: t("No odometer"),
      detail: purchase.tractor?.code
        ? t(
            "Without one, {0}'s miles per gallon for this fill can't be worked out",
            purchase.tractor.code,
          )
        : t("Without one, the miles per gallon for this fill can't be worked out"),
    };
  }
  const since = milesSincePrevious(card);
  if (!since || !previousFill) {
    return {
      key: "odometer",
      state: "info",
      title: t("First fill with an odometer"),
      detail: t("No earlier fill for this tractor recorded one to measure from"),
    };
  }
  const vendor = previousFill.vendor?.trim() ?? "";
  const city = previousFill.vendorCity?.trim() ?? "";
  const where = vendor && city ? t("{0}, {1}", vendor, city) : vendor || city;
  const day = cardDay(previousFill.purchasedAt);
  if (since.miles <= 0) {
    const now = purchase.odometer.toLocaleString();
    const before = (previousFill.odometer ?? 0).toLocaleString();
    return {
      key: "odometer",
      state: "warn",
      title: t("Odometer didn't move on"),
      detail: where
        ? t("{0} mi is no further than the {1} mi recorded at {2} on {3}", now, before, where, day)
        : t("{0} mi is no further than the {1} mi recorded on {2}", now, before, day),
    };
  }
  const miles = since.miles.toLocaleString();
  const travelled = where
    ? t("+{0} mi since {1} on {2}", miles, where, day)
    : t("+{0} mi since the fill on {1}", miles, day);
  return {
    key: "odometer",
    state: "ok",
    title: t("Odometer moved on"),
    detail: since.mpg === null ? travelled : t("{0} · {1} mpg", travelled, since.mpg.toFixed(1)),
  };
}

function cardCheck(
  purchase: DeskFuelPurchase,
  canReadFuelCards: boolean,
  t: TranslateFn,
): PurchaseCheck {
  const card = purchase.fuelCard;
  if (!card) {
    return {
      key: "card",
      state: "info",
      title: t("No card"),
      detail:
        purchase.source === "Manual"
          ? t("Entered by hand without a fuel card")
          : t("The statement named no card on file"),
    };
  }
  if (card.status === "Cancelled") {
    return {
      key: "card",
      state: "warn",
      title: t("Card was cancelled"),
      detail: t("{0} was cancelled before this purchase was recorded", card.label),
    };
  }
  if (!canReadFuelCards) {
    return {
      key: "card",
      state: "info",
      title: t("Card assignment is hidden"),
      detail: t("You don't have access to {0}", referenceKindAccessLabel("fuel_card")),
    };
  }
  if (card.assignedTractorId && card.assignedTractorId !== purchase.tractorId) {
    return {
      key: "card",
      state: "warn",
      title: t("Card belongs to another tractor"),
      detail: card.assignedTractor?.code
        ? t("{0} is assigned to {1}", card.label, card.assignedTractor.code)
        : t("{0} is assigned to another tractor", card.label),
    };
  }
  if (!card.assignedTractorId && !card.assignedWorkerId) {
    return {
      key: "card",
      state: "info",
      title: t("Card isn't assigned"),
      detail: t("{0} has no tractor or driver on it", card.label),
    };
  }
  const holder = card.assignedWorker?.wholeName?.trim() ?? "";
  const unit = card.assignedTractorId ? (purchase.tractor?.code ?? "") : "";
  return {
    key: "card",
    state: "ok",
    title: t("Card belongs with this unit"),
    detail:
      unit && holder
        ? t("{0} is assigned to {1} and {2}", card.label, unit, holder)
        : unit
          ? t("{0} is assigned to {1}", card.label, unit)
          : t("{0} is carried by {1}", card.label, holder),
  };
}

/**
 * What the card checks about the purchase, in the order a person would look. The fuel card's
 * assignment is outside its reference view, so without fuel card read (`canReadFuelCards`
 * false) the card check says it cannot be made rather than reading the card as unassigned.
 */
export function purchaseChecks(
  card: DeskFuelPurchaseCard,
  t: TranslateFn,
  canReadFuelCards = true,
): PurchaseCheck[] {
  return [
    totalCheck(card.purchase, t),
    odometerCheck(card, t),
    cardCheck(card.purchase, canReadFuelCards, t),
  ].filter(
    (check): check is PurchaseCheck => check !== null,
  );
}

export type PurchaseStanding = { label: string; tone: CardTone; mark: CardStatusMark };

/**
 * The head's pill: what to check when something is off, else what the purchase
 * is to IFTA.
 */
export function purchaseStanding(
  card: DeskFuelPurchaseCard,
  checks: readonly PurchaseCheck[],
  t: TranslateFn,
): PurchaseStanding {
  const off = checks.filter((check) => check.state === "warn" || check.state === "bad").length;
  if (off > 0) {
    return {
      label: t("{0, plural, one {# to check} other {# to check}}", off),
      tone: "warn",
      mark: "dot",
    };
  }
  const code = card.purchase.jurisdiction.code;
  if (!card.ifta.countsForIfta) {
    return { label: t("Not IFTA fuel"), tone: "muted", mark: "dot" };
  }
  return card.purchase.taxPaid
    ? { label: t("Tax paid in {0}", code), tone: "success", mark: "check" }
    : { label: t("Untaxed in {0}", code), tone: "muted", mark: "dot" };
}

export type FiledStep = {
  key: string;
  kind: "feed" | "upload" | "hand" | "note";
  title: string;
  detail: string;
};

/** How the purchase came to be on file: the statement row it was read from and each match, or who entered it. */
export function filedSteps(card: DeskFuelPurchaseCard, t: TranslateFn): FiledStep[] {
  const { purchase, provenance, createdBy } = card;
  if (purchase.source === "Manual") {
    return [
      {
        key: "hand",
        kind: "hand",
        title: createdBy ? t("Entered by {0}", createdBy.name) : t("Entered by hand"),
        detail: t(
          "On {0}, with the tractor, driver and place chosen by hand",
          cardDay(purchase.createdAt),
        ),
      },
    ];
  }
  if (!provenance) return [];
  const provider = FUEL_CARD_PROVIDER_LABELS[provenance.batch.provider];
  const read: FiledStep = {
    key: "read",
    kind: provenance.batch.origin === "Feed" ? "feed" : "upload",
    title:
      provenance.batch.origin === "Feed"
        ? t("Read from the {0} transaction file", provider)
        : provenance.batch.fileName
          ? t("Imported from {0}", provenance.batch.fileName)
          : t("Imported from a {0} statement", provider),
    detail: t(
      "Row {0} of {1} · {2}",
      provenance.rowNumber,
      provenance.batch.rowCount,
      cardDay(provenance.batch.createdAt),
    ),
  };
  const notes = provenance.resolutionNotes.map((note, index): FiledStep => ({
    key: `note-${index}`,
    kind: "note",
    title: note,
    detail: "",
  }));
  return [read, ...notes];
}

export type SlipLine = {
  key: string;
  label: string;
  value: string;
  /** How the line reads against the rest: worked out from the others, off from them, or missing. */
  mark?: "derived" | "off" | "missing";
};

/** The receipt's lines as the purchase records them, with what the card worked out beside them. */
export function receiptLines(purchase: DeskFuelPurchase, t: TranslateFn): SlipLine[] {
  const lines: SlipLine[] = [
    { key: "product", label: t("Product"), value: IFTA_FUEL_TYPE_LABELS[purchase.fuelType] },
    { key: "quantity", label: t("Quantity"), value: quantityText(purchase, t) },
  ];
  if (purchase.quantityUnit === "Litre") {
    lines.push({
      key: "gallons",
      label: "",
      value: t("= {0} gal", purchase.gallons),
      mark: "derived",
    });
  }
  if (purchase.unitPrice) {
    lines.push({
      key: "price",
      label: purchase.quantityUnit === "Litre" ? t("Price / L") : t("Price / gal"),
      value: priceText(purchase),
    });
  }
  const difference = totalDifference(purchase);
  if (difference && difference.differenceMinor !== 0) {
    lines.push({
      key: "computed",
      label: t("Qty × price"),
      value: money(difference.computed, purchase.currencyCode),
      mark: "off",
    });
  }
  lines.push(
    purchase.odometer === null
      ? { key: "odometer", label: t("Odometer"), value: t("Not recorded"), mark: "missing" }
      : {
          key: "odometer",
          label: t("Odometer"),
          value: t("{0} mi", purchase.odometer.toLocaleString()),
        },
  );
  return lines;
}

/** The card the purchase was made on, as the foot of a receipt prints it. */
export function receiptCard(purchase: DeskFuelPurchase, t: TranslateFn): string {
  if (purchase.fuelCard) {
    return t(
      "{0} •••• {1}",
      FUEL_CARD_PROVIDER_LABELS[purchase.fuelCard.provider],
      purchase.fuelCard.lastFour,
    );
  }
  return purchase.cardLastFour ? t("•••• {0}", purchase.cardLastFour) : "";
}

export type IftaBadge = { label: string; tone: "success" | "warning" | "neutral" };

/** What the purchase is to the quarter's return, in a word. */
export function iftaBadge(card: DeskFuelPurchaseCard, t: TranslateFn): IftaBadge {
  if (!card.ifta.countsForIfta) return { label: t("Not IFTA fuel"), tone: "neutral" };
  return card.purchase.taxPaid
    ? { label: t("Counts"), tone: "success" }
    : { label: t("Untaxed"), tone: "warning" };
}

export type IftaLine = { key: string; label: string; value: string; note?: string; zero?: boolean };

function gallonFigure(gallons: string): string {
  const figure = Number(gallons);
  return Number.isFinite(figure)
    ? figure.toLocaleString(undefined, { minimumFractionDigits: 3, maximumFractionDigits: 3 })
    : gallons;
}

/** What the purchase puts on the tractor's return for its jurisdiction and quarter. */
export function iftaLines(card: DeskFuelPurchaseCard, t: TranslateFn): IftaLine[] {
  const { purchase, ifta } = card;
  const lines: IftaLine[] = [
    {
      key: "gallons",
      label: t("Gallons"),
      value: gallonFigure(purchase.gallons),
      note: purchase.quantityUnit === "Litre" ? t("from {0} L", purchase.quantity) : undefined,
    },
  ];
  if (!ifta.countsForIfta) return lines;
  lines.push({
    key: "tax-paid",
    label: t("Tax-paid gallons"),
    value: gallonFigure(ifta.taxPaidGallons),
    zero: Number(ifta.taxPaidGallons) === 0,
  });
  if (purchase.taxPaid && ifta.taxPaidCredit !== null && ifta.ratePerGallon !== null) {
    lines.push({
      key: "credit",
      label: t("Tax already paid"),
      value: money(ifta.taxPaidCredit, purchase.currencyCode),
      note: t("at {0}/gal", formatUnitPrice(ifta.ratePerGallon, purchase.currencyCode)),
    });
  } else if (!purchase.taxPaid) {
    lines.push({ key: "credit", label: t("Tax already paid"), value: t("None") });
  }
  const tractor = purchase.tractor?.code ?? "";
  if (tractor) {
    lines.push({
      key: "quarter",
      label: t("{0} in {1} this quarter", tractor, purchase.jurisdiction.name),
      value: t(
        "{0} gal · {1, plural, one {# purchase} other {# purchases}}",
        gallonFigure(ifta.tractorGallons),
        ifta.tractorPurchaseCount,
      ),
    });
  }
  return lines;
}

/** One line under the IFTA panel saying what the figures above it mean. */
export function iftaNote(card: DeskFuelPurchaseCard, t: TranslateFn): string {
  const { purchase, ifta } = card;
  const name = purchase.jurisdiction.name;
  const tractor = purchase.tractor?.code ?? "";
  if (!ifta.countsForIfta) {
    return t(
      "{0} isn't motor fuel IFTA taxes, so it is spend only and stays off the quarter's return.",
      IFTA_FUEL_TYPE_LABELS[purchase.fuelType],
    );
  }
  if (purchase.quantityUnit === "Litre") {
    return t("IFTA reports in US gallons, so litres are converted when the purchase is saved.");
  }
  if (!purchase.taxPaid) {
    return tractor
      ? t(
          "These gallons count toward {0}'s fuel for the quarter, but earn no {1} credit because tax wasn't paid on them.",
          tractor,
          name,
        )
      : t(
          "These gallons count toward the quarter's fuel, but earn no {0} credit because tax wasn't paid on them.",
          name,
        );
  }
  if (ifta.taxPaidCredit === null) {
    return t(
      "Paid at the pump. No {0} rate is on file for Q{1} {2} to value the credit.",
      name,
      ifta.quarter,
      ifta.year,
    );
  }
  return tractor
    ? t("Paid at the pump, so these gallons are credited against what {0} owes {1}.", tractor, name)
    : t("Paid at the pump, so these gallons are credited against what is owed to {0}.", name);
}

/** The head's kicker: what it is, how it came to be on file, and the reference it carries. */
export function purchaseKicker(card: DeskFuelPurchaseCard, t: TranslateFn): string[] {
  const { purchase, provenance } = card;
  const how =
    purchase.source === "Manual"
      ? t("Entered by hand")
      : provenance
        ? provenance.batch.origin === "Feed"
          ? t("{0} feed", FUEL_CARD_PROVIDER_LABELS[provenance.batch.provider])
          : t("{0} statement", FUEL_CARD_PROVIDER_LABELS[provenance.batch.provider])
        : t("Card import");
  const reference = purchase.transactionReference?.trim() ?? "";
  return reference ? [t("Fuel purchase"), how, reference] : [t("Fuel purchase"), how];
}

/** Where and when the fuel was bought, under the vendor's name. */
export function purchaseSubtitle(purchase: DeskFuelPurchase, t: TranslateFn): string {
  return t(
    "{0} · {1}, {2} at {3}",
    purchasePlace(purchase, t),
    cardWeekday(purchase.purchasedAt),
    new Date(purchase.purchasedAt * 1000).getFullYear(),
    cardTime(purchase.purchasedAt),
  );
}

/** What the total is made of, under it: the quantity, and the price when there is one. */
export function figureCaption(purchase: DeskFuelPurchase, t: TranslateFn): string {
  const price = priceText(purchase);
  return price ? t("{0} at {1}", quantityText(purchase, t), price) : quantityText(purchase, t);
}

/** The facts after the head's pill: what it is to IFTA, the tractor, and who drove it. */
export function purchaseFacts(card: DeskFuelPurchaseCard, t: TranslateFn): string[] {
  const { purchase, ifta } = card;
  return [
    ifta.countsForIfta
      ? t("Counts for IFTA, {0}", quarterLabel(ifta.year, ifta.quarter, t))
      : t("Spend only"),
    purchase.tractor?.code ?? "",
    purchase.worker?.wholeName?.trim() ?? "",
  ].filter((fact) => fact !== "");
}
