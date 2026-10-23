import { describe, expect, it } from "vitest";
import { cardDay, cardMoment } from "../../../record-card/record-card-values";
import {
  booksChannel,
  chargeFoot,
  chargeRows,
  creditUses,
  disputeCases,
  ediChannel,
  emailChannel,
  headFacts,
  invoiceFigure,
  invoiceKicker,
  invoiceKind,
  invoiceSubtitle,
  neighborRow,
  invoiceStanding,
  owedSegments,
  remittance,
  settledSummary,
  termsRuler,
  voidNote,
  waitingNote,
} from "../invoice-card";
import {
  chargeGroup,
  consolidatedDraft,
  creditMemo,
  DAY,
  HOUR,
  LONG_TEMPLATE,
  neighbor,
  postedInvoice,
  t,
  voidedInvoice,
} from "./fixtures";

const NOW = DAY.oct14 + 15 * HOUR;

describe("invoiceKind", () => {
  it("reads the bill type as the slip names it", () => {
    expect(invoiceKind(postedInvoice())).toEqual({ kind: "invoice", mark: "INV" });
    expect(invoiceKind(creditMemo())).toEqual({ kind: "credit", mark: "CR" });
    expect(invoiceKind(postedInvoice({ billType: "DebitMemo" }))).toEqual({
      kind: "debit",
      mark: "DR",
    });
  });
});

describe("invoiceKicker", () => {
  it("leads with the number, then what the invoice bills, leaving the terms and PRO to the card", () => {
    expect(invoiceKicker(postedInvoice(), t)).toEqual(["INV-2610-0412", "BOL BOL-55120"]);
    expect(invoiceKicker(consolidatedDraft(), t)).toEqual([
      "INV-2610-0457",
      `${cardDay(DAY.oct1)} – ${cardDay(DAY.oct14)}`,
    ]);
    expect(invoiceKicker(creditMemo(), t)).toEqual(["CM-2610-0031", "Corrects INV-2610-0412"]);
  });
});

describe("invoiceSubtitle", () => {
  it("says what the invoice covers and when it is dated", () => {
    expect(invoiceSubtitle(postedInvoice(), t)).toBe(
      `Invoice for 1 shipment · ${cardDay(DAY.sep2)}`,
    );
    expect(invoiceSubtitle(consolidatedDraft(), t)).toBe(
      `Consolidated invoice for 12 shipments · ${cardDay(DAY.oct1)}`,
    );
    expect(
      invoiceSubtitle(
        postedInvoice({ scope: "Order", orderNumber: "ORD-2610-0118", shipmentCount: 3 }),
        t,
      ),
    ).toBe(`Invoice for order ORD-2610-0118 · ${cardDay(DAY.sep2)}`);
    expect(invoiceSubtitle(creditMemo(), t)).toBe(`Credit memo · ${cardDay(DAY.sep2)}`);
    expect(invoiceSubtitle(postedInvoice({ billType: "DebitMemo", scope: "Memo" }), t)).toBe(
      `Debit memo · ${cardDay(DAY.sep2)}`,
    );
  });
});

describe("headFacts", () => {
  it("names the due date and what is in dispute on an open invoice", () => {
    expect(headFacts(postedInvoice(), NOW, t)).toEqual([
      `Due ${cardDay(DAY.oct2)}`,
      "$185.00 in dispute",
    ]);
  });

  it("counts the days a draft has waited and names a split bill and a factored invoice", () => {
    expect(
      headFacts(consolidatedDraft({ isSplitBill: true, factoringCompanyId: "fc_1" }), NOW, t),
    ).toEqual(["Unposted 13 days", `Due ${cardDay(DAY.nov15)}`, "Split bill", "Factored"]);
  });

  it("names when and by whom an invoice was voided, and nothing owed after", () => {
    expect(headFacts(voidedInvoice(), NOW, t)).toEqual([`Voided ${cardDay(DAY.oct3)} by Mia Chen`]);
    expect(
      headFacts(
        postedInvoice({ settlementStatus: "Paid", openBalance: "0", disputes: [] }),
        NOW,
        t,
      ),
    ).toEqual([]);
  });
});

