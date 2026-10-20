import type { DeskFuelCardCard } from "@/lib/graphql/fuel-desk-cards";
import { translate } from "@trenova/shared/i18n/runtime";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it, vi } from "vitest";
import { cardDay, cardMoment, cardMonth } from "../../../record-card/record-card-values";
import {
  barHint,
  cardFacts,
  cardFigure,
  cardKicker,
  cardStanding,
  cardSubtitle,
  cardTitle,
  expiryFace,
  feedCheck,
  heldPlace,
  heldReason,
  holderFact,
  jurisdictionShare,
  monthScope,
  monthTotals,
  postHeldRows,
  purchaseTax,
  recentNote,
  waitingNote,
} from "../fuel-card-card";
import { LAST_POLL, activeCard, cancelledCard, discoveredCard } from "./fixtures";

const t = translate as TranslateFn;
const NOW = Date.UTC(2026, 9, 20, 7, 18) / 1000;

function withCard(
  card: DeskFuelCardCard,
  change: (copy: DeskFuelCardCard) => void,
): DeskFuelCardCard {
  const copy = structuredClone(card);
  change(copy);
  return copy;
}

describe("the card's head", () => {
  it("names a labelled card by its label and an unlabelled one by its network and last four", () => {
    expect(cardTitle(activeCard().card, t)).toBe("Comdata •••• 4412 — Unit T-101");
    expect(cardTitle(discoveredCard().card, t)).toBe("WEX •••• 7720");
  });

  it("says a card nobody holds yet was found by the feed, and gives others the provider's id", () => {
    expect(cardKicker(discoveredCard().card, t)).toEqual(["Fuel card", "WEX", "Found by the feed"]);
    expect(cardKicker(activeCard().card, t)).toEqual(["Fuel card", "Comdata", "CMD-7781-4412"]);
  });

  it("stands a card by its status, and a suspended one nobody holds as waiting to be assigned", () => {
    expect(cardStanding(activeCard().card, t)).toEqual({
      label: "Active",
      tone: "success",
      mark: "check",
    });
    expect(cardStanding(discoveredCard().card, t)).toEqual({
      label: "Suspended until assigned",
      tone: "warn",
      mark: "dot",
    });
    const paused = withCard(activeCard(), (copy) => {
      copy.card.status = "Suspended";
    });
    expect(cardStanding(paused.card, t)).toEqual({
      label: "Suspended",
      tone: "warn",
      mark: "pause",
    });
    expect(cardStanding(cancelledCard().card, t)).toEqual({
      label: "Cancelled",
      tone: "muted",
      mark: "x",
    });
  });

  it("says when the card expires, expired, was first seen or was cancelled", () => {
    const card = activeCard().card;
    expect(cardSubtitle(card, NOW, t)).toBe(
      `Comdata fleet card · expires ${cardMonth(card.expiresAt)}`,
    );
    const expired = { ...card, expiresAt: NOW - 86_400 };
    expect(cardSubtitle(expired, NOW, t)).toBe(
      `Comdata fleet card · expired ${cardMonth(expired.expiresAt)}`,
    );
    const found = discoveredCard().card;
    expect(cardSubtitle(found, NOW, t)).toBe(
      `WEX fleet card · first seen ${cardDay(found.discoveredAt)} in the WEX transaction file`,
    );
    const cancelled = cancelledCard().card;
    expect(cardSubtitle(cancelled, NOW, t)).toBe(
      `Comdata fleet card · cancelled ${cardMonth(cancelled.cancelledAt)}`,
    );
    expect(cardSubtitle({ ...card, expiresAt: null }, NOW, t)).toBe("Comdata fleet card");
  });

  it("prints the expiry the way the card's face does", () => {
    expect(expiryFace(activeCard().card.expiresAt)).toBe("08/29");
    expect(expiryFace(null)).toBe("—");
  });

  it("leads with this month's spend and gallons", () => {
    expect(cardFigure(activeCard(), t)).toEqual({
      amount: "$4,092.18",
      caption: "spent in October · 1,046.3 gal",
      warn: false,
    });
  });

  it("adds each other currency's spend after the first", () => {
    const card = withCard(activeCard(), (copy) => {
      copy.month.push({
        currencyCode: "CAD",
        purchaseCount: 1,
        gallons: "110.954",
        untaxedGallons: "0",
        totalAmount: "654.78",
      });
    });
    expect(cardFigure(card, t)).toEqual({
      amount: "$4,092.18 + CA$654.78",
      caption: "spent in October · 1,157.3 gal",
      warn: false,
    });
  });

  it("leads a card nobody holds with what is waiting on it", () => {
    expect(cardFigure(discoveredCard(), t)).toEqual({
      amount: "$1,612.45",
      caption: "waiting to post · 418.6 gal",
      warn: true,
    });
  });

  it("draws no figure for a cancelled card or one never used, and nothing spent yet for one used before", () => {
    expect(cardFigure(cancelledCard(), t)).toBeNull();
    const unused = withCard(activeCard(), (copy) => {
      copy.month = [];
      copy.purchaseCount = 0;
      copy.lastPurchasedAt = null;
    });
    expect(cardFigure(unused, t)).toBeNull();
    const quiet = withCard(activeCard(), (copy) => {
      copy.month = [];
    });
    expect(cardFigure(quiet, t)).toEqual({
      amount: "$0.00",
      caption: "spent in October",
      warn: false,
    });
  });

  it("says who holds it, what waits on it and when it was last used", () => {
    const card = activeCard();
    expect(cardFacts(card, t)).toEqual([
      "Held by Rosa Diaz and T-101",
      `Last used ${cardMoment(card.lastPurchasedAt)}`,
    ]);
    expect(cardFacts(discoveredCard(), t)).toEqual([
      "No driver or tractor",
      "12 purchases waiting",
    ]);
    expect(holderFact(cancelledCard().card, t)).toBe("Was held by Jamal Reyes and T-087");
    const tractorOnly = { ...card.card, assignedWorker: null, assignedWorkerId: null };
    expect(holderFact(tractorOnly, t)).toBe("Held by T-101");
  });
});

