import { act, cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useMediaQuery } from "../use-media-query";

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

function Probe({ label }: { label: string }) {
  const desktop = useMediaQuery("(min-width: 640px)");
  return <span>{`${label}:${desktop ? "desktop" : "phone"}`}</span>;
}

describe("useMediaQuery", () => {
  it("shares one listener per query however many components ask", () => {
    const addEventListener = vi.fn();
    const removeEventListener = vi.fn();
    const matchMedia = vi.fn(() => ({ matches: true, addEventListener, removeEventListener }));
    vi.stubGlobal("matchMedia", matchMedia);

    const { unmount } = render(
      <>
        {Array.from({ length: 40 }, (_, i) => (
          <Probe key={i} label={String(i)} />
        ))}
      </>,
    );

    expect(screen.getByText("0:desktop")).toBeTruthy();
    expect(addEventListener).toHaveBeenCalledTimes(1);
    expect(matchMedia).toHaveBeenCalledTimes(1);

    unmount();
    expect(removeEventListener).toHaveBeenCalledTimes(1);
  });

  it("tells every component when the query starts or stops matching", () => {
    let matches = false;
    let onChange: (() => void) | undefined;
    vi.stubGlobal(
      "matchMedia",
      vi.fn(() => ({
        get matches() {
          return matches;
        },
        addEventListener: (_: string, listener: () => void) => {
          onChange = listener;
        },
        removeEventListener: vi.fn(),
      })),
    );

    render(
      <>
        <Probe label="a" />
        <Probe label="b" />
      </>,
    );
    expect(screen.getByText("a:phone")).toBeTruthy();

    matches = true;
    act(() => onChange?.());

    expect(screen.getByText("a:desktop")).toBeTruthy();
    expect(screen.getByText("b:desktop")).toBeTruthy();
  });
});