describe("invoiceStanding", () => {
  it("puts what is late before what is disputed", () => {
    expect(invoiceStanding(postedInvoice(), t)).toEqual({
      label: "12 days past due",
      tone: "warn",
      mark: "dot",
    });
  });

  it("reads a disputed invoice that is not yet due as disputed", () => {
    expect(invoiceStanding(postedInvoice({ daysPastDue: null }), t)).toEqual({
      label: "Disputed",
      tone: "warn",
      mark: "dot",
    });
  });

  it("reads the rest of an invoice's way from its settlement", () => {
    const quiet = { daysPastDue: null, disputeStatus: "None" } as const;
    expect(invoiceStanding(postedInvoice({ ...quiet, settlementStatus: "Paid" }), t)).toEqual({
      label: "Paid",
      tone: "success",
      mark: "check",
    });
    expect(invoiceStanding(postedInvoice(quiet), t).label).toBe("Partly paid");
    expect(invoiceStanding(postedInvoice({ ...quiet, settlementStatus: "Unpaid" }), t).label).toBe(
      "Open",
    );
  });

  it("reads a draft and a voided invoice by their status alone", () => {
    expect(invoiceStanding(consolidatedDraft(), t)).toEqual({
      label: "Draft",
      tone: "draft",
      mark: "dot",
    });
    expect(invoiceStanding(voidedInvoice(), t)).toEqual({
      label: "Voided",
      tone: "muted",
      mark: "dot",
    });
  });

  it("reads a credit memo by how much of it is used", () => {
    expect(invoiceStanding(creditMemo(), t).label).toBe("Partly applied");
    expect(invoiceStanding(creditMemo({ creditRemaining: "0" }), t)).toEqual({
      label: "Fully applied",
      tone: "success",
      mark: "check",
    });
    expect(invoiceStanding(creditMemo({ creditRemaining: "400" }), t).label).toBe(
      "Not applied yet",
    );
  });
});

describe("invoiceFigure", () => {
  it("leads with what is still owed, amber once it is late", () => {
    expect(invoiceFigure(postedInvoice(), t)).toEqual({
      amount: "$591.12",
      caption: "still owed of $3,666.12",
      tone: "owed",
    });
    expect(invoiceFigure(postedInvoice({ daysPastDue: null }), t).tone).toBe("plain");
  });

  it("says a paid invoice's total was paid", () => {
    expect(
      invoiceFigure(
        postedInvoice({ settlementStatus: "Paid", openBalance: "0", daysPastDue: null }),
        t,
      ),
    ).toEqual({ amount: "$3,666.12", caption: "paid in full", tone: "plain" });
  });

  it("names a whole balance by when it is due", () => {
    expect(
      invoiceFigure(
        postedInvoice({ openBalance: "3666.12", settlementStatus: "Unpaid", daysPastDue: null }),
        t,
      ).caption,
    ).toBe(`owed, due ${cardDay(DAY.oct2)}`);
  });

  it("gives a draft its total, a voided invoice its total struck, a credit memo what is left", () => {
    expect(invoiceFigure(consolidatedDraft(), t)).toEqual({
      amount: "$18,420.00",
      caption: "draft total",
      tone: "plain",
    });
    expect(invoiceFigure(voidedInvoice(), t)).toEqual({
      amount: "$2,140.00",
      caption: "voided, nothing owed",
      tone: "struck",
    });
    expect(invoiceFigure(creditMemo(), t)).toEqual({
      amount: "$175.00",
      caption: "credit left of $400.00",
      tone: "plain",
    });
  });
});

