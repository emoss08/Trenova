import type { DeskFuelPurchaseCard } from "@/lib/graphql/fuel-desk-cards";
import { translate } from "@trenova/shared/i18n/runtime";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import { cardDay, cardTime, cardWeekday } from "../../../record-card/record-card-values";
import {
  figureCaption,
  filedSteps,
  fuelMark,
  iftaBadge,
  iftaLines,
  iftaNote,
  milesSincePrevious,
  purchaseChecks,
  purchaseFacts,
  purchaseKicker,
  purchaseStanding,
  purchaseSubtitle,
  receiptCard,
  receiptLines,
} from "../fuel-purchase-card";
import { PURCHASED_AT, fuelCardOnFile, importedPurchase, manualPurchase } from "./fixtures";

const t = translate as TranslateFn;

function withCard(
  card: DeskFuelPurchaseCard,
  change: (copy: DeskFuelPurchaseCard) => void,
): DeskFuelPurchaseCard {
  const copy = structuredClone(card);
  change(copy);
  return copy;
}

function litres(): DeskFuelPurchaseCard {
  return withCard(importedPurchase(), (card) => {
    card.purchase.vendor = "Petro-Canada";
    card.purchase.vendorCity = "Windsor";
    card.purchase.jurisdiction = { id: "jur_on", countryCode: "CA", code: "ON", name: "Ontario" };
    card.purchase.quantity = "420";
    card.purchase.quantityUnit = "Litre";
    card.purchase.gallons = "110.954";
    card.purchase.unitPrice = "1.559";
    card.purchase.totalAmount = "654.78";
    card.purchase.currencyCode = "CAD";
    card.ifta.taxPaidGallons = "110.954";
  });
}

describe("the purchase's head", () => {
  it("marks the stub with the fuel the way a pump abbreviates it", () => {
    expect(fuelMark("Diesel")).toBe("DSL");
    expect(fuelMark("DEF")).toBe("DEF");
    expect(fuelMark("Reefer")).toBe("RFR");
  });

  it("names how a feed's purchase came to be on file and the reference it carries", () => {
    expect(purchaseKicker(importedPurchase(), t)).toEqual([
      "Fuel purchase",
      "Comdata feed",
      "TX-20261019-004812",
    ]);
  });

  it("says an upload's purchase came from a statement, and a hand-entered one by hand", () => {
    const uploaded = withCard(importedPurchase(), (card) => {
      if (card.provenance) card.provenance.batch.origin = "Upload";
    });
    expect(purchaseKicker(uploaded, t)[1]).toBe("Comdata statement");
    expect(purchaseKicker(manualPurchase(), t)[1]).toBe("Entered by hand");
  });

  it("says only that it was imported when the reader may not see the import", () => {
    const unseen = withCard(importedPurchase(), (card) => {
      card.provenance = null;
      card.purchase.transactionReference = null;
    });
    expect(purchaseKicker(unseen, t)).toEqual(["Fuel purchase", "Card import"]);
  });

  it("says where and when it was bought", () => {
    expect(purchaseSubtitle(importedPurchase().purchase, t)).toBe(
      `Amarillo, TX · ${cardWeekday(PURCHASED_AT)}, 2026 at ${cardTime(PURCHASED_AT)}`,
    );
  });

  it("puts the quantity and price under the total, in the unit it was bought in", () => {
    expect(figureCaption(importedPurchase().purchase, t)).toBe("118.42 gal at $3.8990");
    expect(figureCaption(litres().purchase, t)).toBe("420 L at CA$1.5590");
  });

  it("names the quarter it counts for, the tractor and the driver", () => {
    expect(purchaseFacts(importedPurchase(), t)).toEqual([
      "Counts for IFTA, Q4 2026",
      "T-101",
      "Rosa Diaz",
    ]);
    const def = withCard(importedPurchase(), (card) => {
      card.ifta.countsForIfta = false;
      card.purchase.worker = null;
    });
    expect(purchaseFacts(def, t)).toEqual(["Spend only", "T-101"]);
  });
});

