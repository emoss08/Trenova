import { describe, expect, it } from "vitest";
import { fillTemplate, renderTemplate, sharedTemplate } from "../shared-template";

function podMessage(driver: string, pro: string, bol: string, day: string, customer: string) {
  return (
    `Hi ${driver}, please send the signed proof of delivery for PRO ${pro} (BOL ${bol}), ` +
    `delivered on Aug ${day}, 2026 for ${customer}. A clear photo of the signed paperwork in Dash is fine.`
  );
}

describe("sharedTemplate", () => {
  it("finds the wording messages from one template share and what fills each place", () => {
    const texts = [
      podMessage("Laura", "S1001", "B77", "15", "Simpson & Jones Industries"),
      podMessage("Theresa", "S1002", "B78", "16", "Acme Foods"),
      podMessage("María José", "S1003", "B79", "17", "Northwind"),
    ];

    const template = sharedTemplate(texts);

    expect(template).not.toBeNull();
    expect(template?.parts).toEqual([
      { kind: "text", text: "Hi " },
      { kind: "slot", slot: 0 },
      { kind: "text", text: ", please send the signed proof of delivery for PRO " },
      { kind: "slot", slot: 1 },
      { kind: "text", text: " (BOL " },
      { kind: "slot", slot: 2 },
      { kind: "text", text: "), delivered on Aug " },
      { kind: "slot", slot: 3 },
      { kind: "text", text: ", 2026 for " },
      { kind: "slot", slot: 4 },
      {
        kind: "text",
        text: ". A clear photo of the signed paperwork in Dash is fine.",
      },
    ]);
    expect(template?.slots).toBe(5);
    expect(template?.values).toEqual([
      ["Laura", "S1001", "B77", "15", "Simpson & Jones Industries"],
      ["Theresa", "S1002", "B78", "16", "Acme Foods"],
      ["María José", "S1003", "B79", "17", "Northwind"],
    ]);
  });

  it("puts every text back together exactly from the template and its values", () => {
    const texts = [
      podMessage("Laura", "S1001", "B77", "15", "Simpson & Jones Industries"),
      podMessage("Theresa", "S1002", "B78", "16", "Acme Foods"),
    ];

    const template = sharedTemplate(texts);

    expect(template).not.toBeNull();
    texts.forEach((text, i) => {
      const values = template?.values[i];
      expect(values).not.toBeNull();
      expect(renderTemplate(template?.parts ?? [], values ?? [])).toBe(text);
    });
  });

  it("shows a message written differently whole and keeps the template for the rest", () => {
    const texts = [
      podMessage("Laura", "S1001", "B77", "15", "Acme Foods"),
      "Can you send me the POD for S1009 when you get a minute?",
      podMessage("Theresa", "S1002", "B78", "16", "Northwind"),
    ];

    const template = sharedTemplate(texts);

    expect(template).not.toBeNull();
    expect(template?.values[0]).toEqual(["Laura", "S1001", "B77", "15", "Acme Foods"]);
    expect(template?.values[1]).toBeNull();
    expect(template?.values[2]).toEqual(["Theresa", "S1002", "B78", "16", "Northwind"]);
  });

  it("finds the template when the first message is the one written differently", () => {
    const texts = [
      "Can you send me the POD for S1009 when you get a minute?",
      podMessage("Laura", "S1001", "B77", "15", "Acme Foods"),
      podMessage("Theresa", "S1002", "B78", "16", "Northwind"),
    ];

    const template = sharedTemplate(texts);

    expect(template?.values).toEqual([
      null,
      ["Laura", "S1001", "B77", "15", "Acme Foods"],
      ["Theresa", "S1002", "B78", "16", "Northwind"],
    ]);
  });

  it("reads identical texts as one text with nothing to fill", () => {
    const text = "Your load is ready for pickup at dock 4.";

    expect(sharedTemplate([text, text, text])).toEqual({
      parts: [{ kind: "text", text }],
      slots: 0,
      values: [[], [], []],
    });
  });

  it("offers no template for texts that share too little or for a single text", () => {
    expect(sharedTemplate([podMessage("Laura", "S1", "B1", "15", "Acme")])).toBeNull();
    expect(sharedTemplate([])).toBeNull();
    expect(
      sharedTemplate([
        "Please call dispatch about the reefer temperature.",
        "Your settlement for last week is ready to review.",
      ]),
    ).toBeNull();
  });

  it("offers no template when the texts differ in more places than a table can show", () => {
    const words = (n: number) =>
      Array.from({ length: 10 }, (_, i) => `fixed${i} v${n}x${i}`).join(" ");

    expect(sharedTemplate([words(1), words(2), words(3)])).toBeNull();
  });
});

describe("fillTemplate", () => {
  it("gives nothing for a text that does not come out of the template", () => {
    const parts = [
      { kind: "text" as const, text: "Hi " },
      { kind: "slot" as const, slot: 0 },
      { kind: "text" as const, text: ", thanks." },
    ];

    expect(fillTemplate(parts, "Hi Ana, thanks.")).toEqual(["Ana"]);
    expect(fillTemplate(parts, "Hello Ana, thanks.")).toBeNull();
    expect(fillTemplate(parts, "Hi Ana, thanks. And more.")).toBeNull();
  });
});