describe("owedSegments", () => {
  it("splits the total into what was paid, credited, disputed and is still open", () => {
    const segments = owedSegments(postedInvoice(), t);

    expect(segments.map((segment) => [segment.key, segment.amount, segment.zero])).toEqual([
      ["paid", "$2,850.00", false],
      ["credited", "$225.00", false],
      ["disputed", "$185.00", false],
      ["open", "$406.12", false],
    ]);
    expect(segments.reduce((sum, segment) => sum + segment.share, 0)).toBeCloseTo(1, 5);
    expect(segments[0]?.share).toBeCloseTo(2850 / 3666.12, 5);
  });

  it("leaves a reversed payment and a credit taken back out of what settled it", () => {
    const segments = owedSegments(
      postedInvoice({
        creditApplications: [
          {
            ...postedInvoice().creditApplications[0]!,
            status: "Unapplied",
            unappliedAt: DAY.oct10,
          },
        ],
        disputes: [],
        openBalance: "816.12",
      }),
      t,
    );

    expect(segments.map((segment) => [segment.key, segment.amount, segment.zero])).toEqual([
      ["paid", "$2,850.00", false],
      ["credited", "$0.00", true],
      ["disputed", "$0.00", true],
      ["open", "$816.12", false],
    ]);
  });

  it("never draws more in dispute than is still owed", () => {
    const segments = owedSegments(postedInvoice({ openBalance: "100" }), t);

    expect(segments.find((segment) => segment.key === "disputed")?.amount).toBe("$100.00");
    expect(segments.find((segment) => segment.key === "open")?.zero).toBe(true);
  });

  it("splits a credit memo into what was applied and what is left", () => {
    expect(
      owedSegments(creditMemo(), t).map((segment) => [segment.key, segment.label, segment.amount]),
    ).toEqual([
      ["paid", "Applied", "$225.00"],
      ["open", "Left to apply", "$175.00"],
    ]);
  });
});

describe("termsRuler", () => {
  it("runs a late invoice from its date past its due date to today", () => {
    const ruler = termsRuler(postedInvoice(), NOW, t);

    expect(ruler).toMatchObject({
      kind: "late",
      startLabel: "Invoiced",
      startDay: cardDay(DAY.sep2),
      dueLabel: "Due",
      dueDay: `${cardDay(DAY.oct2)} · Net 30`,
      endLabel: "12 days past due",
      endDay: `Today, ${cardDay(NOW)}`,
    });
    expect(ruler?.duePosition).toBeCloseTo(30 / 42, 2);
  });

  it("runs an invoice not yet due from its date to its due date, today between", () => {
    const ruler = termsRuler(postedInvoice({ daysPastDue: null }), DAY.sep18, t);

    expect(ruler).toMatchObject({
      kind: "running",
      endLabel: "Due in 14 days",
      endDay: `${cardDay(DAY.oct2)} · Net 30`,
    });
    expect(ruler?.nowPosition).toBeCloseTo(16 / 30, 2);
  });

  it("counts a draft's terms from its invoice date, so the days it waits are already running", () => {
    const ruler = termsRuler(consolidatedDraft(), NOW, t);

    expect(ruler).toMatchObject({
      kind: "unposted",
      startLabel: "Invoice date",
      startDay: `${cardDay(DAY.oct1)} · drafted`,
      nowLabel: "13 days unposted",
      endLabel: "Due",
      endDay: `${cardDay(DAY.nov15)} · Net 45`,
      waitedDays: 13,
      termDays: 45,
    });
    expect(ruler?.nowPosition).toBeCloseTo(13 / 45, 2);
  });

  it("draws nothing for an invoice that owes nothing or has no due date", () => {
    expect(
      termsRuler(postedInvoice({ settlementStatus: "Paid", openBalance: "0" }), NOW, t),
    ).toBeNull();
    expect(termsRuler(voidedInvoice(), NOW, t)).toBeNull();
    expect(termsRuler(creditMemo(), NOW, t)).toBeNull();
    expect(termsRuler(postedInvoice({ dueDate: null }), NOW, t)).toBeNull();
  });
});