describe("the purchase's checks", () => {
  it("finds nothing to fix on a feed purchase whose total, odometer and card agree", () => {
    const card = importedPurchase();
    const checks = purchaseChecks(card, t);
    expect(checks.map((check) => [check.key, check.state, check.title, check.detail])).toEqual([
      ["total", "ok", "Total matches quantity × price", "118.42 gal × $3.8990 = $461.72"],
      [
        "odometer",
        "ok",
        "Odometer moved on",
        `+736 mi since Love's, Oklahoma City on ${cardDay(card.previousFill?.purchasedAt)} · 6.2 mpg`,
      ],
      [
        "card",
        "ok",
        "Card belongs with this unit",
        "Comdata •••• 4412 — Unit T-101 is assigned to T-101 and Rosa Diaz",
      ],
    ]);
    expect(purchaseStanding(card, checks, t)).toEqual({
      label: "Tax paid in TX",
      tone: "success",
      mark: "check",
    });
  });

  it("flags a total off from quantity × price by how much, and a fill with no odometer", () => {
    const card = manualPurchase();
    const checks = purchaseChecks(card, t);
    expect(checks.map((check) => [check.key, check.state, check.detail])).toEqual([
      ["total", "warn", "160 gal × $3.4890 = $558.24, $4.30 less than the $562.54 entered"],
      [
        "odometer",
        "warn",
        "Without one, T-101's miles per gallon for this fill can't be worked out",
      ],
      ["card", "info", "Entered by hand without a fuel card"],
    ]);
    expect(purchaseStanding(card, checks, t)).toEqual({
      label: "2 to check",
      tone: "warn",
      mark: "dot",
    });
  });

  it("says when the entered total is below quantity × price", () => {
    const card = withCard(manualPurchase(), (copy) => {
      copy.purchase.totalAmount = "550.00";
    });
    expect(purchaseChecks(card, t)[0]?.detail).toBe(
      "160 gal × $3.4890 = $558.24, $8.24 more than the $550.00 entered",
    );
  });

  it("leaves the total unchecked when no price was recorded", () => {
    const card = withCard(importedPurchase(), (copy) => {
      copy.purchase.unitPrice = null;
    });
    expect(purchaseChecks(card, t).map((check) => check.key)).toEqual(["odometer", "card"]);
  });

  it("warns when the odometer is no further on than the fill before it", () => {
    const card = withCard(importedPurchase(), (copy) => {
      copy.purchase.odometer = 180868;
    });
    const odometer = purchaseChecks(card, t).find((check) => check.key === "odometer");
    expect(odometer).toMatchObject({
      state: "warn",
      title: "Odometer didn't move on",
      detail: `180,868 mi is no further than the 180,868 mi recorded at Love's, Oklahoma City on ${cardDay(card.previousFill?.purchasedAt)}`,
    });
    expect(milesSincePrevious(card)).toEqual({ miles: 0, mpg: null });
  });

  it("says a tractor's first fill with an odometer has nothing to measure from", () => {
    const card = withCard(importedPurchase(), (copy) => {
      copy.previousFill = null;
    });
    expect(purchaseChecks(card, t)[1]).toMatchObject({
      state: "info",
      title: "First fill with an odometer",
    });
  });

  it("leaves the odometer out for fuel that is not IFTA's, such as DEF", () => {
    const card = withCard(importedPurchase(), (copy) => {
      copy.purchase.fuelType = "DEF";
      copy.ifta.countsForIfta = false;
    });
    expect(purchaseChecks(card, t).map((check) => check.key)).toEqual(["total", "card"]);
    expect(purchaseStanding(card, purchaseChecks(card, t), t).label).toBe("Not IFTA fuel");
  });

  it("warns about a card that was cancelled or belongs to another tractor", () => {
    const cancelled = withCard(importedPurchase(), (copy) => {
      copy.purchase.fuelCard = fuelCardOnFile({ status: "Cancelled" });
    });
    expect(purchaseChecks(cancelled, t).at(-1)).toMatchObject({
      state: "warn",
      title: "Card was cancelled",
    });

    const elsewhere = withCard(importedPurchase(), (copy) => {
      copy.purchase.fuelCard = fuelCardOnFile({
        assignedTractorId: "tr_205",
        assignedTractor: { id: "tr_205", code: "T-205" },
      });
    });
    expect(purchaseChecks(elsewhere, t).at(-1)).toMatchObject({
      state: "warn",
      title: "Card belongs to another tractor",
      detail: "Comdata •••• 4412 — Unit T-101 is assigned to T-205",
    });
  });

  it("names only the driver for a card that a driver carries and no tractor holds", () => {
    const carried = withCard(importedPurchase(), (copy) => {
      copy.purchase.fuelCard = fuelCardOnFile({ assignedTractorId: null, assignedTractor: null });
    });
    expect(purchaseChecks(carried, t).at(-1)?.detail).toBe(
      "Comdata •••• 4412 — Unit T-101 is carried by Rosa Diaz",
    );
  });

  it("says a card on no tractor and no driver isn't assigned", () => {
    const loose = withCard(importedPurchase(), (copy) => {
      copy.purchase.fuelCard = fuelCardOnFile({
        assignedTractorId: null,
        assignedTractor: null,
        assignedWorkerId: null,
        assignedWorker: null,
      });
    });
    expect(purchaseChecks(loose, t).at(-1)).toMatchObject({
      state: "info",
      title: "Card isn't assigned",
    });
  });
});

