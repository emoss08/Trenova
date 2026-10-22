import { RULES_WASM_BYTES as MODULE } from "@trenova/shared/rules/test-module";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

type LoadModule = typeof import("./load");
type RuntimeModule = typeof import("./runtime");

// Each test gets the loader as a page first sees it, before anything has loaded: the test
// setup has already loaded the module into the copy every other test shares.
async function freshLoader(): Promise<{ load: LoadModule; runtime: RuntimeModule }> {
  vi.resetModules();
  const runtime = await import("./runtime");
  const load = await import("./load");
  return { load, runtime };
}

function wasmResponse(contentType = "application/wasm"): Response {
  return new Response(MODULE, { status: 200, headers: { "content-type": contentType } });
}

describe("loadRules", () => {
  const fetchMock = vi.fn<typeof fetch>();

  beforeEach(() => {
    fetchMock.mockReset();
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("makes rulesSync throw before the module loads and answer after", async () => {
    const { load, runtime } = await freshLoader();
    expect(runtime.isRulesLoaded()).toBe(false);
    expect(() => runtime.rulesSync()).toThrow(runtime.RulesNotLoadedError);

    const loaded = await load.loadRules(MODULE);

    expect(runtime.isRulesLoaded()).toBe(true);
    expect(runtime.rulesSync()).toBe(loaded);
    expect(runtime.rulesSync().version().operations).toContain(runtime.RULES_VERSION_OP);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("fetches rules.wasm next to the loader and hands a wasm response to the streaming compiler", async () => {
    const { load } = await freshLoader();
    const streaming = vi.spyOn(WebAssembly, "instantiateStreaming");
    fetchMock.mockResolvedValue(wasmResponse());

    const loaded = await load.loadRules();

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0]!;
    expect(String(url)).toMatch(/\/rules\/rules\.wasm$/);
    expect(init).toEqual({ credentials: "same-origin" });
    expect(streaming).toHaveBeenCalledTimes(1);
    expect(loaded.version().version).not.toBe("");
  });

  it("falls back to compiling the whole file when the server sends the wrong type", async () => {
    const { load } = await freshLoader();
    const streaming = vi.spyOn(WebAssembly, "instantiateStreaming");
    fetchMock.mockResolvedValue(wasmResponse("application/octet-stream"));

    const loaded = await load.loadRules();

    expect(streaming).not.toHaveBeenCalled();
    expect(loaded.version().operations).toContain("rules.version");
  });

  it("returns one promise to every caller while a load is in flight", async () => {
    const { load } = await freshLoader();
    fetchMock.mockResolvedValue(wasmResponse());

    const first = load.loadRules();
    const second = load.loadRules();

    expect(second).toBe(first);
    await expect(first).resolves.toBe(await second);
    expect(load.loadRules()).toBe(first);
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("forgets a failed fetch and fetches again on the next call", async () => {
    const { load, runtime } = await freshLoader();
    fetchMock.mockRejectedValueOnce(new TypeError("Failed to fetch"));
    fetchMock.mockResolvedValueOnce(wasmResponse());

    const failed = load.loadRules();
    await expect(failed).rejects.toBeInstanceOf(load.RulesLoadError);
    expect(runtime.isRulesLoaded()).toBe(false);

    const retried = load.loadRules();
    expect(retried).not.toBe(failed);
    expect(await retried).toBe(runtime.rulesSync());
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("treats an error status as a failed load and retries it", async () => {
    const { load, runtime } = await freshLoader();
    fetchMock.mockResolvedValueOnce(new Response("Not Found", { status: 404 }));
    fetchMock.mockResolvedValueOnce(wasmResponse());

    await expect(load.loadRules()).rejects.toThrow("rules: fetching rules.wasm answered 404");
    expect(await load.loadRules()).toBe(runtime.rulesSync());
  });

  it("forgets a module that fails to compile", async () => {
    const { load, runtime } = await freshLoader();
    fetchMock.mockResolvedValueOnce(new Response("<!doctype html>", { status: 200 }));
    fetchMock.mockResolvedValueOnce(wasmResponse());

    await expect(load.loadRules()).rejects.toBeInstanceOf(WebAssembly.CompileError);
    expect(await load.loadRules()).toBe(runtime.rulesSync());
  });

  it("tells subscribers once the module has loaded", async () => {
    const { load, runtime } = await freshLoader();
    const listener = vi.fn();
    const unsubscribe = runtime.subscribeRules(listener);
    expect(runtime.getRulesSnapshot()).toBeNull();

    const loaded = await load.loadRules(MODULE);

    expect(listener).toHaveBeenCalledTimes(1);
    expect(runtime.getRulesSnapshot()).toBe(loaded);
    unsubscribe();
  });
});
