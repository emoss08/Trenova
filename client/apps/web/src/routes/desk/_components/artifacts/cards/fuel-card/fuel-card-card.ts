import type {
  DeskFuelCardCard,
  DeskFuelCardPurchase,
  DeskFuelCardRecord,
  DeskFuelHeldRow,
  DeskFuelJurisdictionGallons,
  DeskFuelSpend,
} from "@/lib/graphql/fuel-desk-cards";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { countsForIfta, FUEL_CARD_PROVIDER_LABELS } from "@trenova/shared/types/fuel-ifta-enums";
import type { CardCheckState, CardStatusMark, CardTone } from "../../record-card/record-card";
import { cardDay, cardMonth, cardMoment, money } from "../../record-card/record-card-values";

export function isUnassigned(
  card: Pick<DeskFuelCardRecord, "assignedTractorId" | "assignedWorkerId">,
): boolean {
  return !card.assignedTractorId && !card.assignedWorkerId;
}

export type CardStanding = { label: string; tone: CardTone; mark: CardStatusMark };

export function cardStanding(card: DeskFuelCardRecord, t: TranslateFn): CardStanding {
  switch (card.status) {
    case "Active":
      return { label: t("Active"), tone: "success", mark: "check" };
    case "Suspended":
      return isUnassigned(card)
        ? { label: t("Suspended until assigned"), tone: "warn", mark: "dot" }
        : { label: t("Suspended"), tone: "warn", mark: "pause" };
    case "Cancelled":
      return { label: t("Cancelled"), tone: "muted", mark: "x" };
  }
}

/** When the card expires, as its face prints it: month and two-digit year. */
export function expiryFace(expiresAt: number | null): string {
  if (!expiresAt) return "—";
  const date = new Date(expiresAt * 1000);
  return `${String(date.getMonth() + 1).padStart(2, "0")}/${String(date.getFullYear() % 100).padStart(2, "0")}`;
}

/** The card's subtitle: its provider, and when it expires, expired or was cancelled. */
export function cardSubtitle(card: DeskFuelCardRecord, now: number, t: TranslateFn): string {
  const provider = FUEL_CARD_PROVIDER_LABELS[card.provider];
  if (card.status === "Cancelled" && card.cancelledAt) {
    return t("{0} fleet card · cancelled {1}", provider, cardMonth(card.cancelledAt));
  }
  if (card.discoveredAt && isUnassigned(card)) {
    return t(
      "{0} fleet card · first seen {1} in the {0} transaction file",
      provider,
      cardDay(card.discoveredAt),
    );
  }
  if (!card.expiresAt) return t("{0} fleet card", provider);
  return card.expiresAt <= now
    ? t("{0} fleet card · expired {1}", provider, cardMonth(card.expiresAt))
    : t("{0} fleet card · expires {1}", provider, cardMonth(card.expiresAt));
}

export type CardFigure = { amount: string; caption: string; warn: boolean };

function gallonsText(gallons: string): string {
  const figure = Number(gallons);
  return Number.isFinite(figure)
    ? figure.toLocaleString(undefined, { minimumFractionDigits: 1, maximumFractionDigits: 1 })
    : gallons;
}

export function formatGallons(gallons: string, t: TranslateFn): string {
  return t("{0} gal", gallonsText(gallons));
}

/** Spend in more than one currency is read in its largest, with the rest named after it. */
function spendAmount(
  rows: readonly { currencyCode: string; totalAmount: string }[],
  t: TranslateFn,
): string {
  const [first, ...rest] = rows;
  if (!first) return "";
  const lead = money(first.totalAmount, first.currencyCode);
  return rest.length === 0
    ? lead
    : t("{0} + {1}", lead, rest.map((row) => money(row.totalAmount, row.currencyCode)).join(" + "));
}

function totalGallons(rows: readonly { gallons: string }[]): string {
  return String(rows.reduce((sum, row) => sum + Number(row.gallons), 0));
}

/**
 * The head's figure. A card nobody holds yet leads with what is waiting on it;
 * any other card with what it spent this month. Null for someone who may not
 * read fuel purchases.
 */
export function cardFigure(card: DeskFuelCardCard, t: TranslateFn): CardFigure | null {
  if (isUnassigned(card.card) && card.heldCount > 0 && card.heldSpend.length > 0) {
    return {
      amount: spendAmount(card.heldSpend, t),
      caption: t("waiting to post · {0}", formatGallons(totalGallons(card.heldSpend), t)),
      warn: true,
    };
  }
  if (card.card.status === "Cancelled") return null;
  if (card.month.length === 0) {
    return card.purchaseCount > 0 || card.lastPurchasedAt
      ? { amount: money(0), caption: t("spent in {0}", monthName(card.monthStart)), warn: false }
      : null;
  }
  return {
    amount: spendAmount(card.month, t),
    caption: t(
      "spent in {0} · {1}",
      monthName(card.monthStart),
      formatGallons(totalGallons(card.month), t),
    ),
    warn: false,
  };
}

