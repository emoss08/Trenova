import { describe, expect, it } from "vitest";
import type { SelectOption as GraphQLSelectOption } from "@/lib/graphql/select-options";
import { toEDITransactionSetFilterOptions } from "@/lib/edi-transaction-set-options";

const fallback = [
  { value: "204", label: "204" },
  { value: "990", label: "990" },
];

function option(overrides: Partial<GraphQLSelectOption>): GraphQLSelectOption {
  return {
    id: "edits_x12_204",
    label: "204 - Motor Carrier Load Tender",
    description: null,
    meta: { code: "204", standard: "X12", defaultVersion: "004010", status: "Active" },
    ...overrides,
  };
}

describe("toEDITransactionSetFilterOptions", () => {
  it("keys each option by the catalog code, not the catalog id", () => {
    const options = toEDITransactionSetFilterOptions(
      [
        option({}),
        option({
          id: "edits_x12_990",
          label: "990 - Response to Load Tender",
          description: "Carrier accept or decline.",
          meta: { code: "990" },
        }),
      ],
      fallback,
    );

    expect(options).toEqual([
      { value: "204", label: "204 - Motor Carrier Load Tender" },
      {
        value: "990",
        label: "990 - Response to Load Tender",
        description: "Carrier accept or decline.",
      },
    ]);
  });

  it("skips options whose meta carries no usable code and collapses duplicates", () => {
    const options = toEDITransactionSetFilterOptions(
      [
        option({ meta: null }),
        option({ meta: { code: "" } }),
        option({ meta: { code: 204 } }),
        option({ id: "edits_x12_204_dup", meta: { code: "204" } }),
        option({}),
      ],
      fallback,
    );

    expect(options).toEqual([{ value: "204", label: "204 - Motor Carrier Load Tender" }]);
  });

  it("returns a copy of the fallback when the catalog yields nothing usable", () => {
    const empty = toEDITransactionSetFilterOptions([], fallback);
    expect(empty).toEqual(fallback);
    expect(empty).not.toBe(fallback);

    expect(toEDITransactionSetFilterOptions([option({ meta: null })], fallback)).toEqual(fallback);
  });
});