describe("the card's purchases", () => {
  it("sums this month's purchases and works out the price per gallon", () => {
    expect(monthTotals(activeCard().month)).toEqual({
      purchases: 9,
      gallons: 1046.3,
      untaxedGallons: 15,
      averagePerGallon: 3.911,
    });
    expect(monthTotals([])).toEqual({
      purchases: 0,
      gallons: 0,
      untaxedGallons: 0,
      averagePerGallon: null,
    });
  });

  it("names the month the figures are for and its quarter", () => {
    expect(monthScope(activeCard(), t)).toBe("October so far · Q4 2026");
  });

  it("marks each purchase taxed, untaxed, or spend IFTA leaves out", () => {
    const [diesel, def, untaxed] = activeCard().recent;
    if (!diesel || !def || !untaxed) throw new Error("fixture");
    expect(purchaseTax(diesel, t)).toEqual({ label: "Tax paid", tone: "success" });
    expect(purchaseTax(def, t)).toEqual({ label: "Spend", tone: "neutral" });
    expect(purchaseTax(untaxed, t)).toEqual({ label: "Untaxed", tone: "warning" });
  });

  it("says how many of the card's purchases are shown when it has more", () => {
    expect(recentNote(activeCard(), t)).toBe(
      "3 of 23 shown. Every purchase on the card is on the purchases page.",
    );
    const few = withCard(activeCard(), (copy) => {
      copy.purchaseCount = 3;
    });
    expect(recentNote(few, t)).toBeNull();
  });

  it("draws each jurisdiction's bar against the one the card fuels in most", () => {
    const rows = activeCard().quarterByJurisdiction;
    const [texas, oklahoma] = rows;
    if (!texas || !oklahoma) throw new Error("fixture");
    expect(jurisdictionShare(texas, rows)).toBe(100);
    expect(jurisdictionShare(oklahoma, rows)).toBe(50);
  });
});

