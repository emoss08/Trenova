import { describe, expect, it } from "vitest";
import { hasMaterialChange, saveDemotesToDraft } from "../formula-template-material";

const SOURCE = 'tariff "Per mile"\n\ninput rate: Money/mi = $2.35/mi\n\ncharge "Linehaul" (quantity: shipment.distance, rate: rate)\n';

describe("hasMaterialChange", () => {
  it("is a change to the source", () => {
    expect(hasMaterialChange({ source: SOURCE }, { source: SOURCE })).toBe(false);
    expect(
      hasMaterialChange({ source: SOURCE }, { source: SOURCE.replace("$2.35", "$2.40") }),
    ).toBe(true);
  });

  it("counts a whitespace edit, because the server compares the text it stores", () => {
    expect(hasMaterialChange({ source: SOURCE }, { source: `${SOURCE}\n` })).toBe(true);
  });
});

describe("saveDemotesToDraft", () => {
  const edited = { source: SOURCE.replace("$2.35", "$2.40") };

  it("demotes an active or in-review template whose source changed", () => {
    expect(saveDemotesToDraft({ source: SOURCE, status: "Active" }, edited)).toBe(true);
    expect(saveDemotesToDraft({ source: SOURCE, status: "InReview" }, edited)).toBe(true);
  });

  it("leaves a draft, an archived template and an unchanged source alone", () => {
    expect(saveDemotesToDraft({ source: SOURCE, status: "Draft" }, edited)).toBe(false);
    expect(saveDemotesToDraft({ source: SOURCE, status: "Inactive" }, edited)).toBe(false);
    expect(saveDemotesToDraft({ source: SOURCE, status: "Active" }, { source: SOURCE })).toBe(false);
    expect(saveDemotesToDraft(null, edited)).toBe(false);
  });
});
