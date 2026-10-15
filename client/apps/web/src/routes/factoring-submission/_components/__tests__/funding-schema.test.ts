import type { FactoringSubmissionDetail } from "@/lib/graphql/factoring";
import { describe, expect, it } from "vitest";
import {
  fundingFormDefaults,
  fundingFormSchema,
  fundingFormToInput,
  reserveMovementFormSchema,
  reserveMovementFormToInput,
  type FundingFormValues,
  type ReserveMovementFormValues,
} from "../funding-schema";

const FUNDED_ON = 1_789_600_000;

function item(
  id: string,
  status: FactoringSubmissionDetail["items"][number]["status"],
  faceMinor: number,
) {
  return {
    id,
    submissionId: "facsub_1",
    invoiceId: `inv_${id}`,
    invoiceNumber: `INV-${id}`,
    customerId: "cus_1",
    position: 1,
    faceMinor,
    expectedAdvanceMinor: Math.round(faceMinor * 0.9),
    expectedFeeMinor: Math.round(faceMinor * 0.03),
    expectedReserveMinor: faceMinor - Math.round(faceMinor * 0.9) - Math.round(faceMinor * 0.03),
    expectedDays: 30,
    packetDocumentId: null,
    status,
    releasedAt: null,
    customer: null,
  };
}

const submission = {
  id: "facsub_1",
  version: 7,
  items: [
    item("a", "Pending", 100_000),
    item("b", "Funded", 50_000),
    item("c", "Pending", 20_000),
    item("d", "Rejected", 10_000),
  ],
} as unknown as FactoringSubmissionDetail;

function issuePaths(values: unknown): string[] {
  const result = fundingFormSchema.safeParse(values);
  return result.success ? [] : result.error.issues.map((issue) => issue.path.join("."));
}

describe("funding form", () => {
  it("lists only the invoices still waiting on the factor, each funded at what was expected", () => {
    expect(fundingFormDefaults(submission, FUNDED_ON)).toEqual({
      fundedOn: FUNDED_ON,
      memo: "",
      items: [
        {
          itemId: "a",
          faceMinor: 100_000,
          outcome: "Funded",
          advanceMinor: 90_000,
          feeMinor: 3_000,
          reason: "",
        },
        {
          itemId: "c",
          faceMinor: 20_000,
          outcome: "Funded",
          advanceMinor: 18_000,
          feeMinor: 600,
          reason: "",
        },
      ],
    });
  });

  it("sends the decided invoices only: funded with amounts, rejected with a trimmed reason", () => {
    const values: FundingFormValues = {
      fundedOn: FUNDED_ON,
      memo: "  Remittance 4471 ",
      items: [
        {
          itemId: "a",
          faceMinor: 100_000,
          outcome: "Funded",
          advanceMinor: 89_500,
          feeMinor: 3_500,
          reason: "ignored",
        },
        {
          itemId: "c",
          faceMinor: 20_000,
          outcome: "Rejected",
          advanceMinor: 18_000,
          feeMinor: 600,
          reason: "  Debtor over limit ",
        },
        {
          itemId: "e",
          faceMinor: 5_000,
          outcome: "Undecided",
          advanceMinor: 4_500,
          feeMinor: 150,
          reason: "",
        },
      ],
    };

    expect(fundingFormToInput(submission, values)).toEqual({
      submissionId: "facsub_1",
      version: 7,
      fundedOn: FUNDED_ON,
      items: [
        { itemId: "a", outcome: "Funded", advanceMinor: 89_500, feeMinor: 3_500 },
        { itemId: "c", outcome: "Rejected", reason: "Debtor over limit" },
      ],
      memo: "Remittance 4471",
    });
  });

  it("asks nothing of the server while no invoice is decided or no day is set", () => {
    const undecided = fundingFormDefaults(submission, FUNDED_ON);
    undecided.items = undecided.items.map((row) => ({ ...row, outcome: "Undecided" }));
    expect(fundingFormToInput(submission, undecided)).toBeNull();
    expect(fundingFormToInput(submission, { ...fundingFormDefaults(submission, 0) })).toBeNull();
    expect(issuePaths(undecided)).toEqual(["items"]);
  });

  it("refuses a rejection without a reason and a funding above the invoice", () => {
    const values = fundingFormDefaults(submission, FUNDED_ON);
    values.items[0] = { ...values.items[0], advanceMinor: 98_000, feeMinor: 3_000 };
    values.items[1] = { ...values.items[1], outcome: "Rejected", reason: "   " };

    expect(issuePaths(values)).toEqual(["items.0.feeMinor", "items.1.reason"]);
  });

  it("accepts a funding of exactly the invoice, with no reserve left", () => {
    const values = fundingFormDefaults(submission, FUNDED_ON);
    values.items[0] = { ...values.items[0], advanceMinor: 97_000, feeMinor: 3_000 };

    expect(issuePaths(values)).toEqual([]);
  });
});

function movement(overrides: Partial<ReserveMovementFormValues>): ReserveMovementFormValues {
  return {
    action: "release",
    occurredOn: FUNDED_ON,
    overrideCash: false,
    cashMinor: 7_000,
    reason: "",
    ...overrides,
  };
}

describe("reserve movement form", () => {
  it("leaves the amount to the server's default unless the person says otherwise", () => {
    expect(reserveMovementFormToInput(submission, "b", movement({}))).toEqual({
      submissionId: "facsub_1",
      itemId: "b",
      version: 7,
      occurredOn: FUNDED_ON,
      cashMinor: null,
      reason: null,
    });
    expect(
      reserveMovementFormToInput(
        submission,
        "b",
        movement({ action: "chargeback", overrideCash: true, cashMinor: 0, reason: " Unpaid " }),
      ),
    ).toMatchObject({ cashMinor: 0, reason: "Unpaid" });
  });

  it("never sends an amount for a reversal, which moves the opposite of the chargeback", () => {
    expect(
      reserveMovementFormToInput(
        submission,
        "b",
        movement({ action: "reverse", overrideCash: true, cashMinor: 5_000, reason: "Withdrawn" }),
      ),
    ).toMatchObject({ cashMinor: null, reason: "Withdrawn" });
  });

  it("requires a reason for a chargeback and a reversal but not for a release", () => {
    const paths = (values: ReserveMovementFormValues) => {
      const result = reserveMovementFormSchema.safeParse(values);
      return result.success ? [] : result.error.issues.map((issue) => issue.path.join("."));
    };

    expect(paths(movement({ action: "release" }))).toEqual([]);
    expect(paths(movement({ action: "chargeback" }))).toEqual(["reason"]);
    expect(paths(movement({ action: "reverse", reason: "  " }))).toEqual(["reason"]);
  });
});
