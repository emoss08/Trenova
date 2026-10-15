import { describe, expect, it } from "vitest";
import {
  agreementFormToInput,
  agreementToFormValues,
  emptyAgreementValues,
  factoringAgreementFormSchema,
  feeTierStarts,
  type FactoringAgreementFormValues,
} from "../factoring-agreement-schema";
import type { FactoringAgreement } from "@/lib/graphql/factoring";

const START = 1_790_000_000;

function values(overrides: Partial<FactoringAgreementFormValues> = {}) {
  return { ...emptyAgreementValues(START), ...overrides };
}

function issuePaths(input: FactoringAgreementFormValues): string[] {
  const result = factoringAgreementFormSchema.safeParse(input);
  return result.success ? [] : result.error.issues.map((issue) => issue.path.join("."));
}

describe("factoring agreement form", () => {
  it("accepts the default terms a new agreement opens with", () => {
    expect(issuePaths(values())).toEqual([]);
  });

  it("starts each fee band where the one before it ends", () => {
    expect(
      feeTierStarts([{ daysTo: 15 }, { daysTo: 30 }, { daysTo: 60 }, { daysTo: null }]),
    ).toEqual([0, 15, 30, 60]);
  });

  it("refuses an open-ended band anywhere but last", () => {
    expect(
      issuePaths(
        values({
          feeTiers: [
            { daysTo: null, ratePercent: 3, flatMinor: 0 },
            { daysTo: 60, ratePercent: 5, flatMinor: 0 },
          ],
        }),
      ),
    ).toContain("feeTiers.0.daysTo");
  });

  it("refuses a band that ends on or before the day it starts", () => {
    expect(
      issuePaths(
        values({
          feeTiers: [
            { daysTo: 30, ratePercent: 3, flatMinor: 0 },
            { daysTo: 30, ratePercent: 5, flatMinor: 0 },
          ],
        }),
      ),
    ).toEqual(["feeTiers.1.daysTo"]);
  });

  it("refuses an advance and reserve that together exceed the invoice", () => {
    expect(issuePaths(values({ advanceRatePercent: 95, reserveRatePercent: 10 }))).toEqual([
      "reserveRatePercent",
    ]);
  });

  it("refuses an end date that is not after the start", () => {
    expect(issuePaths(values({ effectiveTo: START }))).toEqual(["effectiveTo"]);
  });

  it("sends percentages as basis points, contiguous bands, the clock time as minutes and document ids", () => {
    const input = agreementFormToInput(
      "fac_1",
      values({
        advanceRatePercent: 92.5,
        reserveRatePercent: 7.5,
        submissionTime: "14:15",
        notes: "  ",
        feeTiers: [
          { daysTo: 30, ratePercent: 2.75, flatMinor: 500 },
          { daysTo: 45, ratePercent: 3.5, flatMinor: 0 },
          { daysTo: null, ratePercent: 5, flatMinor: 0 },
        ],
        documentTypes: ["dt_1", { id: "dt_2", name: "Proof of delivery" }],
      }),
    );

    expect(input).toMatchObject({
      factoringCompanyId: "fac_1",
      effectiveFrom: START,
      effectiveTo: null,
      advanceRateBp: 9250,
      reserveRateBp: 750,
      submissionTime: 855,
      notes: null,
      documentTypeIds: ["dt_1", "dt_2"],
    });
    expect(input.feeTiers).toEqual([
      { daysFrom: 0, daysTo: 30, rateBp: 275, flatMinor: 500 },
      { daysFrom: 30, daysTo: 45, rateBp: 350, flatMinor: 0 },
      { daysFrom: 45, daysTo: null, rateBp: 500, flatMinor: 0 },
    ]);
  });

  it("opens a saved agreement with its bands in day order and its rates as percentages", () => {
    const agreement = {
      id: "facagr_1",
      factoringCompanyId: "fac_1",
      status: "Active",
      effectiveFrom: START,
      effectiveTo: null,
      advanceRateBp: 9000,
      reserveRateBp: 1000,
      minimumFeeMinor: 1500,
      chargebackWindowDays: 90,
      recourse: "NonRecourse",
      accountingTreatment: "SecuredBorrowing",
      fundingDays: 2,
      submissionTime: 540,
      dueFromFactorAccountId: null,
      feeExpenseAccountId: "gla_fee",
      liabilityAccountId: "gla_liab",
      cashAccountId: null,
      notes: null,
      activatedAt: START,
      endedAt: null,
      version: 3,
      createdAt: START,
      updatedAt: START,
      feeTiers: [
        { id: "t2", daysFrom: 30, daysTo: null, rateBp: 500, flatMinor: 0 },
        { id: "t1", daysFrom: 0, daysTo: 30, rateBp: 300, flatMinor: 250 },
      ],
      documentTypes: [{ id: "dt_1", code: "POD", name: "Proof of delivery", color: "" }],
    } satisfies FactoringAgreement;

    expect(agreementToFormValues(agreement)).toEqual({
      effectiveFrom: START,
      effectiveTo: null,
      advanceRatePercent: 90,
      reserveRatePercent: 10,
      minimumFeeMinor: 1500,
      chargebackWindowDays: 90,
      recourse: "NonRecourse",
      accountingTreatment: "SecuredBorrowing",
      fundingDays: 2,
      submissionTime: "09:00",
      notes: "",
      feeTiers: [
        { daysTo: 30, ratePercent: 3, flatMinor: 250 },
        { daysTo: null, ratePercent: 5, flatMinor: 0 },
      ],
      documentTypes: ["dt_1"],
      dueFromFactorAccountId: null,
      feeExpenseAccountId: "gla_fee",
      liabilityAccountId: "gla_liab",
      cashAccountId: null,
    });
  });

  it("sends each posting account it was given and null for one left empty", () => {
    const input = agreementFormToInput(
      "fac_1",
      values({
        accountingTreatment: "Sale",
        dueFromFactorAccountId: "gla_due",
        feeExpenseAccountId: "gla_fee",
        liabilityAccountId: "",
        cashAccountId: null,
      }),
    );

    expect(input).toMatchObject({
      dueFromFactorAccountId: "gla_due",
      feeExpenseAccountId: "gla_fee",
      liabilityAccountId: null,
      cashAccountId: null,
    });
  });

  it("starts a new agreement with no posting accounts, so the organization's apply", () => {
    expect(emptyAgreementValues(START)).toMatchObject({
      dueFromFactorAccountId: null,
      feeExpenseAccountId: null,
      liabilityAccountId: null,
      cashAccountId: null,
    });
  });
});