describe("chargeRows", () => {
  it("names the template that rated the freight and how each accessorial was charged", () => {
    const rows = chargeRows(postedInvoice(), t);

    expect(rows.map((row) => [row.name, row.code, row.amount])).toEqual([
      ["Linehaul", "", "$2,850.00"],
      ["Fuel surcharge", "FSC", "$406.12"],
      ["Detention", "DET", "$225.00"],
      ["Lumper", "LUMP", "$185.00"],
    ]);
    expect(rows[0]?.basis).toEqual({
      kind: "formula",
      templateId: "ft_permile",
      name: LONG_TEMPLATE,
    });
    expect(rows[0]?.detail).toBe("760 mi × $3.75/mi");
    expect(rows.slice(1).map((row) => row.basis)).toEqual([
      { kind: "text", text: "14.25% of $2,850.00" },
      { kind: "text", text: "3 hours × $75.00" },
      { kind: "text", text: "Flat" },
    ]);
  });

  it("says how many shipments share a charge on a consolidated invoice", () => {
    const rows = chargeRows(consolidatedDraft(), t);

    expect(rows.map((row) => row.detail)).toEqual([
      "8 shipments",
      "4 shipments · 1,652.46 cwt × $3.05/cwt",
      "12 shipments",
      "4 shipments",
    ]);
    expect(rows[2]?.basis).toEqual({ kind: "text", text: "11.5% of linehaul" });
    expect(rows[3]?.basis).toEqual({ kind: "text", text: "6 hours × $75.00" });
  });

  it("keeps a template that no longer has an id as a name only", () => {
    const rows = chargeRows(
      postedInvoice({
        chargeSummary: [
          chargeGroup({
            type: "Freight",
            description: "Freight charge",
            formulaTemplateName: "Legacy per mile",
            amount: "900",
          }),
        ],
      }),
      t,
    );

    expect(rows[0]?.basis).toEqual({ kind: "formula", templateId: null, name: "Legacy per mile" });
  });

  it("reads freight with no template, a flat charge billed more than once, and memo lines", () => {
    const rows = chargeRows(
      postedInvoice({
        chargeSummary: [
          chargeGroup({ type: "Freight", description: "Freight charge", amount: "900" }),
          chargeGroup({
            description: "Stop off",
            chargeCode: "STOP",
            chargeMethod: "Flat",
            rate: "69",
            quantity: "2",
            amount: "138",
          }),
          chargeGroup({
            description: "Tarp",
            chargeMethod: "PerUnit",
            quantity: "2",
            amount: "150",
          }),
          chargeGroup({
            type: "Memo",
            description: "Goodwill credit",
            amount: "-50",
            shipmentCount: 0,
          }),
        ],
      }),
      t,
    );

    expect(rows.map((row) => [row.name, row.basis])).toEqual([
      ["Freight charge", { kind: "text", text: "No formula template" }],
      ["Stop off", { kind: "text", text: "2 × $69.00" }],
      ["Tarp", { kind: "text", text: "2 units" }],
      ["Goodwill credit", { kind: "text", text: "" }],
    ]);
  });
});

describe("chargeFoot", () => {
  it("runs an open invoice down from its total to its balance due", () => {
    expect(chargeFoot(postedInvoice(), t)).toEqual([
      { label: "Freight", value: "$2,850.00", kind: "line" },
      { label: "Accessorials", value: "$816.12", kind: "line" },
      { label: "Total", value: "$3,666.12", kind: "total" },
      { label: "Paid", value: "−$2,850.00", kind: "less" },
      { label: "Credited", value: "−$225.00", kind: "less" },
      { label: "Balance due", value: "$591.12", kind: "due" },
    ]);
  });

  it("stops a draft at its total and takes a voided invoice to nothing", () => {
    expect(chargeFoot(consolidatedDraft(), t).map((row) => row.label)).toEqual([
      "Freight",
      "Accessorials",
      "Total",
    ]);
    expect(chargeFoot(voidedInvoice(), t)).toEqual([
      { label: "Freight", value: "$1,850.00", kind: "line" },
      { label: "Accessorials", value: "$290.00", kind: "line" },
      { label: "Total", value: "$2,140.00", kind: "total" },
      { label: "Reversed", value: "−$2,140.00", kind: "less" },
      { label: "Balance due", value: "$0.00", kind: "due" },
    ]);
  });

  it("totals a credit memo as the credit it gives", () => {
    expect(chargeFoot(creditMemo(), t)).toEqual([
      { label: "Credit", value: "−$400.00", kind: "total" },
    ]);
  });
});

