import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import { acceptedShare, figure, learningRowLabel } from "../memory-learning-model";

const t = ((text: string) => text) as TranslateFn;

describe("learningRowLabel", () => {
  it("names a look back by what made it look", () => {
    expect(learningRowLabel({ source: "Reflection", trigger: "PersonTold" }, t)).toBe(
      "A person mentioned it in passing",
    );
    expect(learningRowLabel({ source: "Reflection", trigger: "ChangedAfterwards" }, t)).toBe(
      "People kept changing what it set",
    );
    expect(learningRowLabel({ source: "Reflection", trigger: null }, t)).toBe(
      "Looking back, other",
    );
  });

  it("names every other way a memory comes to be kept", () => {
    expect(learningRowLabel({ source: "Agent", trigger: null }, t)).toBe(
      "Saved in a conversation when asked",
    );
    expect(learningRowLabel({ source: "User", trigger: null }, t)).toBe("Written down by a person");
    expect(learningRowLabel({ source: "Decision", trigger: null }, t)).toBe(
      "A reason given on a decision",
    );
    expect(learningRowLabel({ source: "Feedback", trigger: null }, t)).toBe("Ratings on replies");
  });
});

describe("acceptedShare", () => {
  it("is the share of decided offers that were kept, retired ones included", () => {
    expect(acceptedShare({ source: "Reflection", kept: 5, retired: 1, dismissed: 2 })).toBe(75);
    expect(acceptedShare({ source: "Reflection", kept: 0, retired: 0, dismissed: 4 })).toBe(0);
    expect(acceptedShare({ source: "Agent", kept: 3, retired: 0, dismissed: 0 })).toBe(100);
  });

  it("is absent while nobody has decided anything", () => {
    expect(acceptedShare({ source: "Reflection", kept: 0, retired: 0, dismissed: 0 })).toBeNull();
  });

  it("is absent for what a person wrote down, which was never an offer", () => {
    expect(acceptedShare({ source: "User", kept: 2, retired: 0, dismissed: 0 })).toBeNull();
  });
});

describe("figure", () => {
  it("shows a dash for nought and for a figure the row does not have", () => {
    expect(figure(0)).toBe("—");
    expect(figure(12, false)).toBe("—");
    expect(figure(1200)).toBe("1,200");
  });
});
