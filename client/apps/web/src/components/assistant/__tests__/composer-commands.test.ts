import { describe, expect, it } from "vitest";
import { matchSuggestions, slashQuery } from "../composer-commands";

const suggestions = [
  { label: "Where is a shipment right now?", prompt: "Where is PRO S12345 right now?" },
  { label: "What is picking up today?", prompt: "Which shipments pick up today?" },
  { label: "Is a driver available?", prompt: "Is Maria Ortiz free tomorrow?" },
];

describe("slashQuery", () => {
  it("reads a leading slash as a request for the starter questions", () => {
    expect(slashQuery("/")).toBe("");
    expect(slashQuery("/ship")).toBe("ship");
    expect(slashQuery("/  Driver ")).toBe("driver");
  });

  it("ignores a slash that is part of a sentence or a multi-line draft", () => {
    expect(slashQuery("what about 12/24?")).toBeNull();
    expect(slashQuery("/ship\nmore")).toBeNull();
    expect(slashQuery("")).toBeNull();
  });
});

describe("matchSuggestions", () => {
  it("offers everything for an empty query", () => {
    expect(matchSuggestions("", suggestions)).toHaveLength(3);
  });

  it("matches on the label or the prompt, ignoring case", () => {
    expect(matchSuggestions("SHIP", suggestions).map((s) => s.label)).toEqual([
      "Where is a shipment right now?",
      "What is picking up today?",
    ]);
    expect(matchSuggestions("ortiz", suggestions).map((s) => s.label)).toEqual([
      "Is a driver available?",
    ]);
    expect(matchSuggestions("zzz", suggestions)).toEqual([]);
  });
});
