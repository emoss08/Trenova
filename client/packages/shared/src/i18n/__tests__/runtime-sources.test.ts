import {
  registerCatalogSource,
  requireCatalog,
  setLocale,
  translate,
} from "@trenova/shared/i18n/runtime";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

beforeAll(() => {
  requireCatalog("core", "web");
});

afterEach(async () => {
  await setLocale("en");
});

describe("registerCatalogSource", () => {
  it("adds a package's strings on top of the app catalog, even after a locale loaded", async () => {
    await setLocale("es");
    expect(translate("Edition-only sentence")).toBe("Edition-only sentence");

    const es = vi.fn(async () => ({ "Edition-only sentence": "Frase de la edición" }));
    await registerCatalogSource({ es });

    expect(translate("Edition-only sentence")).toBe("Frase de la edición");
    expect(translate("Save")).toBe("Guardar");
    expect(es).toHaveBeenCalledTimes(1);
  });

  it("lets the package's entry win for a key the app also translates", async () => {
    await registerCatalogSource({ "zh-TW": async () => ({ Save: "儲存（版本）" }) });
    await setLocale("zh-TW");

    expect(translate("Save")).toBe("儲存（版本）");
  });

  it("leaves a locale the package does not translate to the app catalog", async () => {
    await setLocale("zh-CN");

    expect(translate("Edition-only sentence")).toBe("Edition-only sentence");
    expect(translate("Shipments")).toBe("运单");
  });

  it("ignores a source with no catalogs", async () => {
    await expect(registerCatalogSource({})).resolves.toBeUndefined();
  });
});
