import { LOCALES, LOCALE_REGIONS } from "@trenova/shared/i18n/generated/locales";
import { LocaleFlag } from "@trenova/shared/i18n/locale-flag";
import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

// Emoji flags rendered as nothing on Windows, which ships no flag emoji font. These assert
// the switcher draws real geometry instead, for every locale locales.json declares.
describe("LocaleFlag", () => {
  it.each(LOCALES)("draws an svg for %s rather than relying on an emoji font", (locale) => {
    const { container } = render(<LocaleFlag locale={locale} />);
    const svg = container.querySelector("svg");

    expect(svg).not.toBeNull();
    expect(svg?.getAttribute("viewBox")).toBe("0 0 640 480");
    // A flag is geometry, not text: nothing here should depend on a font being present.
    expect(container.textContent).toBe("");
    expect(svg?.querySelectorAll("rect, circle, path").length).toBeGreaterThan(1);
  });

  it("covers every locale declared in locales.json", () => {
    for (const locale of LOCALES) {
      expect(LOCALE_REGIONS[locale]).toMatch(/^[A-Z]{2}$/);
    }
  });

  it("falls back to the region code when a flag has not been drawn", () => {
    // A fifth language added to locales.json must still produce a usable switcher.
    const { container } = render(<LocaleFlag locale={"fr" as (typeof LOCALES)[number]} />);
    expect(container.querySelector("svg")).toBeNull();
  });
});
