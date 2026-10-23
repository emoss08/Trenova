import { rulesSync } from "@trenova/shared/rules/runtime";
import { describe, expect, it } from "vitest";
import { hasMaterialChange, saveDemotesToDraft } from "../formula-template-material";

const SOURCE =
  'tariff "Per mile"\n\ninput rate: Money/mi = $2.35/mi\n\ncharge "Linehaul" (quantity: shipment.distance, rate: rate)\n';
const TEMPLATE = { name: "Per mile", source: SOURCE };
const RENAMED = SOURCE.replace('tariff "Per mile"', 'tariff "Per mile, regional"');

describe("hasMaterialChange", () => {
  it("is a change to the source", () => {
    expect(hasMaterialChange(rulesSync(), TEMPLATE, TEMPLATE)).toBe(false);
    expect(
      hasMaterialChange(rulesSync(), TEMPLATE, {
        name: "Per mile",
        source: SOURCE.replace("$2.35", "$2.40"),
      }),
    ).toBe(true);
  });

  it("counts a whitespace edit, because the server compares the text it stores", () => {
    expect(
      hasMaterialChange(rulesSync(), TEMPLATE, { name: "Per mile", source: `${SOURCE}\n` }),
    ).toBe(true);
  });

  it("does not count a rename, which changes only the header", () => {
    expect(
      hasMaterialChange(rulesSync(), TEMPLATE, { name: "Per mile, regional", source: RENAMED }),
    ).toBe(false);
  });

  it("counts a rename made with a change to what it bills", () => {
    expect(
      hasMaterialChange(rulesSync(), TEMPLATE, {
        name: "Per mile, regional",
        source: RENAMED.replace("$2.35", "$2.40"),
      }),
    ).toBe(true);
  });
});

describe("saveDemotesToDraft", () => {
  const edited = { name: "Per mile", source: SOURCE.replace("$2.35", "$2.40") };

  it("demotes an active or in-review template whose source changed", () => {
    expect(saveDemotesToDraft(rulesSync(), { ...TEMPLATE, status: "Active" }, edited)).toBe(true);
    expect(saveDemotesToDraft(rulesSync(), { ...TEMPLATE, status: "InReview" }, edited)).toBe(true);
  });

  it("leaves a draft, an archived template, an unchanged source and a rename alone", () => {
    expect(saveDemotesToDraft(rulesSync(), { ...TEMPLATE, status: "Draft" }, edited)).toBe(false);
    expect(saveDemotesToDraft(rulesSync(), { ...TEMPLATE, status: "Inactive" }, edited)).toBe(
      false,
    );
    expect(saveDemotesToDraft(rulesSync(), { ...TEMPLATE, status: "Active" }, TEMPLATE)).toBe(
      false,
    );
    expect(
      saveDemotesToDraft(
        rulesSync(),
        { ...TEMPLATE, status: "Active" },
        { name: "Per mile, regional", source: RENAMED },
      ),
    ).toBe(false);
    expect(saveDemotesToDraft(rulesSync(), null, edited)).toBe(false);
  });
});
