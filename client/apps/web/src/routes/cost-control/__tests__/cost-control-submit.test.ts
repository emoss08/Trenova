import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { describe, expect, it } from "vitest";
import type { CostControlFormValues } from "@/types/cost-control";
import {
  costControlFieldPath,
  costControlRefusal,
  costControlSave,
  prepareCostControl,
  toCostControlFormValues,
} from "../cost-control-submit";
import { savedCostControl } from "./cost-control-fixture";

const saved = toCostControlFormValues(savedCostControl);

function withCategory(
  values: CostControlFormValues,
  index: number,
  change: Partial<CostControlFormValues["categories"][number]>,
): CostControlFormValues {
  return {
    ...values,
    categories: values.categories.map((category, at) =>
      at === index ? { ...category, ...change } : category,
    ),
  };
}

describe("toCostControlFormValues", () => {
  it("reads decimals as numbers and GL account links as their account ids", () => {
    const values = toCostControlFormValues({
      ...savedCostControl,
      targetMarginPercent: "12.5",
      categories: [
        {
          ...savedCostControl.categories[0]!,
          rateSource: "Override",
          overrideRatePerMile: "0.9",
          glAccounts: [
            { id: "ccga_1", glAccountId: "gla_1", accountCode: "6000", accountName: "Wages" },
            { id: "ccga_2", glAccountId: "gla_2", accountCode: "6010", accountName: "Bonus" },
          ],
        },
      ],
    });

    expect(values.milesPerGallon).toBe(6.5);
    expect(values.targetMarginPercent).toBe(12.5);
    expect(values.categories).toEqual([
      {
        id: "ccat_00",
        category: "DriverWages",
        name: "Driver Wages",
        costBehavior: "Variable",
        rateSource: "Override",
        benchmarkRatePerMile: "0.82",
        overrideRatePerMile: 0.9,
        isActive: true,
        glAccountIds: ["gla_1", "gla_2"],
        version: 1,
      },
    ]);
  });

  it("keeps a withheld target margin empty", () => {
    expect(saved.targetMarginPercent).toBeNull();
  });
});

describe("prepareCostControl", () => {
  it("sends an inactive category left on an override with no rate back to the benchmark", () => {
    const values = withCategory(saved, 4, {
      isActive: false,
      rateSource: "Override",
      overrideRatePerMile: null,
    });

    expect(prepareCostControl(values).categories[4]).toEqual({
      ...values.categories[4],
      rateSource: "Benchmark",
      overrideRatePerMile: null,
    });
  });

  it("keeps an inactive category's override when it has a rate", () => {
    const values = withCategory(saved, 4, {
      isActive: false,
      rateSource: "Override",
      overrideRatePerMile: 0.3,
    });

    expect(prepareCostControl(values).categories[4]).toEqual(values.categories[4]);
  });

  it("keeps an active category's override as entered", () => {
    const values = withCategory(saved, 4, { rateSource: "Override", overrideRatePerMile: null });

    expect(prepareCostControl(values).categories[4]!.rateSource).toBe("Override");
  });

  it("sends a hidden rolling window as it is, which the server checks only while GL actuals are on", () => {
    expect(prepareCostControl({ ...saved, glRollingMonths: 0 }).glRollingMonths).toBe(0);
    expect(prepareCostControl({ ...saved, glRollingMonths: 13 }).glRollingMonths).toBe(13);
  });

  it("sends hidden planned monthly miles as they are, which the server checks only while GL actuals are on", () => {
    expect(prepareCostControl({ ...saved, plannedMonthlyMiles: 0 }).plannedMonthlyMiles).toBe(0);
    expect(prepareCostControl({ ...saved, plannedMonthlyMiles: 9000 }).plannedMonthlyMiles).toBe(
      9000,
    );
  });

  it("leaves the GL actuals settings alone while GL actuals are on", () => {
    const values = {
      ...saved,
      glActualsEnabled: true,
      glRollingMonths: 0,
      plannedMonthlyMiles: 0,
    };

    expect(prepareCostControl(values)).toEqual(values);
  });
});

