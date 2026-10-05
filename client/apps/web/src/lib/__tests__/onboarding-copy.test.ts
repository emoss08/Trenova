import {
  BOLD_MARKS,
  boldSegments,
  firstNameOf,
  novaLine,
  segmentsText,
} from "@/lib/onboarding-copy";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";

function format(message: string | null | undefined, ...args: unknown[]): string {
  return (message ?? "").replace(/\{(\d)\}/g, (_, index: string) => String(args[Number(index)]));
}
const t = format as TranslateFn;

describe("boldSegments", () => {
  it("marks the values a line says in bold, wherever the translation puts them", () => {
    expect(
      boldSegments(format("{1} est réglé, {0}.", ...BOLD_MARKS), ["Rivera", "Central"]),
    ).toEqual([
      { text: "Central", bold: true },
      { text: " est réglé, ", bold: false },
      { text: "Rivera", bold: true },
      { text: ".", bold: false },
    ]);
  });

  it("drops an empty value rather than printing an empty bold run", () => {
    expect(boldSegments(format("Saving {0} as headquarters", ...BOLD_MARKS), [""])).toEqual([
      { text: "Saving ", bold: false },
      { text: " as headquarters", bold: false },
    ]);
  });
});

describe("novaLine", () => {
  const context = { firstName: "Marcus", company: "Rivera Freight", browserZoneLabel: "Central" };

  it("greets by first name, and without one when the session has none", () => {
    expect(segmentsText(novaLine(t, "name", context))).toMatch(/^Hi Marcus, I'm Nova\. /);
    expect(segmentsText(novaLine(t, "name", { ...context, firstName: "" }))).toMatch(
      /^Hi, I'm Nova\. /,
    );
  });

  it("names the browser's zone only when there is one", () => {
    expect(novaLine(t, "timezone", context)).toContainEqual({ text: "Central", bold: true });
    expect(segmentsText(novaLine(t, "timezone", { ...context, browserZoneLabel: "" }))).toBe(
      "Got it, Rivera Freight. Which timezone should dispatch, appointments and reports use?",
    );
  });
});

describe("firstNameOf", () => {
  it("takes the first word of the name", () => {
    expect(firstNameOf("  Marcus  Rivera ")).toBe("Marcus");
    expect(firstNameOf(undefined)).toBe("");
  });
});