function monthName(seconds: number): string {
  return new Date(seconds * 1000).toLocaleDateString(undefined, { month: "long" });
}

/** Under the month's figures: which month they are for and the quarter it falls in. */
export function monthScope(card: DeskFuelCardCard, t: TranslateFn): string {
  return t("{0} so far · Q{1} {2}", monthName(card.monthStart), card.quarter, card.year);
}

/** This month's spend summed across currencies' counts and gallons, for the strip under the card. */
export function monthTotals(month: readonly DeskFuelSpend[]): {
  purchases: number;
  gallons: number;
  untaxedGallons: number;
  averagePerGallon: number | null;
} {
  const purchases = month.reduce((sum, row) => sum + row.purchaseCount, 0);
  const gallons = month.reduce((sum, row) => sum + Number(row.gallons), 0);
  const untaxedGallons = month.reduce((sum, row) => sum + Number(row.untaxedGallons), 0);
  const lead = month[0];
  const leadGallons = lead ? Number(lead.gallons) : 0;
  return {
    purchases,
    gallons,
    untaxedGallons,
    averagePerGallon:
      lead && leadGallons > 0
        ? Math.round((Number(lead.totalAmount) / leadGallons) * 1000) / 1000
        : null,
  };
}

/** A jurisdiction's bar against the jurisdiction the card bought most fuel in. */
export function jurisdictionShare(
  row: DeskFuelJurisdictionGallons,
  rows: readonly DeskFuelJurisdictionGallons[],
): number {
  const most = Math.max(...rows.map((candidate) => Number(candidate.gallons)));
  return most > 0 ? Math.round((Number(row.gallons) / most) * 1000) / 10 : 0;
}

export type FeedCheck = { state: CardCheckState; title: string; detail: string; aside: string };

/** Whether the provider's transaction feed is reading, and when it last did. */
export function feedCheck(card: DeskFuelCardCard, t: TranslateFn): FeedCheck | null {
  const feed = card.feed;
  if (!feed) return null;
  const provider = FUEL_CARD_PROVIDER_LABELS[card.card.provider];
  if (feed.failureCount > 0) {
    return {
      state: "warn",
      title: t("The {0} feed is failing", provider),
      detail: feed.lastError?.trim()
        ? t(
            "{0, plural, one {# failed run} other {# failed runs}} · {1}",
            feed.failureCount,
            feed.lastError.trim(),
          )
        : t("{0, plural, one {# failed run} other {# failed runs}}", feed.failureCount),
      aside: feed.lastSuccessAt
        ? t("Last read {0}", cardMoment(feed.lastSuccessAt))
        : t("Never read"),
    };
  }
  return {
    state: "ok",
    title: t("{0} transaction file", provider),
    detail: t("Read every hour by the {0} connection", provider),
    aside: feed.lastSuccessAt ? t("Last read {0}", cardMoment(feed.lastSuccessAt)) : "",
  };
}

/** Why a held row is still waiting, in the statement's own terms. */
export function heldReason(row: DeskFuelHeldRow, t: TranslateFn): string {
  if (row.status === "New") return t("Waiting for the import to be committed");
  return row.error?.trim() || t("Waiting for its card to be assigned");
}

export function heldPlace(row: DeskFuelHeldRow, t: TranslateFn): string {
  const parsed = row.parsed;
  const city = parsed?.vendorCity?.trim() ?? "";
  const code = parsed?.jurisdictionCode?.trim() ?? "";
  return city && code ? t("{0}, {1}", city, code) : city || code;
}

/** The card's name: the label it was given, or its network and last four. */
export function cardTitle(card: DeskFuelCardRecord, t: TranslateFn): string {
  return (
    card.label.trim() || t("{0} •••• {1}", FUEL_CARD_PROVIDER_LABELS[card.provider], card.lastFour)
  );
}

/** The head's kicker: what it is, its network, and how it came to be on file or the provider's id for it. */
export function cardKicker(card: DeskFuelCardRecord, t: TranslateFn): string[] {
  const kicker = [t("Fuel card"), FUEL_CARD_PROVIDER_LABELS[card.provider]];
  if (card.discoveredAt && isUnassigned(card)) return [...kicker, t("Found by the feed")];
  const external = card.externalCardId?.trim() ?? "";
  return external ? [...kicker, external] : kicker;
}

