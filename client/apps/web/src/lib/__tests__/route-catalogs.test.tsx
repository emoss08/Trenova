import { lazyPage } from "@/lib/route-catalogs";
import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const catalogs = vi.hoisted(() => {
  let release: () => void = () => undefined;
  let fail: (error: Error) => void = () => undefined;
  return {
    whenCatalogsReady: vi.fn(
      () =>
        new Promise<void>((resolve, reject) => {
          release = resolve;
          fail = reject;
        }),
    ),
    release: () => release(),
    fail: (error: Error) => fail(error),
  };
});

// afterCatalogs as the runtime defines it, over a whenCatalogsReady the test controls; the
// runtime's own behaviour is pinned in packages/shared (runtime-bundles.test.ts).
vi.mock("@trenova/shared/i18n/runtime", () => ({
  afterCatalogs: async <T,>(value: T): Promise<T> => {
    await catalogs.whenCatalogsReady().catch(() => undefined);
    return value;
  },
}));

function Page() {
  return null;
}

async function isSettled(promise: Promise<unknown>): Promise<boolean> {
  let settled = false;
  void promise.then(
    () => {
      settled = true;
    },
    () => {
      settled = true;
    },
  );
  await new Promise((resolve) => setTimeout(resolve, 0));
  return settled;
}

beforeEach(() => {
  catalogs.whenCatalogsReady.mockClear();
});

// What the router awaits before it draws the page for the first time.
function load(importer: () => Promise<{ Page: typeof Page }>) {
  return lazyPage(importer, "Page").preload();
}

describe("lazyPage", () => {
  it("holds a page until the catalogs its code required have loaded", async () => {
    const loading = load(async () => ({ Page }));
    expect(await isSettled(loading)).toBe(false);
    expect(catalogs.whenCatalogsReady).toHaveBeenCalledTimes(1);

    catalogs.release();
    await expect(loading).resolves.toBeUndefined();
  });

  it("asks for the catalogs only after the page's module has evaluated", async () => {
    let evaluated = false;
    catalogs.whenCatalogsReady.mockImplementationOnce(async () => {
      expect(evaluated).toBe(true);
    });

    await load(async () => {
      await Promise.resolve();
      evaluated = true;
      return { Page };
    });
    expect(catalogs.whenCatalogsReady).toHaveBeenCalledTimes(1);
  });

  it("still renders the page, in English, when a catalog fails to download", async () => {
    const loading = load(async () => ({ Page }));
    await vi.waitFor(() => expect(catalogs.whenCatalogsReady).toHaveBeenCalled());
    catalogs.fail(new Error("chunk failed to load"));

    await expect(loading).resolves.toBeUndefined();
  });

  it("draws the page once it has loaded", async () => {
    const Lazy = lazyPage(async () => ({ Page: () => <p>Shipments</p> }), "Page");
    const loading = Lazy.preload();
    await vi.waitFor(() => expect(catalogs.whenCatalogsReady).toHaveBeenCalled());
    catalogs.release();
    await loading;

    render(<Lazy />);

    expect(screen.getByText("Shipments")).toBeInTheDocument();
  });

  it("passes on a page module that fails to load, and imports it again on a retry", async () => {
    const importer = vi
      .fn<() => Promise<{ Page: typeof Page }>>()
      .mockRejectedValueOnce(new Error("page chunk failed"))
      .mockResolvedValue({ Page });

    await expect(lazyPage(importer, "Page").preload()).rejects.toThrow("page chunk failed");
    expect(catalogs.whenCatalogsReady).not.toHaveBeenCalled();
  });

  it("passes on a page module that fails to load", async () => {
    await expect(
      load(async () => {
        throw new Error("page chunk failed");
      }),
    ).rejects.toThrow("page chunk failed");
    expect(catalogs.whenCatalogsReady).not.toHaveBeenCalled();
  });
});
