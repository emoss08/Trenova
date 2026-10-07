import type * as Locales from "@trenova/shared/i18n/generated/locales";
import type * as Runtime from "@trenova/shared/i18n/runtime";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

type Messages = Record<string, string>;

// Every load the runtime starts, so a test can hold one open (a slow route chunk) or fail it
// (a chunk that 404s after a deploy) and watch what the runtime does meanwhile.
type PendingLoad = {
  locale: string;
  bundle: string;
  resolve: (messages: Messages) => void;
  reject: (error: Error) => void;
};

const CATALOGS: Record<string, Record<string, Messages>> = {
  es: {
    core: { Save: "Guardar" },
    web: { Shipments: "Envíos" },
    "routes/shipment": { "New shipment": "Nuevo envío" },
    "routes/customer": { "New customer": "Nuevo cliente" },
  },
  "zh-CN": {
    core: { Save: "保存" },
    web: { Shipments: "运单" },
    "routes/shipment": { "New shipment": "新建运单" },
    "routes/customer": { "New customer": "新建客户" },
  },
};

let loads: PendingLoad[] = [];
let autoResolve = true;

function loaderFor(locale: string, bundle: string) {
  return () =>
    new Promise<Messages>((resolve, reject) => {
      if (autoResolve) {
        resolve(CATALOGS[locale][bundle]);
        return;
      }
      loads.push({ locale, bundle, resolve, reject });
    });
}

vi.mock("@trenova/shared/i18n/generated/locales", async (importOriginal) => {
  const actual = await importOriginal<typeof Locales>();
  const loaders: Record<string, Record<string, () => Promise<Messages>>> = { en: {} };
  for (const [locale, bundles] of Object.entries(CATALOGS)) {
    loaders[locale] = {};
    for (const bundle of Object.keys(bundles)) loaders[locale][bundle] = loaderFor(locale, bundle);
  }
  return { ...actual, CATALOG_LOADERS: loaders };
});

let runtime: typeof Runtime;

function settleLoad(locale: string, bundle: string): void {
  const index = loads.findIndex((load) => load.locale === locale && load.bundle === bundle);
  if (index === -1) throw new Error(`no pending load of ${bundle} for ${locale}`);
  const [load] = loads.splice(index, 1);
  load.resolve(CATALOGS[locale][bundle]);
}

function pendingBundles(locale: string): string[] {
  return loads.filter((load) => load.locale === locale).map((load) => load.bundle);
}

