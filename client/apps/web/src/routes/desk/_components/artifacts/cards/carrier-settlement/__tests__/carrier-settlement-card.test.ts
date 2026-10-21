import { describe, expect, it } from "vitest";
import {
  carrierName,
  costGroups,
  headFacts,
  lineTotal,
  moveRows,
  moveStanding,
  moveTally,
  offBy,
  remittance,
  settlementFigure,
  settlementKicker,
  settlementSubtitle,
} from "../carrier-settlement-card";
import {
  emptySettlement,
  move,
  paidSettlement,
  pendingSettlement,
  t,
  voidedSettlement,
} from "./fixtures";

describe("carrier settlement card values", () => {
  it("heads the card with the number, the period and the carrier's terms", () => {
    expect(settlementKicker(pendingSettlement(), t)).toEqual([
      "CS-2610-0088",
      "Oct 5 – Oct 11",
      "Net 30",
    ]);
    expect(settlementKicker(emptySettlement(), t)).toEqual(["CS-2610-0091", "Oct 5 – Oct 11"]);
  });

  it("names the carrier and the loads it is paid for, with its MC number when it has one", () => {
    expect(carrierName(pendingSettlement(), t)).toBe("Blue Ridge Freight LLC");
    expect(carrierName(pendingSettlement({ carrier: null }), t)).toBe("Unknown carrier");
    expect(settlementSubtitle(pendingSettlement(), t)).toBe("Payable for 6 loads · MC 884213");
    expect(settlementSubtitle(pendingSettlement({ shipmentCount: 1 }), t)).toBe(
      "Payable for 1 load · MC 884213",
    );
    expect(settlementSubtitle(emptySettlement(), t)).toBe("No loads this period");
  });

  it("captions the net payable by where the settlement stands", () => {
    const pending = settlementFigure(pendingSettlement(), t);
    expect(pending.amount).toBe("$10,600.00");
    expect(pending.caption).toMatch(/^net payable · pays \w+, Nov 10$/u);
    expect(pending.struck).toBe(false);
    expect(settlementFigure(paidSettlement(), t)).toEqual({
      amount: "$10,600.00",
      caption: "net payable · paid Nov 10",
      struck: false,
    });
    expect(settlementFigure(voidedSettlement(), t)).toEqual({
      amount: "$10,600.00",
      caption: "net payable · voided",
      struck: true,
    });
  });

  it("reads each invoice status as matched, off the buy rate, waiting or not in", () => {
    expect(moveStanding(move({ match: { status: "Matched" } }))).toBe("matched");
    expect(moveStanding(move({ match: { status: "Resolved", varianceMinor: 6_000 } }))).toBe(
      "matched",
    );
    expect(moveStanding(move({ match: { status: "Variance" } }))).toBe("off");
    expect(moveStanding(move({ match: { status: "Suggested" } }))).toBe("waiting");
    expect(moveStanding(move({ match: null }))).toBe("none");
  });

  it("counts the moves each way, and none when the server read no invoices", () => {
    expect(moveTally(pendingSettlement())).toEqual({ matched: 4, off: 1, waiting: 0, none: 1 });
    expect(moveTally(emptySettlement())).toEqual({ matched: 0, off: 0, waiting: 0, none: 0 });
  });

  it("puts the invoices that need someone beside the pill until the settlement is done with", () => {
    expect(headFacts(pendingSettlement(), t)).toEqual([
      "1 invoice off the buy rate",
      "1 load not invoiced",
    ]);
    expect(
      headFacts(
        pendingSettlement({
          moveInvoices: [
            move({ moveId: "smv_a", match: { status: "Suggested" } }),
            move({ moveId: "smv_b", match: { status: "Suggested" } }),
            move({ moveId: "smv_c", match: { status: "Variance", varianceMinor: -500 } }),
          ],
        }),
        t,
      ),
    ).toEqual(["1 invoice off the buy rate", "2 invoices waiting to be accepted"]);
    expect(headFacts(voidedSettlement(), t)).toEqual([]);
    expect(headFacts(paidSettlement(), t)).toEqual(["Check · CHK 10442"]);
    expect(headFacts(paidSettlement({ paymentReference: " " }), t)).toEqual(["Check"]);
  });

  it("lists the moves that need someone, off the buy rate first, and never a matched one", () => {
    const record = pendingSettlement({
      moveInvoices: [
        move({ moveId: "smv_a", proNumber: "S-1", costMinor: 100_000, match: null }),
        move({
          moveId: "smv_b",
          proNumber: "S-2",
          match: {
            status: "Suggested",
            invoiceNumber: "INV-9",
            invoiceTotalMinor: 120_000,
            expectedTotalMinor: 120_000,
          },
        }),
        move({ moveId: "smv_c", proNumber: "S-3" }),
        move({
          moveId: "smv_d",
          proNumber: "S-4",
          match: {
            status: "Variance",
            invoiceNumber: "INV-7",
            invoiceTotalMinor: 95_000,
            expectedTotalMinor: 100_000,
            varianceMinor: -5_000,
          },
        }),
      ],
    });

    expect(moveRows(record, t)).toEqual([
      {
        moveId: "smv_d",
        standing: "off",
        title: "Invoice INV-7",
        pro: "S-4",
        line: "Billed $950.00 · buy rate $1,000.00",
        variance: "−$50.00",
        status: "Under",
      },
      {
        moveId: "smv_b",
        standing: "waiting",
        title: "Invoice INV-9",
        pro: "S-2",
        line: "Billed $1,200.00 · buy rate $1,200.00",
        variance: "",
        status: "Waiting",
      },
      {
        moveId: "smv_a",
        standing: "none",
        title: "No invoice yet",
        pro: "S-1",
        line: "Costs $1,000.00 on this statement",
        variance: "",
        status: "Not invoiced",
      },
    ]);
  });

  it("sums the invoices off the buy rate, over against under", () => {
    expect(offBy(pendingSettlement(), t)).toBe("+$85.00");
    const mixed = pendingSettlement({
      moveInvoices: [
        move({ moveId: "smv_a", match: { status: "Variance", varianceMinor: 8_500 } }),
        move({ moveId: "smv_b", match: { status: "Variance", varianceMinor: -12_000 } }),
        move({ moveId: "smv_c", match: { status: "Resolved", varianceMinor: 40_000 } }),
      ],
    });
    expect(offBy(mixed, t)).toBe("−$35.00");
    expect(offBy(emptySettlement(), t)).toBe("");
  });

  it("sums the costs by kind over the adjustments, a hand-added one marked", () => {
    expect(costGroups(pendingSettlement(), t)).toEqual([
      {
        key: "loads",
        title: "Load costs",
        total: "$10,625.00",
        rows: [
          {
            key: "load:0",
            name: "Linehaul",
            detail: "Rate confirmation · 6 loads",
            basis: "As rate confirmed",
            amount: "$9,600.00",
            manual: false,
          },
          {
            key: "load:1",
            name: "Fuel surcharge",
            detail: "6 loads",
            basis: "As rate confirmed",
            amount: "$875.00",
            manual: false,
          },
          {
            key: "load:2",
            name: "Lumper",
            detail: "1 load",
            basis: "As rate confirmed",
            amount: "$150.00",
            manual: false,
          },
        ],
      },
      {
        key: "adjustments",
        title: "Adjustments",
        total: "−$25.00",
        rows: [
          {
            key: "adjustment:0",
            name: "Invoice variance INV-77120",
            detail: "1 load",
            basis: "Invoice variance accepted",
            amount: "$60.00",
            manual: false,
          },
          {
            key: "adjustment:1",
            name: "Late delivery credit",
            detail: "",
            basis: "Flat",
            amount: "−$85.00",
            manual: true,
          },
        ],
      },
    ]);
    expect(lineTotal(pendingSettlement())).toBe(15);
    expect(costGroups(emptySettlement(), t)).toEqual([]);
    expect(lineTotal(emptySettlement())).toBe(0);
  });

  it("says where the money goes, the carrier's preferred method until it is paid", () => {
    expect(remittance(pendingSettlement(), t)).toEqual({
      withheld: false,
      lines: ["Blue Ridge Freight Remittance", "PO Box 4410", "Roanoke, VA 24015"],
      method: "ACH (manual) · the carrier's preferred",
      terms: "Net 30 · pays Nov 10",
    });
    expect(remittance(paidSettlement(), t).method).toBe("Check");
    expect(remittance(emptySettlement(), t)).toEqual({
      withheld: false,
      lines: [],
      method: "Check · the carrier's preferred",
      terms: "Pays Nov 10",
    });
  });

  it("says the carrier's payment details are withheld, not missing, without carrier read", () => {
    const base = pendingSettlement().carrier;
    if (!base) throw new Error("the fixture settlement pays a carrier");
    const carrier = {
      ...base,
      paymentMethod: null,
      paymentTermDays: null,
      remitToName: null,
      remitAddressLine1: null,
      remitAddressLine2: null,
      remitCity: null,
      remitPostalCode: null,
      remitState: null,
    };

    expect(remittance(pendingSettlement({ carrier }), t)).toEqual({
      withheld: true,
      lines: [],
      method: null,
      terms: "Pays Nov 10",
    });
    expect(remittance(pendingSettlement({ carrier, paymentMethod: "Check" }), t).method).toBe(
      "Check",
    );
    expect(settlementKicker(pendingSettlement({ carrier }), t)).toEqual([
      "CS-2610-0088",
      "Oct 5 – Oct 11",
    ]);
  });
});