describe("the receipt", () => {
  it("prints the lines as recorded, with the card and the odometer", () => {
    const card = importedPurchase();
    expect(
      receiptLines(card.purchase, t).map((line) => [line.label, line.value, line.mark]),
    ).toEqual([
      ["Product", "Diesel", undefined],
      ["Quantity", "118.42 gal", undefined],
      ["Price / gal", "$3.8990", undefined],
      ["Odometer", "181,604 mi", undefined],
    ]);
    expect(receiptCard(card.purchase, t)).toBe("Comdata •••• 4412");
  });

  it("sets the gallons a litre purchase was converted to under its quantity", () => {
    expect(receiptLines(litres().purchase, t).slice(1, 3)).toEqual([
      { key: "quantity", label: "Quantity", value: "420 L" },
      { key: "gallons", label: "", value: "= 110.954 gal", mark: "derived" },
    ]);
  });

  it("adds quantity × price where it disagrees with the total, and marks a missing odometer", () => {
    const lines = receiptLines(manualPurchase().purchase, t);
    expect(lines.find((line) => line.key === "computed")).toEqual({
      key: "computed",
      label: "Qty × price",
      value: "$558.24",
      mark: "off",
    });
    expect(lines.at(-1)).toEqual({
      key: "odometer",
      label: "Odometer",
      value: "Not recorded",
      mark: "missing",
    });
  });

  it("prints the last four the statement gave when no card on file matched", () => {
    const card = withCard(importedPurchase(), (copy) => {
      copy.purchase.fuelCard = null;
    });
    expect(receiptCard(card.purchase, t)).toBe("•••• 4412");
    expect(receiptCard(manualPurchase().purchase, t)).toBe("");
  });
});

