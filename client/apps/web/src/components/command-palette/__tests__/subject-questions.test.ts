import { describe, expect, it } from "vitest";
import { subjectQuestions } from "../subject-questions";
import { t } from "./palette-test-fixtures";

describe("subjectQuestions", () => {
  it("offers what is worth asking about an invoice: why it is unpaid, its dispute, a reminder", () => {
    expect(subjectQuestions("invoice", t)).toEqual([
      "Why hasn't it been paid?",
      "What's in dispute, and for how much?",
      "Draft a payment reminder for this invoice",
    ]);
  });

  it("offers nothing for a kind it has no questions for", () => {
    expect(subjectQuestions("trailer", t)).toEqual([]);
  });
});
