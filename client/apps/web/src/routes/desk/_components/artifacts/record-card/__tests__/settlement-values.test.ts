import { formatMessage } from "@trenova/shared/i18n/format-message";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { Operation } from "@trenova/shared/types/permission";
import { describe, expect, it } from "vitest";
import {
  reachedStep,
  SETTLEMENT_ACTION_OPERATION,
  settlementActions,
  type SettlementLife,
  settlementStanding,
  settlementTrack,
  signoffs,
  trackNote,
} from "../settlement-values";

const t: TranslateFn = (message, ...args) => formatMessage("en", message ?? "", args);

const OCT_5 = 1_791_158_400;
const OCT_10 = 1_791_590_400;
const OCT_13 = 1_791_849_600;
const OCT_17 = 1_792_195_200;

function life(overrides: Partial<SettlementLife> = {}): SettlementLife {
  return {
    status: "Draft",
    payDate: OCT_17,
    createdAt: OCT_5,
    paymentMethod: "",
    paymentReference: "",
    ...overrides,
  };
}

describe("settlementTrack", () => {
  it("ticks the steps behind a settlement and marks the one it stands at", () => {
    expect(settlementTrack("Approved", t).map((step) => [step.label, step.state])).toEqual([
      ["Draft", "done"],
      ["Submitted", "done"],
      ["Approved", "now"],
      ["Posted", "next"],
      ["Paid", "next"],
    ]);
  });

  it("strikes the whole way off for a voided settlement", () => {
    expect(settlementTrack("Voided", t).every((step) => step.state === "off")).toBe(true);
  });
});

describe("settlementActions", () => {
  it("offers the one step that moves each status on, and the steps beside it", () => {
    expect(settlementActions("Draft")).toEqual({
      primary: "submit",
      more: ["recalculate", "void"],
    });
    expect(settlementActions("PendingApproval")).toEqual({
      primary: "approve",
      more: ["reject", "void"],
    });
    expect(settlementActions("Approved")).toEqual({ primary: "post", more: ["void"] });
    expect(settlementActions("Posted")).toEqual({ primary: "pay", more: ["void"] });
  });

  it("offers nothing once a settlement is paid or voided", () => {
    expect(settlementActions("Paid")).toEqual({ primary: null, more: [] });
    expect(settlementActions("Voided")).toEqual({ primary: null, more: [] });
  });

  it("needs the permission the server checks for each step", () => {
    expect(SETTLEMENT_ACTION_OPERATION).toEqual({
      submit: Operation.Submit,
      recalculate: Operation.Update,
      approve: Operation.Approve,
      reject: Operation.Reject,
      post: Operation.Approve,
      pay: Operation.Update,
      void: Operation.Cancel,
    });
  });
});

describe("settlementStanding", () => {
  it("calls a submitted settlement waiting on approval, in amber", () => {
    expect(settlementStanding("PendingApproval", t)).toEqual({
      label: "Waiting on approval",
      tone: "warn",
      mark: "dot",
    });
    expect(settlementStanding("Paid", t)).toEqual({
      label: "Paid",
      tone: "success",
      mark: "check",
    });
    expect(settlementStanding("Voided", t)).toEqual({ label: "Voided", tone: "muted", mark: "x" });
  });
});

describe("trackNote", () => {
  it("names who submitted a settlement and when it pays", () => {
    expect(
      trackNote(
        life({
          status: "PendingApproval",
          submittedAt: OCT_13,
          submittedBy: { name: "Mia Chen" },
        }),
        t,
      ),
    ).toMatch(/^Submitted by Mia Chen Oct 13 · approve it to pay \w+, Oct 17$/u);
  });

  it("says whether a paid settlement went out on its pay date", () => {
    expect(trackNote(life({ status: "Paid", payDate: OCT_10, paidAt: OCT_10 + 3_600 }), t)).toBe(
      "Paid Oct 10 · on its pay date",
    );
    expect(trackNote(life({ status: "Paid", payDate: OCT_10, paidAt: OCT_13 }), t)).toBe(
      "Paid Oct 13 · after its pay date of Oct 10",
    );
  });

  it("says whether a voided settlement had reached the ledger", () => {
    expect(trackNote(life({ status: "Voided" }), t)).toBe(
      "Voided before it was posted · nothing reached the ledger",
    );
    expect(trackNote(life({ status: "Voided", postedAt: OCT_10 }), t)).toBe(
      "Voided after it was posted · its ledger entry was reversed",
    );
  });
});

describe("signoffs", () => {
  it("lists each step passed with who took it, the payment with how", () => {
    expect(
      signoffs(
        life({
          status: "Paid",
          submittedAt: OCT_5,
          submittedBy: { name: "Mia Chen" },
          approvedAt: OCT_10,
          approvedBy: { name: "Dana Ruiz" },
          paidAt: OCT_13,
          paymentMethod: "ACH",
        }),
        t,
      ),
    ).toEqual([
      { key: "submitted", step: "Submitted", who: "Mia Chen", when: "Oct 5" },
      { key: "approved", step: "Approved", who: "Dana Ruiz", when: "Oct 10" },
      { key: "paid", step: "Paid", who: "Someone", when: "Oct 13 · ACH" },
    ]);
  });
});

describe("reachedStep", () => {
  it("names the furthest step a voided settlement got to", () => {
    expect(reachedStep(life({ submittedAt: OCT_5, approvedAt: OCT_10 }), t)).toBe(
      "Approved · never posted",
    );
    expect(reachedStep(life(), t)).toBe("Draft · never submitted");
  });
});
