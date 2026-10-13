import { describe, expect, it } from "vitest";
import { askQuestion, commandModeQuery, questionSearchText } from "../ask-question";

/**
 * The palette turns into a question when the entry reads as one: a question
 * mark at the end, or a question word opening three words or more.
 */
describe("askQuestion", () => {
  it("reads a question mark at the end as a question", () => {
    expect(askQuestion("any loads at risk today?")).toBe("any loads at risk today?");
    expect(askQuestion("  late? ")).toBe("late?");
  });

  it("reads a question word opening three or more words as a question", () => {
    expect(askQuestion("show me late loads")).toBe("show me late loads");
    expect(askQuestion("which drivers are available")).toBe("which drivers are available");
    expect(askQuestion("summarize Acme's open invoices")).toBe("summarize Acme's open invoices");
  });

  it("leaves short openers and plain searches alone", () => {
    expect(askQuestion("where is")).toBeNull();
    expect(askQuestion("listings for acme")).toBeNull();
    expect(askQuestion("shipments")).toBeNull();
    expect(askQuestion("?")).toBeNull();
    expect(askQuestion("")).toBeNull();
  });
});

describe("commandModeQuery", () => {
  it("strips the chevron and keeps what follows as the command query", () => {
    expect(commandModeQuery(">theme")).toBe("theme");
    expect(commandModeQuery(">")).toBe("");
    expect(commandModeQuery("theme")).toBeNull();
  });
});

describe("questionSearchText", () => {
  it("keeps the words that name a record and drops the way the question is asked", () => {
    expect(questionSearchText("Tell me about SEED-PAY-002?")).toBe("SEED-PAY-002");
    expect(questionSearchText("HARBOR Line Foods?")).toBe("HARBOR Line Foods");
    expect(questionSearchText("What's going on with HARBOR Line Foods?")).toBe("HARBOR Line Foods");
    expect(questionSearchText("why is S-1001 running late?")).toBe("S-1001 running late");
  });

  it("drops a possessive and the punctuation around a word, not inside it", () => {
    expect(questionSearchText("summarize Acme's open invoices")).toBe("Acme open invoices");
    expect(questionSearchText("Where is #4471, and who’s driving?")).toBe("#4471 driving");
  });

  it("is empty when nothing but the asking is left", () => {
    expect(questionSearchText("what is this?")).toBe("");
    expect(questionSearchText("  ")).toBe("");
  });
});