describe("remittance", () => {
  it("lists what settled the invoice newest first, a reversed payment struck through", () => {
    const rows = remittance(postedInvoice(), t);

    expect(
      rows.map((row) => [row.kind, row.title, row.tag, row.line, row.amount, row.undone]),
    ).toEqual([
      ["credit", "Credit memo CM-2610-0031", "", `Applied ${cardDay(DAY.oct9)}`, "−$225.00", false],
      [
        "payment",
        "ACH · ref 884120",
        "Short $816.12",
        `Received ${cardDay(DAY.sep30)}`,
        "−$2,850.00",
        false,
      ],
      [
        "payment",
        "Check · ref 1042",
        "",
        `Received ${cardDay(DAY.sep18)} · reversed`,
        "−$3,666.12",
        true,
      ],
    ]);
    expect(rows[0]?.open).toEqual({ entity: "invoice", id: "inv_cm31" });
  });
});

describe("creditUses", () => {
  it("lists the invoices a credit memo settled, one taken back struck through with why", () => {
    expect(
      creditUses(creditMemo(), t).map((row) => [
        row.title,
        row.line,
        row.amount,
        row.undone,
        row.open,
      ]),
    ).toEqual([
      [
        "Invoice INV-2610-0412",
        `Applied ${cardDay(DAY.oct9)}`,
        "$225.00",
        false,
        { entity: "invoice", id: "inv_412" },
      ],
      [
        "Invoice INV-2610-0401",
        `Applied ${cardDay(DAY.oct9)}, taken back ${cardDay(DAY.oct10)} · Applied to the wrong invoice`,
        "$175.00",
        true,
        { entity: "invoice", id: "inv_401" },
      ],
    ]);
  });
});

describe("disputeCases", () => {
  it("holds the open case apart from the closed ones", () => {
    const cases = disputeCases(postedInvoice(), NOW, t);

    expect(cases.open).toEqual({
      reason: "Accessorial disputed",
      amount: "$185.00",
      notes: "Customer says their driver paid the lumper at the dock.",
      opened: `Opened by Dana Ruiz · ${cardMoment(DAY.oct6 + 10 * HOUR)}`,
      age: "8 days open",
    });
    expect(cases.closed).toEqual([
      {
        key: "idp_rate",
        title: "Rate discrepancy · Credit issued",
        amount: "$225.00",
        line: `Resolved by Dana Ruiz · ${cardDay(DAY.oct9)} · Detention hours did not match the gate log`,
      },
    ]);
  });

  it("names a withdrawn case and one closed by no one the card can name", () => {
    const cases = disputeCases(
      postedInvoice({
        disputes: [
          {
            ...postedInvoice().disputes[0]!,
            status: "Withdrawn",
            resolvedAt: DAY.oct9,
            resolvedBy: null,
          },
        ],
      }),
      NOW,
      t,
    );

    expect(cases.open).toBeNull();
    expect(cases.closed[0]?.title).toBe("Accessorial disputed · Withdrawn");
    expect(cases.closed[0]?.line).toBe(`Closed ${cardDay(DAY.oct9)}`);
  });
});

