import { describe, expect, it, vi } from "vitest";
import { createBatchLoader } from "../batch-loader";

describe("createBatchLoader", () => {
  it("answers every key asked for in one tick with one fetch, asking each key once", async () => {
    const fetch = vi.fn(async (keys: readonly string[]) => new Map(keys.map((key) => [key, key.length])));
    const loader = createBatchLoader(fetch);

    const answers = await Promise.all([loader.load("a"), loader.load("bb"), loader.load("a")]);

    expect(answers).toEqual([1, 2, 1]);
    expect(fetch).toHaveBeenCalledTimes(1);
    expect(fetch).toHaveBeenCalledWith(["a", "bb"]);
  });

  it("splits a large tick into batches no bigger than the limit", async () => {
    const fetch = vi.fn(async (keys: readonly number[]) => new Map(keys.map((key) => [key, key])));
    const loader = createBatchLoader(fetch, { maxBatchSize: 2 });

    await Promise.all([1, 2, 3, 4, 5].map((key) => loader.load(key)));

    expect(fetch.mock.calls.map(([keys]) => keys)).toEqual([[1, 2], [3, 4], [5]]);
  });

  it("resolves a key the answer leaves out as undefined and rejects every key when the fetch fails", async () => {
    const loader = createBatchLoader(async () => new Map([["kept", 1]]));
    await expect(loader.load("missing")).resolves.toBeUndefined();

    const failing = createBatchLoader<string, number>(async () => {
      throw new Error("offline");
    });
    await expect(Promise.all([failing.load("a"), failing.load("b")])).rejects.toThrow("offline");
  });

  it("starts a fresh batch for keys asked after the last one was sent", async () => {
    const fetch = vi.fn(async (keys: readonly string[]) => new Map(keys.map((key) => [key, 0])));
    const loader = createBatchLoader(fetch);

    await loader.load("first");
    await loader.load("second");

    expect(fetch).toHaveBeenCalledTimes(2);
  });
});
