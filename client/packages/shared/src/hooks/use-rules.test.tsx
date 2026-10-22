import { RULES_WASM_BYTES as MODULE } from "@trenova/shared/rules/test-module";
import { afterEach, describe, expect, it, vi } from "vitest";

// React, the testing library, the hook and the loader are imported after the module registry
// is reset, so they are one consistent copy that has not loaded the rules module yet.
async function freshModules() {
  vi.resetModules();
  const React = await import("react");
  const testing = await import("@testing-library/react");
  const { useRules } = await import("@trenova/shared/hooks/use-rules");
  const load = await import("@trenova/shared/rules/load");

  function Version() {
    return React.createElement("p", null, useRules().version().version);
  }
  const tree = React.createElement(
    React.Suspense,
    { fallback: React.createElement("p", null, "loading") },
    React.createElement(Version),
  );
  return { testing, tree, load };
}

let cleanup: (() => void) | null = null;

afterEach(() => {
  cleanup?.();
  cleanup = null;
  vi.unstubAllGlobals();
});

describe("useRules", () => {
  it("suspends until the module loads, then renders with it synchronously", async () => {
    const fetchMock = vi.fn<typeof fetch>(() =>
      Promise.resolve(new Response(MODULE, { headers: { "content-type": "application/wasm" } })),
    );
    vi.stubGlobal("fetch", fetchMock);
    const { testing, tree, load } = await freshModules();
    cleanup = testing.cleanup;

    await testing.act(async () => {
      testing.render(tree);
    });
    const runtime = await testing.act(() => load.loadRules());

    expect(testing.screen.getByText(runtime.version().version)).toBeInTheDocument();
    expect(testing.screen.queryByText("loading")).toBeNull();
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("shows the fallback while the module is loading", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn<typeof fetch>(() => new Promise<Response>(() => {})),
    );
    const { testing, tree } = await freshModules();
    cleanup = testing.cleanup;

    await testing.act(async () => {
      testing.render(tree);
    });

    expect(testing.screen.getByText("loading")).toBeInTheDocument();
  });

  it("renders without suspending once the module has loaded", async () => {
    const { testing, tree, load } = await freshModules();
    cleanup = testing.cleanup;
    const runtime = await load.loadRules(MODULE);

    testing.render(tree);

    expect(testing.screen.getByText(runtime.version().version)).toBeInTheDocument();
    expect(testing.screen.queryByText("loading")).toBeNull();
  });
});