describe("the feed", () => {
  it("says when a working feed last read", () => {
    expect(feedCheck(activeCard(), t)).toEqual({
      state: "ok",
      title: "Comdata transaction file",
      detail: "Read every hour by the Comdata connection",
      aside: `Last read ${cardMoment(LAST_POLL)}`,
    });
  });

  it("warns when the feed is failing, with its error and when it last read", () => {
    const failing = withCard(activeCard(), (copy) => {
      if (copy.feed) {
        copy.feed.failureCount = 3;
        copy.feed.lastError = "401 Unauthorized";
      }
    });
    expect(feedCheck(failing, t)).toEqual({
      state: "warn",
      title: "The Comdata feed is failing",
      detail: "3 failed runs · 401 Unauthorized",
      aside: `Last read ${cardMoment(LAST_POLL)}`,
    });
    const never = withCard(failing, (copy) => {
      if (copy.feed) {
        copy.feed.failureCount = 1;
        copy.feed.lastError = null;
        copy.feed.lastSuccessAt = null;
      }
    });
    expect(feedCheck(never, t)).toMatchObject({ detail: "1 failed run", aside: "Never read" });
  });

  it("draws nothing for a provider with no feed", () => {
    const manual = withCard(activeCard(), (copy) => {
      copy.feed = null;
    });
    expect(feedCheck(manual, t)).toBeNull();
  });
});

describe("what is waiting on the card", () => {
  it("says why each held row waits, and where it was bought", () => {
    const [error, waiting] = discoveredCard().held;
    if (!error || !waiting) throw new Error("fixture");
    expect(heldReason(error, t)).toBe("No tractor could be determined for card ending 7720");
    expect(heldReason(waiting, t)).toBe("Waiting for the import to be committed");
    expect(heldReason({ ...error, error: null }, t)).toBe("Waiting for its card to be assigned");
    expect(heldPlace(error, t)).toBe("Joplin, MO");
    expect(heldPlace({ ...error, parsed: null }, t)).toBe("");
  });

  it("says assigning a suspended card activates it and posts what waits, for someone who may work imports", () => {
    expect(waitingNote(discoveredCard(), true, t)).toEqual({
      title: "Assigning it activates the card",
      detail: "The 12 purchases waiting on it are posted once it's assigned.",
    });
    expect(waitingNote(discoveredCard(), false, t)).toEqual({
      title: "Assigning it activates the card",
      detail: "Someone who can import fuel statements then posts what's waiting on it.",
    });
    const one = withCard(discoveredCard(), (copy) => {
      copy.heldCount = 1;
    });
    expect(waitingNote(one, true, t)?.detail).toBe(
      "The purchase waiting on it is posted once it's assigned.",
    );
  });

  it("says nothing about assigning a card someone holds, or a cancelled one", () => {
    expect(waitingNote(activeCard(), true, t)).toBeNull();
    expect(waitingNote(cancelledCard(), true, t)).toBeNull();
  });

  it("points the bar at assigning a card nobody holds, and a cancelled one at its replacement", () => {
    expect(barHint(discoveredCard(), true, t)).toBe(
      "Assigning it activates the card and posts what's waiting",
    );
    expect(barHint(discoveredCard(), false, t)).toBe(
      "Assign, suspend or cancel it on the fuel cards page",
    );
    expect(barHint(cancelledCard(), true, t)).toBe(
      "A cancelled card can't be reopened; add its replacement on the fuel cards page",
    );
  });
});

describe("posting what was waiting", () => {
  it("works every import holding rows in turn, at the version it was read at", async () => {
    const resolve = vi.fn().mockResolvedValue({});
    const refusals = await postHeldRows(discoveredCard().heldBatches, resolve, String);
    expect(resolve.mock.calls).toEqual([
      ["fpib_feed", 4],
      ["fpib_upload", 2],
    ]);
    expect(refusals).toEqual([]);
  });

  it("goes on to the next import when one refuses, and returns why it refused", async () => {
    const resolve = vi
      .fn()
      .mockRejectedValueOnce(new Error("The import was changed by someone else"))
      .mockResolvedValueOnce({});
    const refusals = await postHeldRows(
      discoveredCard().heldBatches,
      resolve,
      (error) => (error as Error).message,
    );
    expect(resolve).toHaveBeenCalledTimes(2);
    expect(refusals).toEqual(["The import was changed by someone else"]);
  });
});
