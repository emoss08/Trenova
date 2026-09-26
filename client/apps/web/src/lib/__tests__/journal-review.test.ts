import type { JournalReviewResult } from "@/lib/graphql/journal-review";
import { journalReviewMessage } from "@/lib/journal-review";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";

const t = ((template: string, ...args: unknown[]) =>
  template.replace(/\{(\d+)\}/g, (_, index: string) => String(args[Number(index)]))) as TranslateFn;

function result(outcomes: Array<{ entryNumber: string; error: string }>): JournalReviewResult {
  const failed = outcomes.filter((outcome) => outcome.error !== "").length;
  return {
    changed: outcomes.length - failed,
    failed,
    outcomes: outcomes.map((outcome, index) => ({
      entryId: `je_${index}`,
      entryNumber: outcome.entryNumber,
      status: outcome.error ? "Pending" : "Approved",
      changed: outcome.error === "",
      error: outcome.error,
    })),
  };
}

describe("journalReviewMessage", () => {
  it("reports every entry changed as a success", () => {
    const message = journalReviewMessage(
      t,
      "post",
      result([
        { entryNumber: "JE-1", error: "" },
        { entryNumber: "JE-2", error: "" },
      ]),
    );
    expect(message).toEqual({ tone: "success", title: "2 posted to the general ledger" });
  });

  it("names the first refusal and counts the rest when some entries changed", () => {
    const message = journalReviewMessage(
      t,
      "approve",
      result([
        { entryNumber: "JE-1", error: "" },
        { entryNumber: "JE-2", error: "Journal entry JE-2 is already approved" },
        { entryNumber: "JE-3", error: "Journal entry not found" },
      ]),
    );
    expect(message.tone).toBe("warning");
    expect(message.title).toBe("1 approved; 2 could not be changed");
    expect(message.description).toBe("JE-2: Journal entry JE-2 is already approved (and 1 more)");
  });

  it("is an error when nothing changed", () => {
    const message = journalReviewMessage(
      t,
      "post",
      result([{ entryNumber: "", error: "Journal entry not found" }]),
    );
    expect(message.tone).toBe("error");
    expect(message.title).toBe("None of the 1 selected could be changed");
    expect(message.description).toBe("Journal entry not found");
  });
});
