import { resolveInitialLocale } from "@trenova/shared/i18n/provider";
import { getLocale, setLocale, translate } from "@trenova/shared/i18n/runtime";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

beforeEach(() => {
  window.localStorage.clear();
});

afterEach(async () => {
  window.localStorage.clear();
  await setLocale("en");
});

describe("resolveInitialLocale", () => {
  it("prefers the signed-in user's stored preference", () => {
    window.localStorage.setItem("trenova.locale", "es");
    expect(resolveInitialLocale("zh-TW")).toBe("zh-TW");
  });

  it("falls back to the remembered browser choice", () => {
    window.localStorage.setItem("trenova.locale", "es");
    expect(resolveInitialLocale(null)).toBe("es");
  });

  it("ignores a stored language we no longer ship", () => {
    window.localStorage.setItem("trenova.locale", "fr");
    expect(resolveInitialLocale(null)).toBe("en");
  });

  it("ignores a user preference we no longer ship", () => {
    expect(resolveInitialLocale("klingon")).toBe("en");
  });

  it("defaults to English with nothing to go on", () => {
    expect(resolveInitialLocale(undefined)).toBe("en");
  });
});

describe("translate", () => {
  it("falls back to the English source when a translation is missing", () => {
    expect(translate("A string nobody has translated")).toBe(
      "A string nobody has translated",
    );
  });

  it("returns an empty string unchanged", () => {
    expect(translate("")).toBe("");
  });

  it("interpolates even when falling back", () => {
    expect(translate('Delete "{0}"?', "Route 9")).toBe('Delete "Route 9"?');
  });

  it("renders a real translation after switching locale", async () => {
    await setLocale("es");
    expect(getLocale()).toBe("es");
    expect(translate("Save")).toBe("Guardar");
  });

  it("switches back", async () => {
    await setLocale("es");
    await setLocale("en");
    expect(translate("Save")).toBe("Save");
  });

  it("sets the document language so screen readers and hyphenation follow", async () => {
    await setLocale("zh-TW");
    expect(document.documentElement.lang).toBe("zh-TW");
  });
});

describe("real catalog entries", () => {
  it("renders navigation in Spanish", async () => {
    await setLocale("es");
    expect(translate("Shipments")).toBe("Envíos");
    expect(translate("Create Shipment")).toBe("Crear envío");
    expect(translate("Save Changes")).toBe("Guardar cambios");
  });

  it("renders navigation in Traditional Chinese with Taiwan vocabulary", async () => {
    await setLocale("zh-TW");
    expect(translate("Shipments")).toBe("運單");
    expect(translate("User Settings")).toBe("使用者設定");
    expect(translate("Trailers")).toBe("拖車");
  });

  it("renders navigation in Simplified Chinese with mainland vocabulary", async () => {
    await setLocale("zh-CN");
    expect(translate("Shipments")).toBe("运单");
    expect(translate("User Settings")).toBe("用户设置");
    expect(translate("Trailers")).toBe("挂车");
  });

  it("keeps the two Chinese variants distinct", async () => {
    await setLocale("zh-TW");
    const traditional = translate("Audit Entries");
    await setLocale("zh-CN");
    expect(translate("Audit Entries")).not.toBe(traditional);
  });
});