describe("voidNote", () => {
  it("says why and how an invoice was voided, and what became of its freight", () => {
    expect(voidNote(voidedInvoice(), t)).toEqual({
      reason: "Billed to Acme, but the consignee pays on this lane.",
      freight: "Back in the billing queue to bill again",
      when: `${cardMoment(DAY.oct3 + 11 * HOUR)} by Mia Chen`,
    });
    expect(voidNote(voidedInvoice({ voidDisposition: "DoNotRebill", voidedBy: null }), t)).toEqual({
      reason: "Billed to Acme, but the consignee pays on this lane.",
      freight: "Its shipments settled without another bill",
      when: cardMoment(DAY.oct3 + 11 * HOUR),
    });
    expect(voidNote(postedInvoice(), t)).toBeNull();
  });
});

describe("emailChannel", () => {
  it("names who a sent invoice went to from its last attempt", () => {
    expect(
      emailChannel(
        postedInvoice(),
        {
          attempt: { toRecipients: ["ap@acmefoods.com"], sentAt: DAY.sep2 + 9 * HOUR, error: null },
          plan: null,
        },
        t,
      ),
    ).toEqual({
      state: "Sent",
      tone: "ok",
      detail: `${cardMoment(DAY.sep2 + 9 * HOUR)} to ap@acmefoods.com`,
      why: false,
    });
  });

  it("says why an email that failed did not go", () => {
    expect(
      emailChannel(
        postedInvoice({ sendStatus: "Failed" }),
        {
          attempt: { toRecipients: ["ap@acmefoods.com"], sentAt: null, error: "Mailbox full" },
          plan: null,
        },
        t,
      ),
    ).toEqual({ state: "Failed", tone: "warn", detail: "Mailbox full", why: true });
  });

  it("reads an unsent invoice from its send plan: who it goes to, or why it cannot go", () => {
    expect(
      emailChannel(
        consolidatedDraft(),
        { attempt: null, plan: { to: ["ap@tyson.com"], errors: [] } },
        t,
      ),
    ).toEqual({
      state: "Not sent",
      tone: "idle",
      detail: "Goes to ap@tyson.com when sent",
      why: false,
    });
    expect(
      emailChannel(
        consolidatedDraft(),
        { attempt: null, plan: { to: [], errors: ["Tyson Foods has no billing email on file."] } },
        t,
      ),
    ).toEqual({
      state: "Cannot send",
      tone: "warn",
      detail: "Tyson Foods has no billing email on file.",
      why: true,
    });
  });

  it("names a voided invoice that never went as not sent", () => {
    expect(
      emailChannel(voidedInvoice({ sendStatus: "NotSent" }), { attempt: null, plan: null }, t),
    ).toEqual({ state: "Not sent", tone: "idle", detail: "", why: false });
  });
});

describe("ediChannel", () => {
  it("names the partner and the way a sent 210 went", () => {
    expect(ediChannel(postedInvoice(), t)).toEqual({
      state: "Sent",
      tone: "ok",
      detail: `${cardMoment(DAY.sep2 + 9 * HOUR)} to Acme Foods by AS2`,
      why: false,
      messageId: "edim_210",
    });
  });

  it("says why a 210 cannot go, whatever the plan's status", () => {
    expect(ediChannel(consolidatedDraft(), t)).toEqual({
      state: "Cannot send",
      tone: "warn",
      detail: "Tyson Foods has no 210 document profile.",
      why: true,
      messageId: null,
    });
  });

  it("reads a customer who does not take invoices by EDI as not set up", () => {
    const plan = { ...postedInvoice().ediSendPlan, enabled: false, status: "NotSent" as const };
    expect(ediChannel(postedInvoice({ ediSendPlan: plan }), t)).toEqual({
      state: "Not set up",
      tone: "idle",
      detail: "Acme Foods does not take invoices by EDI",
      why: false,
      messageId: null,
    });
  });

  it("keeps the error of a 210 that failed or was set aside", () => {
    const plan = {
      ...postedInvoice().ediSendPlan,
      status: "DeadLettered" as const,
      lastError: "AS2 endpoint refused the connection",
    };
    expect(ediChannel(postedInvoice({ ediSendPlan: plan }), t)).toMatchObject({
      state: "Not delivered",
      tone: "warn",
      detail: "AS2 endpoint refused the connection",
      why: true,
    });
  });
});

