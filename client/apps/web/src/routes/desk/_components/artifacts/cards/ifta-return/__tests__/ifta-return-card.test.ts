import { translate } from "@trenova/shared/i18n/runtime";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import { cardDate, money } from "../../../record-card/record-card-values";
import {
  netFigure,
  returnChecks,
  returnFacts,
  returnStanding,
  returnTitle,
  weightiestLines,
} from "../ifta-return-card";
import { NOW, draftCard, manyLines, openQuarterCard } from "./fixtures";

const t: TranslateFn = (message, ...args) => translate(message, ...args);

describe("the IFTA return card's words", () => {
  it("names the return by its quarter and, for an amendment, which one", () => {
    expect(returnTitle(draftCard(), t)).toBe("Q3 2026");
    expect(returnTitle(draftCard({ amendmentNumber: 2 }), t)).toBe("Q3 2026, amendment 2");
  });

  it("says what the quarter comes to as owed or as a credit", () => {
    expect(netFigure(draftCard(), t)).toEqual({ amount: money("258.26"), words: "owed" });
    expect(netFigure(openQuarterCard(), t)).toEqual({ amount: money("42.10"), words: "credit" });
  });

  it("colours the status as the IFTA page does", () => {
    expect(returnStanding("Draft")).toEqual({ label: "Draft", tone: "draft" });
    expect(returnStanding("Finalized")).toEqual({ label: "Finalized", tone: "progress" });
    expect(returnStanding("Filed")).toEqual({ label: "Filed", tone: "success" });
  });

  it("gives the filing date, and the quarter's last day while it runs", () => {
    expect(returnFacts(draftCard(), NOW, t)).toEqual([
      `Due ${cardDate(draftCard().period.dueDate)}`,
      "",
      "",
    ]);
    const open = openQuarterCard();
    expect(returnFacts(open, NOW, t)).toEqual([
      `Due ${cardDate(open.period.dueDate)}`,
      `Quarter ends ${cardDate(open.period.end - 86_400)}`,
      "",
    ]);
    const filed = draftCard({
      status: "Filed",
      filedAt: NOW - 86_400,
      filingReference: " IFTA-77 ",
    });
    expect(returnFacts(filed, NOW, t)).toEqual([
      `Filed ${cardDate(NOW - 86_400)}`,
      "",
      "Reference IFTA-77",
    ]);
  });

  it("puts a missing rate first as what blocks filing, groups the rest by kind and says when it is late", () => {
    expect(returnChecks(draftCard(), NOW, t)).toEqual([
      {
        key: "MissingRate",
        state: "bad",
        title: "Missing tax rate",
        detail: "No Diesel tax rate is on file for Oklahoma in Q3 2026",
      },
      {
        key: "UnattributedMiles",
        state: "warn",
        title: "Miles not attributed to a jurisdiction",
        detail: "3 completed moves totalling 412.00 miles have no jurisdiction breakdown",
      },
      {
        key: "NonMemberActivity",
        state: "warn",
        title: "Activity in a non-member jurisdiction",
        detail: "Mexico is not an IFTA member, and 1 more like it",
      },
      {
        key: "late",
        state: "warn",
        title: "Past the filing date",
        detail: `The return is not filed; the quarter's filing date was ${cardDate(draftCard().period.dueDate)}.`,
      },
    ]);
  });

  it("says a clear return is clear, and never calls a filed return late", () => {
    expect(returnChecks(openQuarterCard(), NOW, t)).toEqual([
      expect.objectContaining({ key: "clear", state: "ok" }),
    ]);
    expect(
      returnChecks(draftCard({ status: "Filed", filedAt: NOW, problems: [] }), NOW, t),
    ).toEqual([expect.objectContaining({ key: "clear" })]);
  });

  it("draws the unpriced line first, then the largest owed or credit, and counts the rest", () => {
    const { rows, rest } = weightiestLines(draftCard(), t);
    expect(rows.map((row) => [row.name, row.amount, row.state])).toEqual([
      ["Oklahoma", "Rate missing", "bad"],
      ["Texas", `${money("1352.40")} credit`, "ok"],
      ["Georgia", `${money("610.32")} owed`, "info"],
      ["Arkansas", `${money("464.78")} owed`, "info"],
    ]);
    expect(rows[1].sub).toBe("Diesel, 48,210 mi, 14,160 gal bought");
    expect(rest).toBe(0);

    const many = weightiestLines(draftCard({ lines: manyLines() }), t);
    expect(many.rows).toHaveLength(8);
    expect(many.rows[0].name).toBe("Jurisdiction 9");
    expect(many.rest).toBe(2);
  });
});
