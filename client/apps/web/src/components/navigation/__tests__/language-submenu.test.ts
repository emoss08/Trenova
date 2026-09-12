import { switchLocale } from "@/components/navigation/language-submenu";
import { getLocale, setLocale } from "@trenova/shared/i18n/runtime";
import type { UpdateMySettings } from "@trenova/shared/types/user";
import { beforeEach, describe, expect, it, vi } from "vitest";

const user = { timezone: "America/Chicago", timeFormat: "24-hour" } as const;

beforeEach(async () => {
  await setLocale("en");
});

describe("switchLocale", () => {
  it("applies the language and sends the other preferences unchanged", async () => {
    const save = vi.fn<(v: UpdateMySettings) => Promise<unknown>>().mockResolvedValue({});
    const onSaved = vi.fn();

    await switchLocale({ next: "es", previous: "en", user, save, onSaved });

    expect(getLocale()).toBe("es");
    expect(save).toHaveBeenCalledWith({
      locale: "es",
      timezone: "America/Chicago",
      timeFormat: "24-hour",
    });
    expect(onSaved).toHaveBeenCalledOnce();
  });

  it("puts the language back when the save fails", async () => {
    const save = vi.fn().mockRejectedValue(new Error("422"));
    const onSaved = vi.fn();

    await switchLocale({ next: "zh-TW", previous: "en", user, save, onSaved });

    expect(getLocale()).toBe("en");
    expect(onSaved).not.toHaveBeenCalled();
  });

  it("does nothing when the language is already active", async () => {
    const save = vi.fn();
    const onSaved = vi.fn();

    await switchLocale({ next: "en", previous: "en", user, save, onSaved });

    expect(save).not.toHaveBeenCalled();
    expect(onSaved).not.toHaveBeenCalled();
  });

  it("falls back to defaults when the user has no stored preferences", async () => {
    const save = vi.fn().mockResolvedValue({});

    await switchLocale({ next: "es", previous: "en", user: null, save, onSaved: () => {} });

    expect(save).toHaveBeenCalledWith({
      locale: "es",
      timezone: "America/New_York",
      timeFormat: "12-hour",
    });
  });
});
