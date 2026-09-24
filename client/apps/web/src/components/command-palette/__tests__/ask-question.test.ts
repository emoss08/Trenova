import { describe, expect, it } from "vitest";
import { askQuestion } from "../ask-question";

/**
 * The palette turns into a question when the entry reads as one: a question
 * mark at the end, or a chevron at the start for people who prefer a prefix.
 * Short entries ending in a question mark are still searches, so "S1?" does
 * not become a question about nothing.
 */
describe("askQuestion", () => {
  it("reads a question mark at the end as a question", () => {
    expect(askQuestion("How many loads are late today?")).toBe("How many loads are late today?");
    expect(askQuestion("  what is late? ")).toBe("what is late?");
  });

  it("reads a chevron prefix as a question without the chevron", () => {
    expect(askQuestion("> show me late loads")).toBe("show me late loads");
    expect(askQuestion(">")).toBeNull();
  });

  it("leaves searches alone", () => {
    expect(askQuestion("shipments")).toBeNull();
    expect(askQuestion("S1?")).toBeNull();
    expect(askQuestion("")).toBeNull();
  });
});
