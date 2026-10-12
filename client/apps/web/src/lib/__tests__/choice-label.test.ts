import { describe, expect, it } from "vitest";
import { choiceLabel, customerPaymentTermChoices } from "@/lib/choices";

describe("choiceLabel", () => {
  it("names a value by its choice's label", () => {
    expect(choiceLabel(customerPaymentTermChoices, "Net30")).toBe("Net 30");
    expect(choiceLabel(customerPaymentTermChoices, "DueOnReceipt")).toBe("Due on receipt");
  });

  it("falls back to the value itself when no choice has it", () => {
    expect(choiceLabel(customerPaymentTermChoices, "Net120" as never)).toBe("Net120");
  });
});
