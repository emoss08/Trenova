import { applyLocale, persistLocale } from "@/components/navigation/language-submenu";
import { getLocale, setLocale } from "@trenova/shared/i18n/runtime";
import type { UpdateMySettings } from "@trenova/shared/types/user";
import { beforeEach, describe, expect, it, vi } from "vitest";

const user = { timezone: "America/Chicago", timeFormat: "24-hour" } as const;

beforeEach(async () => {
  await setLocale("en");
});

describe("applyLocale", () => {
  it("puts the interface into the new language", async () => {
    await applyLocale("es");
    expect(getLocale()).toBe("es");
  });
});

describe("persistLocale", () => {
  it("sends the other preferences unchanged alongside the locale", async () => {
    const save = vi.fn<(v: UpdateMySettings) => Promise<unknown>>().mockResolvedValue({});
    const onSaved = vi.fn();

    await persistLocale({ next: "es", previous: "en", user, save, onSaved });

    expect(save).toHaveBeenCalledWith({
      locale: "es",
      timezone: "America/Chicago",
      timeFormat: "24-hour",
    });
    expect(onSaved).toHaveBeenCalledOnce();
  });

  it("puts the language back when the save fails", async () => {
    await applyLocale("zh-TW");
    const save = vi.fn().mockRejectedValue(new Error("422"));
    const onSaved = vi.fn();

    await persistLocale({ next: "zh-TW", previous: "en", user, save, onSaved });

    expect(getLocale()).toBe("en");
    expect(onSaved).not.toHaveBeenCalled();
  });

  it("falls back to defaults when the user has no stored preferences", async () => {
    const save = vi.fn().mockResolvedValue({});

    await persistLocale({ next: "es", previous: "en", user: null, save, onSaved: () => {} });

    expect(save).toHaveBeenCalledWith({
      locale: "es",
      timezone: "America/New_York",
      timeFormat: "12-hour",
    });
  });
});