function holders(card: DeskFuelCardRecord): { worker: string; tractor: string } {
  return {
    worker: card.assignedWorker?.wholeName?.trim() ?? "",
    tractor: card.assignedTractor?.code ?? "",
  };
}

/** Who carries the card, as the head says it; in the past tense once it is cancelled. */
export function holderFact(card: DeskFuelCardRecord, t: TranslateFn): string {
  const { worker, tractor } = holders(card);
  const was = card.status === "Cancelled";
  if (worker && tractor) {
    return was
      ? t("Was held by {0} and {1}", worker, tractor)
      : t("Held by {0} and {1}", worker, tractor);
  }
  const one = worker || tractor;
  if (one) return was ? t("Was held by {0}", one) : t("Held by {0}", one);
  return t("No driver or tractor");
}

/** The facts after the head's pill: who holds it, what is waiting on it, and when it was last used. */
export function cardFacts(card: DeskFuelCardCard, t: TranslateFn): string[] {
  const facts = [holderFact(card.card, t)];
  if (card.heldCount > 0) {
    facts.push(
      t("{0, plural, one {# purchase waiting} other {# purchases waiting}}", card.heldCount),
    );
  }
  if (card.lastPurchasedAt) facts.push(t("Last used {0}", cardMoment(card.lastPurchasedAt)));
  return facts;
}

export type PurchaseTax = { label: string; tone: "success" | "warning" | "neutral" };

/** What a purchase on the card is to IFTA: taxed at the pump, untaxed, or fuel the return leaves out. */
export function purchaseTax(purchase: DeskFuelCardPurchase, t: TranslateFn): PurchaseTax {
  if (!countsForIfta(purchase.fuelType)) return { label: t("Spend"), tone: "neutral" };
  return purchase.taxPaid
    ? { label: t("Tax paid"), tone: "success" }
    : { label: t("Untaxed"), tone: "warning" };
}

/** Under the recent purchases, when there are more than the card shows. */
export function recentNote(card: DeskFuelCardCard, t: TranslateFn): string | null {
  if (card.purchaseCount <= card.recent.length) return null;
  return t(
    "{0} of {1} shown. Every purchase on the card is on the purchases page.",
    card.recent.length,
    card.purchaseCount,
  );
}

export type WaitingNote = { title: string; detail: string };

/**
 * What assigning the card does, for a card nobody holds yet: it is activated,
 * and what is waiting on it is posted when the person may work imports again.
 * Null when nothing would change by assigning it.
 */
export function waitingNote(
  card: DeskFuelCardCard,
  mayResolve: boolean,
  t: TranslateFn,
): WaitingNote | null {
  if (!isUnassigned(card.card) || card.card.status === "Cancelled") return null;
  const title =
    card.card.status === "Suspended"
      ? t("Assigning it activates the card")
      : t("Assigning it ties the card to who carries it");
  if (card.heldCount === 0) {
    return {
      title,
      detail: t("Its next purchases are matched to the driver and tractor you choose."),
    };
  }
  return {
    title,
    detail: mayResolve
      ? t(
          "{0, plural, one {The purchase waiting on it is posted once it's assigned.} other {The # purchases waiting on it are posted once it's assigned.}}",
          card.heldCount,
        )
      : t("Someone who can import fuel statements then posts what's waiting on it."),
  };
}

/** The bar's hint, saying what can be done with the card and where. */
export function barHint(card: DeskFuelCardCard, mayAssign: boolean, t: TranslateFn): string {
  if (card.card.status === "Cancelled") {
    return t("A cancelled card can't be reopened; add its replacement on the fuel cards page");
  }
  if (mayAssign && isUnassigned(card.card)) {
    return card.heldCount > 0
      ? t("Assigning it activates the card and posts what's waiting")
      : t("Assign it to the driver or tractor that carries it");
  }
  return t("Assign, suspend or cancel it on the fuel cards page");
}

/** One import a card's held rows sit in, as resolving it again needs it. */
export type HeldBatchRef = { id: string; version: number };

/**
 * Works each import holding rows for the card again, one after another, once
 * the card is assigned, so the rows it held take the card's new driver and
 * tractor; a feed's import posts them as it goes. An import that refuses says
 * why, and the rest are still worked. Returns the refusals, in import order.
 */
export async function postHeldRows(
  batches: readonly HeldBatchRef[],
  resolve: (id: string, version: number) => Promise<unknown>,
  describe: (error: unknown) => string,
): Promise<string[]> {
  const refusals: string[] = [];
  for (const batch of batches) {
    try {
      // oxlint-disable-next-line no-await-in-loop -- each import is worked in turn so one refusal doesn't race the next
      await resolve(batch.id, batch.version);
    } catch (error) {
      refusals.push(describe(error));
    }
  }
  return refusals;
}
