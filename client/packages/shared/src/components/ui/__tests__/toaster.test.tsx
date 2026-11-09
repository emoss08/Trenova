import { toast } from "@trenova/shared/components/ui/toast";
import { useToastSoundStore } from "@trenova/shared/components/ui/toast-sound";
import { Toaster } from "@trenova/shared/components/ui/toaster";
import { act, cleanup, render, screen, waitFor, within } from "@testing-library/react";
import { toast as ding } from "@yusufceng1z/ding";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

vi.mock("@trenova/shared/i18n/use-t", () => ({
  useT: () => (message: string) => `[${message}]`,
}));

beforeAll(() => {
  // ding measures and springs the pill with ResizeObserver and the Web Animations API.
  if (!("ResizeObserver" in window)) {
    Object.defineProperty(window, "ResizeObserver", {
      configurable: true,
      value: class {
        observe() {}
        unobserve() {}
        disconnect() {}
      },
    });
  }
  if (typeof Element.prototype.animate !== "function") {
    Element.prototype.animate = function animate() {
      return {
        cancel() {},
        finish() {},
        finished: Promise.resolve(),
        addEventListener() {},
      } as unknown as Animation;
    };
  }
});

afterEach(() => {
  act(() => ding.dismiss(undefined, { immediate: true }));
  cleanup();
  useToastSoundStore.setState({ on: true });
  vi.restoreAllMocks();
});

describe("Toaster", () => {
  it("draws the toast's title and description in a region named in the person's language", () => {
    render(<Toaster />);

    act(() => {
      toast.success("Aging report is ready", { description: "AR aging · 1,284 invoices" });
    });

    const region = screen.getByRole("region", { name: "[Notifications]" });
    const card = region.querySelector<HTMLElement>(".tst");
    expect(card).toHaveAttribute("data-type", "success");
    expect(within(card as HTMLElement).getByText("Aging report is ready")).toBeInTheDocument();
    expect(within(card as HTMLElement).getByText("AR aging · 1,284 invoices")).toBeInTheDocument();
  });

  it("names the close button in the person's language", async () => {
    render(<Toaster />);

    act(() => {
      toast.error("Couldn't send the invoice");
    });

    await waitFor(() =>
      expect(screen.getByRole("button", { name: "[Dismiss notification]" })).toBeInTheDocument(),
    );
  });

  it("draws the toast dark in both themes, from the tokens", () => {
    const { container } = render(<Toaster position="top-center" />);

    const viewport = container.querySelector(".dark > .tst-viewport");
    expect(viewport).toHaveAttribute("data-theme", "dark");
    expect(viewport).toHaveAttribute("data-position", "top-center");
  });

  it("plays cues unless this person turned toast sounds off", () => {
    const configure = vi.spyOn(ding, "configure");
    render(<Toaster />);
    expect(configure).toHaveBeenCalledWith({ sound: true, volume: 0.35 });

    act(() => useToastSoundStore.setState({ on: false }));
    expect(configure).toHaveBeenLastCalledWith({ sound: false, volume: 0.35 });
  });
});
