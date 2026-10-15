import { formatMessage } from "@trenova/shared/i18n/format-message";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { describe, expect, it } from "vitest";
import { invoiceEdiSendBlocker, invoiceEmailSendBlocker } from "../invoice-actions";

const t: TranslateFn = (message, ...args) => formatMessage("en", message ?? "", args);

const posted = { status: "Posted", sendStatus: "Sent", pdfDocumentId: "doc_1" };
const readyPlan = { errors: [] };

describe("invoiceEmailSendBlocker", () => {
  it("lets a posted invoice with its PDF and a clean plan be emailed", () => {
    expect(invoiceEmailSendBlocker(posted, readyPlan, false, t)).toBeNull();
  });

  it("never emails a draft or a voided invoice, whatever its plan says", () => {
    expect(invoiceEmailSendBlocker({ ...posted, status: "Draft" }, readyPlan, false, t)).toBe(
      "Post the invoice before sending it. Draft invoices are never emailed.",
    );
    expect(invoiceEmailSendBlocker({ ...posted, status: "Voided" }, readyPlan, false, t)).toBe(
      "A voided invoice is never sent.",
    );
  });

  it("waits for the plan, then for the PDF, then for a send already running", () => {
    expect(invoiceEmailSendBlocker(posted, undefined, true, t)).toBe(
      "Checking invoice email configuration.",
    );
    expect(invoiceEmailSendBlocker({ ...posted, pdfDocumentId: null }, readyPlan, false, t)).toBe(
      "Generate the invoice PDF before sending.",
    );
    expect(invoiceEmailSendBlocker({ ...posted, sendStatus: "Sending" }, readyPlan, false, t)).toBe(
      "This invoice is already being sent. Wait for that send to finish.",
    );
    expect(invoiceEmailSendBlocker(posted, null, false, t)).toBe("Send plan is unavailable.");
  });

  it("names every error on the plan", () => {
    expect(
      invoiceEmailSendBlocker(
        posted,
        { errors: ["No billing email", "The PDF is out of date"] },
        false,
        t,
      ),
    ).toBe("No billing email; The PDF is out of date");
  });
});

describe("invoiceEdiSendBlocker", () => {
  const plan = { enabled: true, blockers: [], status: "Sent" };

  it("lets a posted invoice's 210 go again once the last one is done", () => {
    expect(invoiceEdiSendBlocker("Posted", plan, t)).toBeNull();
  });

  it("puts the plan's own blocker ahead of anything the invoice says", () => {
    expect(
      invoiceEdiSendBlocker("Draft", { ...plan, blockers: ["No 210 document profile"] }, t),
    ).toBe("No 210 document profile");
    expect(invoiceEdiSendBlocker("Posted", { ...plan, enabled: false }, t)).toBe(
      "EDI invoicing is not enabled for this customer.",
    );
  });

  it("waits for the plan, for posting, and for a send already on its way", () => {
    expect(invoiceEdiSendBlocker("Posted", undefined, t)).toBe(
      "Checking the customer's EDI configuration.",
    );
    expect(invoiceEdiSendBlocker("Draft", plan, t)).toBe(
      "Only a posted invoice can be sent by EDI.",
    );
    expect(invoiceEdiSendBlocker("Posted", { ...plan, status: "Queued" }, t)).toBe(
      "An EDI send for this invoice is already in progress.",
    );
  });
});
