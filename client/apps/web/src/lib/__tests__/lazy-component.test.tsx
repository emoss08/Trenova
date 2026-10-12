import { act, render, screen } from "@testing-library/react";
import { Suspense } from "react";
import { describe, expect, it, vi } from "vitest";
import { lazyComponent, preloadable } from "../lazy-component";

describe("preloadable", () => {
  it("reads the module once however often it is asked for", async () => {
    const load = vi.fn(async () => ({ value: 1 }));
    const read = preloadable(load);

    const first = read();
    const second = read();

    expect(second).toBe(first);
    await first;
    await read();
    expect(load).toHaveBeenCalledTimes(1);
  });

  it("forgets a failed read so the next call asks again", async () => {
    const load = vi
      .fn<() => Promise<{ value: number }>>()
      .mockRejectedValueOnce(new Error("offline"))
      .mockResolvedValueOnce({ value: 2 });
    const read = preloadable(load);

    await expect(read()).rejects.toThrow("offline");
    await expect(read()).resolves.toEqual({ value: 2 });
    expect(load).toHaveBeenCalledTimes(2);
  });
});

describe("lazyComponent", () => {
  it("draws the fallback until the code is read, then the component", async () => {
    let open: (module: { Shown: () => React.ReactNode }) => void = () => {};
    const loaded = new Promise<{ Shown: () => React.ReactNode }>((resolve) => {
      open = resolve;
    });
    const { Component } = lazyComponent(
      () => loaded,
      (module) => module.Shown,
    );

    render(
      <Suspense fallback={<p>Outline</p>}>
        <Component />
      </Suspense>,
    );
    expect(screen.getByText("Outline")).toBeInTheDocument();

    await act(async () => {
      open({ Shown: () => <p>Card</p> });
      await loaded;
    });

    expect(await screen.findByText("Card")).toBeInTheDocument();
    expect(screen.queryByText("Outline")).not.toBeInTheDocument();
  });

  it("shares one read between a preload and the drawing", async () => {
    const load = vi.fn(async () => ({ Shown: () => <p>Card</p> }));
    const { Component, preload } = lazyComponent(load, (module) => module.Shown);

    await preload();
    render(
      <Suspense fallback={<p>Outline</p>}>
        <Component />
      </Suspense>,
    );

    expect(await screen.findByText("Card")).toBeInTheDocument();
    expect(load).toHaveBeenCalledTimes(1);
  });
});