describe("the IFTA panel", () => {
  it("credits a taxed fill at the quarter's rate and totals the tractor's fuel there", () => {
    const card = importedPurchase();
    expect(iftaBadge(card, t)).toEqual({ label: "Counts", tone: "success" });
    expect(iftaLines(card, t)).toEqual([
      { key: "gallons", label: "Gallons", value: "118.420", note: undefined },
      { key: "tax-paid", label: "Tax-paid gallons", value: "118.420", zero: false },
      {
        key: "credit",
        label: "Tax already paid",
        value: "$23.68",
        note: "at $0.2000/gal",
      },
      {
        key: "quarter",
        label: "T-101 in Texas this quarter",
        value: "612.840 gal · 5 purchases",
      },
    ]);
    expect(iftaNote(card, t)).toBe(
      "Paid at the pump, so these gallons are credited against what T-101 owes Texas.",
    );
  });

  it("leaves the credit out of a taxed fill when no rate is on file, and says so", () => {
    const card = withCard(importedPurchase(), (copy) => {
      copy.ifta.ratePerGallon = null;
      copy.ifta.taxPaidCredit = null;
    });
    expect(iftaLines(card, t).map((line) => line.key)).toEqual(["gallons", "tax-paid", "quarter"]);
    expect(iftaNote(card, t)).toBe(
      "Paid at the pump. No Texas rate is on file for Q4 2026 to value the credit.",
    );
  });

  it("marks an untaxed fill's tax-paid gallons as none and earns it no credit", () => {
    const card = manualPurchase();
    expect(iftaBadge(card, t)).toEqual({ label: "Untaxed", tone: "warning" });
    const lines = iftaLines(card, t);
    expect(lines[1]).toEqual({
      key: "tax-paid",
      label: "Tax-paid gallons",
      value: "0.000",
      zero: true,
    });
    expect(lines[2]).toEqual({ key: "credit", label: "Tax already paid", value: "None" });
    expect(lines[3]?.value).toBe("1,204.500 gal · 8 purchases");
    expect(iftaNote(card, t)).toBe(
      "These gallons count toward T-101's fuel for the quarter, but earn no Texas credit because tax wasn't paid on them.",
    );
  });

  it("shows a litre purchase's gallons against the litres it came from", () => {
    const card = litres();
    expect(iftaLines(card, t)[0]).toEqual({
      key: "gallons",
      label: "Gallons",
      value: "110.954",
      note: "from 420 L",
    });
    expect(iftaNote(card, t)).toBe(
      "IFTA reports in US gallons, so litres are converted when the purchase is saved.",
    );
  });

  it("keeps fuel IFTA doesn't tax off the return", () => {
    const card = withCard(importedPurchase(), (copy) => {
      copy.purchase.fuelType = "DEF";
      copy.ifta.countsForIfta = false;
    });
    expect(iftaBadge(card, t)).toEqual({ label: "Not IFTA fuel", tone: "neutral" });
    expect(iftaLines(card, t).map((line) => line.key)).toEqual(["gallons"]);
    expect(iftaNote(card, t)).toBe(
      "Diesel exhaust fluid (DEF) isn't motor fuel IFTA taxes, so it is spend only and stays off the quarter's return.",
    );
  });
});

describe("how the purchase was filed", () => {
  it("reads a feed's row and each match the import made", () => {
    const card = importedPurchase();
    expect(filedSteps(card, t)).toEqual([
      {
        key: "read",
        kind: "feed",
        title: "Read from the Comdata transaction file",
        detail: `Row 37 of 112 · ${cardDay(card.provenance?.batch.createdAt)}`,
      },
      { key: "note-0", kind: "note", title: "Card found by its last four", detail: "" },
      { key: "note-1", kind: "note", title: "Tractor matched by unit code", detail: "" },
    ]);
  });

  it("names the uploaded file a statement's purchase came from", () => {
    const card = withCard(importedPurchase(), (copy) => {
      if (copy.provenance) {
        copy.provenance.batch.origin = "Upload";
        copy.provenance.batch.fileName = "comdata-oct.csv";
      }
    });
    expect(filedSteps(card, t)[0]).toMatchObject({
      kind: "upload",
      title: "Imported from comdata-oct.csv",
    });
  });

  it("says who entered a purchase by hand", () => {
    const card = manualPurchase();
    expect(filedSteps(card, t)).toEqual([
      {
        key: "hand",
        kind: "hand",
        title: "Entered by Dana Park",
        detail: `On ${cardDay(card.purchase.createdAt)}, with the tractor, driver and place chosen by hand`,
      },
    ]);
  });

  it("draws no path for an import the reader may not see", () => {
    const card = withCard(importedPurchase(), (copy) => {
      copy.provenance = null;
    });
    expect(filedSteps(card, t)).toEqual([]);
  });
});
