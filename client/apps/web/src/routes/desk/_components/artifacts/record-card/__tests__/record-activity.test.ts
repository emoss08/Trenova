import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import { activityActor, activitySentence, fieldName } from "../record-activity";

const t = ((message: string, ...args: unknown[]) =>
  message
    .replace(
      /\{(\d+), plural, one \{([^}]*)\} other \{([^}]*)\}\}/gu,
      (_, index, one: string, other: string) => {
        const count = Number(args[Number(index)]);
        return (count === 1 ? one : other).replace("#", String(count));
      },
    )
    .replace(/\{(\d+)\}/gu, (_, index) => String(args[Number(index)]))) as TranslateFn;

describe("fieldName", () => {
  it("reads a field path as words a sentence can hold, from its last part, keeping short forms", () => {
    expect(fieldName("poNumber")).toBe("PO number");
    expect(fieldName("charges.0.amount")).toBe("amount");
    expect(fieldName("bill_to_customer_id")).toBe("bill to customer ID");
    expect(fieldName("totalAmount")).toBe("total amount");
  });
});

describe("activitySentence", () => {
  it("names the fields a change touched, counting the rest past three", () => {
    expect(
      activitySentence(
        { operation: "update", changes: { status: { from: "Draft", to: "Confirmed" } } },
        t,
      ),
    ).toBe("Changed status");
    expect(
      activitySentence(
        {
          operation: "UPDATE",
          changes: {
            status: { from: "a", to: "b" },
            poNumber: { from: "1", to: "2" },
            bol: { from: null, to: "B" },
            totalAmount: { from: "1", to: "2" },
            currencyCode: { from: "USD", to: "CAD" },
          },
        },
        t,
      ),
    ).toMatch(/^Changed .+, .+, .+ and 2 more$/u);
  });

  it("reads every other operation in the past tense, and an update with no fields as updated", () => {
    expect(activitySentence({ operation: "create", changes: null }, t)).toBe("Created");
    expect(activitySentence({ operation: "approve", changes: null }, t)).toBe("Approved");
    expect(activitySentence({ operation: "update", changes: {} }, t)).toBe("Updated");
    expect(activitySentence({ operation: "rebill", changes: null }, t)).toBe("Rebill");
  });
});

describe("activityActor", () => {
  it("names the person, or what acted when no person did", () => {
    expect(
      activityActor({ user: { id: "u", name: " Mia Chen " }, principalType: "session_user" }, t),
    ).toBe("Mia Chen");
    expect(activityActor({ user: null, principalType: "api_key" }, t)).toBe("An API key");
    expect(activityActor({ user: null, principalType: "capture_device" }, t)).toBe(
      "A capture device",
    );
    expect(activityActor({ user: null, principalType: "system" }, t)).toBe("Trenova");
  });
});
