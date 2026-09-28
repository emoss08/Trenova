import { describe, expect, it } from "vitest";
import type { AICorrectionFieldResult } from "@/lib/graphql/extraction-eval";
import {
  accuracyGap,
  hasProblems,
  outcomesDiffer,
  pairFieldResults,
  shadowSettingsProblems,
} from "../shadow-model";

function result(
  key: string,
  outcome: AICorrectionFieldResult["outcome"],
  predicted: string,
  confirmed: string,
): AICorrectionFieldResult {
  return { key, outcome, predicted, confirmed, source: "ai", confidence: 0.9 };
}

describe("shadowSettingsProblems", () => {
  it("accepts a disabled shadow with no provider", () => {
    const problems = shadowSettingsProblems({
      enabled: false,
      providerId: "",
      samplePercent: "10",
      dailyLimit: "200",
    });
    expect(problems).toEqual({});
    expect(hasProblems(problems)).toBe(false);
  });

  it("needs a provider only when the shadow is on", () => {
    expect(
      shadowSettingsProblems({
        enabled: true,
        providerId: "",
        samplePercent: "10",
        dailyLimit: "200",
      }).providerId,
    ).toBeDefined();
  });

  it("bounds the sample and the limit to the whole numbers the server accepts", () => {
    for (const samplePercent of ["0", "101", "12.5", "", " ", "-3", "1e2"]) {
      expect(
        shadowSettingsProblems({ enabled: false, providerId: "", samplePercent, dailyLimit: "5" })
          .samplePercent,
      ).toBeDefined();
    }
    for (const dailyLimit of ["0", "5001", "3.2", ""]) {
      expect(
        shadowSettingsProblems({ enabled: false, providerId: "", samplePercent: "5", dailyLimit })
          .dailyLimit,
      ).toBeDefined();
    }
    expect(
      shadowSettingsProblems({
        enabled: true,
        providerId: "aip_1",
        samplePercent: " 100 ",
        dailyLimit: "5000",
      }),
    ).toEqual({});
  });
});

describe("pairFieldResults", () => {
  it("lines up both sides by field, keeping a field only one side scored", () => {
    const rows = pairFieldResults(
      [result("rate", "Correct", "2563.12", "2563.12"), result("weight", "Missed", "", "42000")],
      [
        result("rate", "Corrected", "2500.00", "2563.12"),
        result("pieceCount", "Correct", "24", "24"),
      ],
    );

    expect(rows.map((row) => row.key)).toEqual(["rate", "weight", "pieceCount"]);
    const [rate, weight, pieces] = rows;
    expect(rate.candidate?.predicted).toBe("2563.12");
    expect(rate.production?.predicted).toBe("2500.00");
    expect(rate.confirmed).toBe("2563.12");
    expect(outcomesDiffer(rate)).toBe(true);

    expect(weight.production).toBeUndefined();
    expect(weight.confirmed).toBe("42000");
    expect(outcomesDiffer(weight)).toBe(true);

    expect(pieces.candidate).toBeUndefined();
    expect(pieces.confirmed).toBe("24");
  });

  it("takes the confirmed value from whichever side has it", () => {
    const [row] = pairFieldResults(
      [result("reference", "Unconfirmed", "LD-1", "")],
      [result("reference", "Missed", "", "LD-1")],
    );
    expect(row.confirmed).toBe("LD-1");
  });

  it("treats matching outcomes as agreement", () => {
    const [row] = pairFieldResults(
      [result("rate", "Correct", "1", "1")],
      [result("rate", "Correct", "1.00", "1")],
    );
    expect(outcomesDiffer(row)).toBe(false);
  });
});

describe("accuracyGap", () => {
  it("is the difference in whole points, and nothing without scores on both sides", () => {
    expect(accuracyGap({ accuracy: 0.92, scored: 10 }, { accuracy: 0.871, scored: 10 })).toBe(5);
    expect(accuracyGap({ accuracy: 0.8, scored: 10 }, { accuracy: 0.9, scored: 10 })).toBe(-10);
    expect(accuracyGap({ accuracy: 0, scored: 0 }, { accuracy: 0.9, scored: 10 })).toBeNull();
    expect(accuracyGap({ accuracy: 0.9, scored: 10 }, { accuracy: 0, scored: 0 })).toBeNull();
  });
});