beforeEach(async () => {
  loads = [];
  autoResolve = true;
  vi.resetModules();
  runtime = await import("@trenova/shared/i18n/runtime");
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("requireCatalog", () => {
  it("leaves a bundle nobody required out of a locale switch", async () => {
    runtime.requireCatalog("core");
    await runtime.setLocale("es");

    expect(runtime.translate("Save")).toBe("Guardar");
    expect(runtime.translate("New shipment")).toBe("New shipment");
  });

  it("loads a bundle required after the switch and re-renders subscribers", async () => {
    runtime.requireCatalog("core");
    await runtime.setLocale("es");
    const listener = vi.fn();
    runtime.subscribe(listener);
    const version = runtime.getCatalogVersion();
    const translator = runtime.getTranslator();

    runtime.requireCatalog("routes/shipment");
    await runtime.whenCatalogsReady();

    expect(runtime.translate("New shipment")).toBe("Nuevo envío");
    expect(listener).toHaveBeenCalled();
    expect(runtime.getCatalogVersion()).not.toBe(version);
    // Effects keyed on `t` must not re-run because a route's strings arrived.
    expect(runtime.getTranslator()).toBe(translator);
    expect(runtime.getTranslator()("New shipment")).toBe("Nuevo envío");
  });

  it("downloads nothing while English is showing, and everything on the switch", async () => {
    autoResolve = false;
    runtime.requireCatalog("core", "routes/shipment");
    runtime.requireCatalog("routes/customer");
    await runtime.whenCatalogsReady();
    expect(loads).toHaveLength(0);

    const switched = runtime.setLocale("zh-CN");
    await vi.waitFor(() => expect(pendingBundles("zh-CN")).toHaveLength(3));
    for (const bundle of ["core", "routes/shipment", "routes/customer"]) {
      settleLoad("zh-CN", bundle);
    }
    await switched;

    expect(runtime.translate("New customer")).toBe("新建客户");
    expect(runtime.translate("New shipment")).toBe("新建运单");
  });

  it("does not finish a switch without a bundle required while it was in flight", async () => {
    autoResolve = false;
    runtime.requireCatalog("core");

    let switched = false;
    const switching = runtime.setLocale("es").then(() => {
      switched = true;
    });
    await vi.waitFor(() => expect(pendingBundles("es")).toEqual(["core"]));

    runtime.requireCatalog("routes/shipment");
    settleLoad("es", "core");
    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(switched).toBe(false);
    expect(runtime.getLocale()).toBe("en");

    settleLoad("es", "routes/shipment");
    await switching;

    expect(runtime.getLocale()).toBe("es");
    expect(runtime.translate("New shipment")).toBe("Nuevo envío");
  });

  it("starts each bundle once however many modules require it", async () => {
    autoResolve = false;
    await runtime.setLocale("es");

    runtime.requireCatalog("routes/shipment");
    runtime.requireCatalog("routes/shipment");
    const ready = runtime.whenCatalogsReady();
    const again = runtime.whenCatalogsReady();

    expect(pendingBundles("es")).toEqual(["routes/shipment"]);
    settleLoad("es", "routes/shipment");
    await Promise.all([ready, again]);
  });

  it("treats a route folder with no strings as already loaded", async () => {
    await runtime.setLocale("es");
    runtime.requireCatalog("routes/no-strings-here");

    await expect(runtime.whenCatalogsReady()).resolves.toBeUndefined();
  });

  it("reports a failed download and retries it on the next wait", async () => {
    await runtime.setLocale("es");
    autoResolve = false;

    runtime.requireCatalog("routes/customer");
    const failed = runtime.whenCatalogsReady();
    const [load] = loads.splice(0, 1);
    load.reject(new Error("chunk failed to load"));
    await expect(failed).rejects.toThrow("chunk failed to load");
    expect(runtime.translate("New customer")).toBe("New customer");

    const retried = runtime.whenCatalogsReady();
    await vi.waitFor(() => expect(pendingBundles("es")).toEqual(["routes/customer"]));
    settleLoad("es", "routes/customer");
    await retried;

    expect(runtime.translate("New customer")).toBe("Nuevo cliente");
  });
});

describe("setLocale", () => {
  it("gives translate a new identity per language and keeps it for the same one", async () => {
    const english = runtime.getTranslator();
    await runtime.setLocale("es");
    const spanish = runtime.getTranslator();
    await runtime.setLocale("es");

    expect(spanish).not.toBe(english);
    expect(runtime.getTranslator()).toBe(spanish);
  });

  it("keeps a translator bound to the language it was rendered in", async () => {
    runtime.requireCatalog("web");
    await runtime.setLocale("es");
    const spanish = runtime.getTranslator();
    await runtime.setLocale("zh-CN");

    expect(spanish("Shipments")).toBe("Envíos");
    expect(runtime.translate("Shipments")).toBe("运单");
  });
});

describe("registerCatalogSource", () => {
  it("keeps the edition's entry over a route bundle that lands after it", async () => {
    runtime.requireCatalog("core");
    await runtime.registerCatalogSource({
      es: async () => ({ "New shipment": "Nuevo envío (edición)" }),
    });
    await runtime.setLocale("es");

    runtime.requireCatalog("routes/shipment");
    await runtime.whenCatalogsReady();

    expect(runtime.translate("New shipment")).toBe("Nuevo envío (edición)");
  });
});
