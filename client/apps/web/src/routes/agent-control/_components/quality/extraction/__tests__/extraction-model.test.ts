import { describe, expect, it } from "vitest";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import {
  accuracyTone,
  documentKindLabel,
  fieldLabel,
  isRunActive,
  modelLabel,
  nextCaseStatuses,
} from "../extraction-model";

const t = ((text: string, ...args: unknown[]) =>
  args.reduce<string>(
    (out, arg, index) => out.replace(`{${index}}`, String(arg)),
    text,
  )) as TranslateFn;

describe("fieldLabel", () => {
  it("names top-level fields and stop fields as a person reads them", () => {
    expect(fieldLabel("rate", t)).toBe("Rate");
    expect(fieldLabel("pieceCount", t)).toBe("Pieces");
    expect(fieldLabel("stops.pickup.city", t)).toBe("Pickup city");
    expect(fieldLabel("stops.delivery[1].postalCode", t)).toBe("Delivery 2 ZIP");
    expect(fieldLabel("somethingNew", t)).toBe("somethingNew");
  });
});

describe("case lifecycle", () => {
  it("matches the server's transitions", () => {
    expect(nextCaseStatuses("Candidate")).toEqual(["Active", "Retired"]);
    expect(nextCaseStatuses("Active")).toEqual(["Retired"]);
    expect(nextCaseStatuses("Retired")).toEqual(["Active"]);
  });
});

describe("run and accuracy helpers", () => {
  it("treats queued and running as in flight", () => {
    expect(isRunActive("Queued")).toBe(true);
    expect(isRunActive("Running")).toBe(true);
    expect(isRunActive("BudgetStopped")).toBe(false);
  });

  it("tones accuracy only when something was scored", () => {
    expect(accuracyTone(0.95, 10)).toBe("success");
    expect(accuracyTone(0.8, 10)).toBe("warning");
    expect(accuracyTone(0.5, 10)).toBe("danger");
    expect(accuracyTone(0, 0)).toBeUndefined();
  });

  it("labels drafts no model read and known document kinds", () => {
    expect(modelLabel("", t)).toBe("Rules only");
    expect(modelLabel("gpt-large", t)).toBe("gpt-large");
    expect(documentKindLabel("RateConfirmation", t)).toBe("Rate confirmation");
    expect(documentKindLabel("", t)).toBe("Unclassified");
  });
});
