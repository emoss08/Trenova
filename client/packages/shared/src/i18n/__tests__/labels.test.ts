import type * as Locales from "@trenova/shared/i18n/generated/locales";
import { defineLabels, sourceLabels } from "@trenova/shared/i18n/labels";
import { requireCatalog, setLocale } from "@trenova/shared/i18n/runtime";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

const SPANISH: Record<string, string> = {
  Online: "En línea",
  "On the job": "En el puesto",
  "Over {0} miles": "Más de {0} millas",
};

const CHINESE: Record<string, string> = {
  Online: "线上",
  "On the job": "在岗",
};

vi.mock("@trenova/shared/i18n/generated/locales", async (importOriginal) => {
  const actual = await importOriginal<typeof Locales>();
  return {
    ...actual,
    CATALOG_LOADERS: {
      ...actual.CATALOG_LOADERS,
      es: { core: async () => SPANISH },
      "zh-CN": { core: async () => CHINESE },
    },
  };
});

const DELIVERY = defineLabels({
  Online: "Online",
  OnTheJob: "On the job",
  Classroom: "Classroom",
  Long: "Over {0} miles",
});

describe("defineLabels", () => {
  beforeAll(() => {
    requireCatalog("core");
  });

  afterEach(async () => {
    await setLocale("en");
  });

  it("reads each label in the language on screen, though the map was built before it", async () => {
    expect(DELIVERY.OnTheJob).toBe("On the job");

    await setLocale("es");
    expect(DELIVERY.OnTheJob).toBe("En el puesto");

    await setLocale("zh-CN");
    expect(DELIVERY.OnTheJob).toBe("在岗");
  });

  it("falls back to the English source for a label with no translation yet", async () => {
    await setLocale("es");
    expect(DELIVERY.Classroom).toBe("Classroom");
  });

  it("translates the values an iteration sees, and keeps the keys", async () => {
    await setLocale("es");

    expect(Object.keys(DELIVERY)).toEqual(["Online", "OnTheJob", "Classroom", "Long"]);
    expect(Object.entries(DELIVERY).slice(0, 2)).toEqual([
      ["Online", "En línea"],
      ["OnTheJob", "En el puesto"],
    ]);
    expect({ ...DELIVERY }.Online).toBe("En línea");
  });

  it("returns a label as written, never treating it as a message template", async () => {
    await setLocale("es");
    expect(DELIVERY.Long).toBe("Más de {0} millas");
  });

  it("cannot be changed after it is declared", () => {
    expect(Object.isFrozen(DELIVERY)).toBe(true);
    expect(() => {
      (DELIVERY as Record<string, string>).Online = "x";
    }).toThrow();
  });

  it("hands data built at import the English source, whatever language is on screen", async () => {
    await setLocale("es");

    expect(sourceLabels(DELIVERY).OnTheJob).toBe("On the job");
    expect(() => sourceLabels({ a: "A" } as never)).toThrow(/defineLabels/);
  });
});
