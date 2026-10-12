import type { AskField, AskInput } from "@/components/assistant/ask-requests";
import { activeMentions } from "@/components/assistant/composer-types";
import { describe, expect, it } from "vitest";
import { askFieldsAnswer, askInputAnswer, askValueReady } from "../ask-answer";

const acme = { type: "customer", id: "cus_01J8ZV4M7Q", label: "Acme Freight" };

describe("askInputAnswer", () => {
  it("sends words as typed, without the spaces around them", () => {
    expect(askInputAnswer({ type: "text" }, "  45 days ")).toEqual({
      text: "45 days",
      mentions: [],
    });
  });

  it("sends a number as typed", () => {
    expect(askInputAnswer({ type: "number" }, "1200.50")).toEqual({
      text: "1200.50",
      mentions: [],
    });
    expect(askInputAnswer({ type: "number" }, "-3")?.text).toBe("-3");
  });

  it("sends a day as YYYY-MM-DD", () => {
    expect(askInputAnswer({ type: "date" }, "2026-10-14")?.text).toBe("2026-10-14");
  });

  // datetime-local holds the person's own clock with a T, and may carry seconds.
  it("sends a moment as YYYY-MM-DD HH:mm on the person's own clock", () => {
    expect(askInputAnswer({ type: "datetime" }, "2026-10-14T07:30")?.text).toBe("2026-10-14 07:30");
    expect(askInputAnswer({ type: "datetime" }, "2026-10-14T07:30:15")?.text).toBe(
      "2026-10-14 07:30",
    );
  });

  // The model must read a picked record exactly as it reads one the person
  // named with @ in the composer: the @label in the words, and the record
  // beside them, which the composer's own filter then keeps.
  it("writes a record as the composer writes a mention, and sends the record with it", () => {
    const answer = askInputAnswer({ type: "record", recordKind: "customer" }, acme);

    expect(answer).toEqual({ text: "@Acme Freight", mentions: [acme] });
    expect(activeMentions(answer?.text ?? "", answer?.mentions ?? [])).toEqual([acme]);
  });

  it("names a record with no label by its id", () => {
    const unnamed = { type: "customer", id: "cus_01J8ZV4M7Q", label: "" };
    expect(askInputAnswer({ type: "record", recordKind: "customer" }, unnamed)).toEqual({
      text: "@cus_01J8ZV4M7Q",
      mentions: [{ ...unnamed, label: "cus_01J8ZV4M7Q" }],
    });
  });

  it("sends nothing while the value is empty or not yet of its type", () => {
    expect(askInputAnswer({ type: "text" }, "   ")).toBeNull();
    expect(askInputAnswer({ type: "text" }, null)).toBeNull();
    expect(askInputAnswer({ type: "number" }, "")).toBeNull();
    expect(askInputAnswer({ type: "number" }, "twelve")).toBeNull();
    expect(askInputAnswer({ type: "date" }, "2026-02-30")).toBeNull();
    expect(askInputAnswer({ type: "date" }, "10/14/2026")).toBeNull();
    expect(askInputAnswer({ type: "datetime" }, "2026-10-14")).toBeNull();
    expect(askInputAnswer({ type: "datetime" }, "2026-13-01T07:30")).toBeNull();
    expect(askInputAnswer({ type: "record", recordKind: "customer" }, "Acme")).toBeNull();
    expect(askInputAnswer({ type: "text" }, acme)).toBeNull();
  });
});

describe("askValueReady", () => {
  it("is ready once the value is entered and of its type", () => {
    expect(askValueReady({ type: "number" }, "12")).toBe(true);
    expect(askValueReady({ type: "number" }, "")).toBe(false);
  });
});

function field(key: string, label: string, input: AskInput, optional = false): AskField {
  return { ...input, key, label, hint: "", optional };
}

const customer: AskInput = { type: "record", recordKind: "customer" };

const fields: AskField[] = [
  field("customer", "Customer", customer),
  field("pickup", "Pickup", { type: "datetime" }),
  field("weight", "Weight", { type: "number" }, true),
  field("note", "Note", { type: "text" }, true),
];

describe("askFieldsAnswer", () => {
  it("sends one line per value, Label: value, in the form's order", () => {
    expect(
      askFieldsAnswer(fields, {
        note: "Dock 4",
        pickup: "2026-10-14T07:30",
        customer: acme,
        weight: "42000",
      }),
    ).toEqual({
      text: "Customer: @Acme Freight\nPickup: 2026-10-14 07:30\nWeight: 42000\nNote: Dock 4",
      mentions: [acme],
    });
  });

  it("leaves out an optional value left empty", () => {
    expect(
      askFieldsAnswer(fields, { customer: acme, pickup: "2026-10-14T07:30", weight: " " }),
    ).toEqual({ text: "Customer: @Acme Freight\nPickup: 2026-10-14 07:30", mentions: [acme] });
  });

  it("sends nothing until every value the form needs is entered", () => {
    expect(askFieldsAnswer(fields, { customer: acme })).toBeNull();
    expect(askFieldsAnswer(fields, { customer: acme, pickup: "2026-10-14" })).toBeNull();
    expect(askFieldsAnswer(fields, {})).toBeNull();
  });

  it("sends nothing for a form of optional values with none entered", () => {
    const optional = fields.filter((field) => field.optional);
    expect(askFieldsAnswer(optional, {})).toBeNull();
    expect(askFieldsAnswer(optional, { note: "Call first" })?.text).toBe("Note: Call first");
  });

  it("sends a record named twice once", () => {
    const twice = [field("from", "Bill to", customer), field("to", "Ship to", customer)];
    expect(askFieldsAnswer(twice, { from: acme, to: acme })).toEqual({
      text: "Bill to: @Acme Freight\nShip to: @Acme Freight",
      mentions: [acme],
    });
  });
});