describe("costControlSave", () => {
  it("sends nothing when nothing changed", () => {
    expect(costControlSave(saved, saved)).toBeNull();
  });

  it("sends the control's settings in the server's input shape when one changed", () => {
    const values = { ...saved, milesPerGallon: 7.25, targetMarginPercent: 15 };

    expect(costControlSave(values, saved)).toEqual({
      input: {
        control: {
          fuelIndexId: null,
          useLiveFuelPrice: false,
          milesPerGallon: "7.25",
          includeDeadheadMiles: true,
          glActualsEnabled: false,
          glRollingMonths: 3,
          plannedMonthlyMiles: null,
          targetMarginPercent: "15",
          version: 2,
        },
        categories: [],
      },
      categoryIndexes: [],
    });
  });

  it("sends only the categories whose rate, state or accounts changed, with their places", () => {
    let values = withCategory(saved, 1, { rateSource: "Override", overrideRatePerMile: 0.25 });
    values = withCategory(values, 5, { isActive: false });
    values = withCategory(values, 8, { rateSource: "GLActual", glAccountIds: ["gla_9"] });

    expect(costControlSave(values, saved)).toEqual({
      input: {
        control: null,
        categories: [
          {
            id: "ccat_01",
            rateSource: "Override",
            overrideRatePerMile: "0.25",
            isActive: true,
            glAccountIds: [],
            version: 1,
          },
          {
            id: "ccat_05",
            rateSource: "Benchmark",
            overrideRatePerMile: null,
            isActive: false,
            glAccountIds: [],
            version: 1,
          },
          {
            id: "ccat_08",
            rateSource: "GLActual",
            overrideRatePerMile: null,
            isActive: true,
            glAccountIds: ["gla_9"],
            version: 1,
          },
        ],
      },
      categoryIndexes: [1, 5, 8],
    });
  });

  it("sends the control and its categories in the one save", () => {
    const values = withCategory({ ...saved, includeDeadheadMiles: false }, 2, { isActive: false });

    const change = costControlSave(values, saved);

    expect(change?.input.control?.includeDeadheadMiles).toBe(false);
    expect(change?.input.categories.map((category) => category.id)).toEqual(["ccat_02"]);
    expect(change?.categoryIndexes).toEqual([2]);
  });

  it("treats the same GL accounts in another order as unchanged", () => {
    const original = withCategory(saved, 0, { glAccountIds: ["gla_1", "gla_2"] });
    const values = withCategory(saved, 0, { glAccountIds: ["gla_2", "gla_1"] });

    expect(costControlSave(values, original)).toBeNull();
  });
});

describe("costControlFieldPath", () => {
  it("puts the control's settings at the form's root", () => {
    expect(costControlFieldPath(["control", "glRollingMonths"], [])).toEqual(["glRollingMonths"]);
  });

  it("puts a category the save names by its place back where the form holds it", () => {
    expect(costControlFieldPath(["categories", 1, "glAccountIds"], [3, 7])).toEqual([
      "categories",
      7,
      "glAccountIds",
    ]);
  });

  it("leaves a path it cannot place as the server sent it", () => {
    expect(costControlFieldPath(["categories", 4, "id"], [3])).toEqual(["categories", 4, "id"]);
    expect(costControlFieldPath(["categories"], [3])).toEqual(["categories"]);
  });
});

describe("costControlRefusal", () => {
  const refused = () =>
    new GraphQLRequestError({
      kind: "graphql",
      message: "Validation failed",
      status: 422,
      graphQLErrors: [
        {
          message: "Validation failed",
          type: "validation-error",
          extensions: {},
          errors: [
            {
              path: ["categories", 1, "glAccountIds"],
              message: "Map at least one GL account to use GL actuals for this category",
              code: "REQUIRED",
            },
            {
              path: ["control", "glRollingMonths"],
              message: "GL rolling months must be between 1 and 12",
              code: "INVALID",
            },
          ],
        },
      ],
    });

  it("moves each field error to where the form shows it", () => {
    const error = costControlRefusal(refused(), [2, 6]) as GraphQLRequestError;

    expect(error).toBeInstanceOf(GraphQLRequestError);
    expect(error.getFieldErrors().map((fieldError) => fieldError.field)).toEqual([
      "categories.6.glAccountIds",
      "glRollingMonths",
    ]);
  });

  it("passes any other failure through", () => {
    const failure = new Error("offline");

    expect(costControlRefusal(failure, [0])).toBe(failure);
  });
});