describe("booksChannel", () => {
  const synced = {
    providerName: "QuickBooks Online",
    record: {
      id: "acctsr_1",
      status: "Synced" as const,
      syncedAt: DAY.sep2 + 9 * HOUR,
      nextAttemptAt: null,
      resolution: "",
      errorMessage: "",
      attemptCount: 1,
      skippedReason: "",
      externalDocNumber: "1042",
      externalUrl: "https://app.qbo.intuit.com/app/invoice?txnId=1042",
    },
  };

  it("names the accounting system and the number the invoice has there", () => {
    expect(booksChannel(synced, t)).toEqual({
      state: "In QuickBooks Online",
      tone: "ok",
      detail: `As 1042 · synced ${cardDay(DAY.sep2)}`,
      why: false,
      provider: "QuickBooks Online",
      externalUrl: "https://app.qbo.intuit.com/app/invoice?txnId=1042",
      recordId: "acctsr_1",
      needsAction: false,
    });
  });

  it("says why a record is held, and that it waits on someone", () => {
    expect(
      booksChannel(
        {
          providerName: "NetSuite",
          record: {
            ...synced.record,
            status: "Blocked",
            errorMessage: "The customer is not mapped in NetSuite.",
            externalUrl: "",
            externalDocNumber: "",
          },
        },
        t,
      ),
    ).toMatchObject({
      state: "Held from NetSuite",
      tone: "warn",
      detail: "The customer is not mapped in NetSuite.",
      why: true,
      externalUrl: null,
      needsAction: true,
    });
  });

  it("reads a record on its way as in progress", () => {
    expect(
      booksChannel({ providerName: "Xero", record: { ...synced.record, status: "InFlight" } }, t),
    ).toMatchObject({ state: "Sending to Xero", tone: "progress", detail: "", why: false });
  });
});

describe("settledSummary", () => {
  it("counts what still stands and what was undone", () => {
    expect(settledSummary(remittance(postedInvoice(), t), false, t)).toBe("2 applied · 1 reversed");
    expect(settledSummary(creditUses(creditMemo(), t), true, t)).toBe("1 applied · 1 taken back");
    expect(
      settledSummary(remittance(postedInvoice({ paymentApplications: [] }), t), false, t),
    ).toBe("1 applied");
  });
});

describe("neighborRow", () => {
  it("names another invoice by its kind and number, with how it stands", () => {
    expect(neighborRow(neighbor(), t)).toMatchObject({
      title: "Invoice INV-2610-0412",
      line: "Acme Foods · 12 days past due",
      amount: "$3,666.12",
      undone: false,
      open: { entity: "invoice", id: "inv_412" },
    });
    expect(neighborRow(neighbor({ daysPastDue: null }), t).line).toBe(
      "Acme Foods · $591.12 still owed",
    );
    expect(
      neighborRow(
        neighbor({ billType: "CreditMemo", number: "CM-1", totalAmount: "-50", openBalance: "0" }),
        t,
      ),
    ).toMatchObject({ title: "Credit memo CM-1", line: "Acme Foods · Settled", amount: "$50.00" });
    expect(neighborRow(neighbor({ status: "Voided" }), t)).toMatchObject({
      line: "Acme Foods · Voided",
      undone: true,
    });
  });
});

describe("waitingNote", () => {
  it("counts the customer's days a draft has used up while it waits", () => {
    expect(waitingNote({ waitedDays: 13, termDays: 45 }, t)).toBe(
      "Its terms count from the invoice date, so 13 of its 45 days have run while it waits to be posted.",
    );
    expect(waitingNote({ waitedDays: 1, termDays: 30 }, t)).toBe(
      "Its terms count from the invoice date, so 1 of its 30 days has run while it waits to be posted.",
    );
  });
});
