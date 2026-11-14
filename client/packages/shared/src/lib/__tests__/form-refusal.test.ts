import { describe, expect, it } from "vitest";
import { refusalMessages } from "../form-refusal";

const refused = (message: string, types?: Record<string, string>) => ({
  type: "validation",
  message,
  ...(types ? { types } : {}),
});

describe("the messages of a refused save", () => {
  it("are none while the form holds no errors", () => {
    expect(refusalMessages({})).toEqual([]);
  });

  it("put the form-wide message first, then each field's in the form's order", () => {
    expect(
      refusalMessages({
        defaultApAccountId: refused("Default AP account is required"),
        root: refused("The period is closed"),
        defaultExpenseAccountId: refused("Default expense account is required"),
      }),
    ).toEqual([
      "The period is closed",
      "Default AP account is required",
      "Default expense account is required",
    ]);
  });

  it("include everything the server said about one field, not only what the field shows", () => {
    expect(
      refusalMessages({
        autoPostSourceEvents: refused("At least one event is required", {
          0: "At least one event is required",
          1: "Events must include InvoicePosted",
          2: "Events must include VendorBillPosted",
        }),
      }),
    ).toEqual([
      "At least one event is required",
      "Events must include InvoicePosted",
      "Events must include VendorBillPosted",
    ]);
  });

  it("reach fields inside lists and groups", () => {
    expect(
      refusalMessages({
        lines: [undefined, { glAccountId: refused("GL account 4000 is inactive") }],
        address: { city: refused("City is required") },
      }),
    ).toEqual(["GL account 4000 is inactive", "City is required"]);
  });

  it("leave out what the form's own validation found, which the fields already say", () => {
    expect(
      refusalMessages({
        name: { type: "too_small", message: "Name is required" },
        scacCode: refused("SCAC code is already in use"),
      }),
    ).toEqual(["SCAC code is already in use"]);
  });

  it("say a message given for two fields once", () => {
    expect(
      refusalMessages({
        startDate: refused("Dates must not overlap"),
        endDate: refused("Dates must not overlap"),
      }),
    ).toEqual(["Dates must not overlap"]);
  });
});
