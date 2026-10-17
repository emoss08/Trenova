import { toast } from "@trenova/shared/components/ui/toast";
import { toastStore } from "@trenova/shared/components/ui/toast-store";
import { Toaster } from "@trenova/shared/components/ui/toaster";
import { act, cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { LazyMotion, MotionGlobalConfig, domMax } from "motion/react";
import type { ReactNode } from "react";
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from "vitest";

function withMotion(children: ReactNode) {
  return <LazyMotion features={domMax}>{children}</LazyMotion>;
}

function titles() {
  return toastStore.getSnapshot().toasts.map((entry) => entry.title);
}

function cardsWith(text: string): HTMLElement[] {
  return screen
    .queryAllByText(text)
    .map((element) => element.closest<HTMLElement>("[data-slot='toast']"))
    .filter((card): card is HTMLElement => card !== null);
}

function cardOf(text: string): HTMLElement {
  const cards = cardsWith(text);
  expect(cards).toHaveLength(1);
  return cards[0] as HTMLElement;
}

beforeAll(() => {
  MotionGlobalConfig.skipAnimations = true;
});

afterAll(() => {
  MotionGlobalConfig.skipAnimations = false;
});

afterEach(() => {
  act(() => toast.dismiss());
  cleanup();
  vi.useRealTimers();
});

describe("Toaster", () => {
  it("shows a toast's title and description", () => {
    render(withMotion(<Toaster />));

    act(() => {
      toast.success("Aging report is ready", { description: "AR aging · 1,284 invoices" });
    });

    const card = cardOf("Aging report is ready");
    expect(within(card).getByText("AR aging · 1,284 invoices")).toBeInTheDocument();
    expect(card).toHaveAttribute("data-type", "success");
  });

  it("runs the action and closes the toast", async () => {
    const download = vi.fn();
    render(withMotion(<Toaster />));
    act(() => {
      toast.success("Aging report is ready", { action: { label: "Download", onClick: download } });
    });

    await userEvent.click(screen.getByRole("button", { name: "Download" }));

    expect(download).toHaveBeenCalledTimes(1);
    expect(titles()).toEqual([]);
    await waitFor(() => expect(cardsWith("Aging report is ready")).toEqual([]));
  });

  it("stays open when the action prevents the default", async () => {
    render(withMotion(<Toaster />));
    act(() => {
      toast("Shipment S-104882 archived", {
        action: { label: "Undo", onClick: (event) => event.preventDefault() },
      });
    });

    await userEvent.click(screen.getByRole("button", { name: "Undo" }));

    expect(titles()).toEqual(["Shipment S-104882 archived"]);
  });

  it("closes from its Dismiss button and from Escape", async () => {
    const onDismiss = vi.fn();
    render(withMotion(<Toaster />));
    act(() => {
      toast("First", { onDismiss });
      toast("Second", { onDismiss });
    });

    await userEvent.click(within(cardOf("Second")).getByRole("button", { name: "Dismiss" }));
    expect(titles()).toEqual(["First"]);

    const first = cardOf("First");
    act(() => first.focus());
    fireEvent.keyDown(first, { key: "Escape" });

    expect(titles()).toEqual([]);
    expect(onDismiss).toHaveBeenCalledTimes(2);
  });

  it("runs out after its duration", () => {
    vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout", "Date"] });
    const onAutoClose = vi.fn();
    render(withMotion(<Toaster />));
    act(() => {
      toast("Tracking link copied", { duration: 2000, onAutoClose });
    });

    act(() => vi.advanceTimersByTime(1999));
    expect(titles()).toEqual(["Tracking link copied"]);

    act(() => vi.advanceTimersByTime(1));
    expect(titles()).toEqual([]);
    expect(onAutoClose).toHaveBeenCalledTimes(1);
  });

  it("holds its time while focus is inside it, and counts on from where it stopped", () => {
    vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout", "Date"] });
    render(
      withMotion(
        <>
          <button type="button">Elsewhere</button>
          <Toaster />
        </>,
      ),
    );
    act(() => {
      toast("Template duplicated", { duration: 2000 });
    });

    act(() => vi.advanceTimersByTime(500));
    act(() => cardOf("Template duplicated").focus());
    act(() => vi.advanceTimersByTime(10000));
    expect(titles()).toEqual(["Template duplicated"]);

    act(() => screen.getByRole("button", { name: "Elsewhere" }).focus());
    act(() => vi.advanceTimersByTime(1499));
    expect(titles()).toEqual(["Template duplicated"]);
    act(() => vi.advanceTimersByTime(1));
    expect(titles()).toEqual([]);
  });

  it("restarts the clock when the same news repeats, and counts it", () => {
    vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout", "Date"] });
    render(withMotion(<Toaster />));
    act(() => {
      toast.error("Couldn't reach PC*MILER", { duration: 3000 });
    });
    act(() => vi.advanceTimersByTime(2000));
    act(() => {
      toast.error("Couldn't reach PC*MILER", { duration: 3000 });
      toast.error("Couldn't reach PC*MILER", { duration: 3000 });
    });

    expect(cardsOf("Couldn't reach PC*MILER")).toHaveLength(1);
    expect(within(cardOf("Couldn't reach PC*MILER")).getByText("×3")).toBeInTheDocument();

    act(() => vi.advanceTimersByTime(2999));
    expect(titles()).toEqual(["Couldn't reach PC*MILER"]);
    act(() => vi.advanceTimersByTime(1));
    expect(titles()).toEqual([]);
  });

  it("opens Details to show the reference, and copies it", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", { value: { writeText }, configurable: true });
    render(withMotion(<Toaster />));
    act(() => {
      toast.error("Rate confirmation didn't send", {
        description: "The carrier's mail server refused it.",
        details: [
          { label: "Reference", value: "trace-01JAB7K9QX", monospace: true },
          { label: "Shipment", value: "S-104882" },
        ],
      });
    });
    expect(screen.queryByText("trace-01JAB7K9QX")).not.toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "Details" }));
    expect(screen.getByText("trace-01JAB7K9QX")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Hide details" })).toHaveAttribute(
      "aria-expanded",
      "true",
    );

    await userEvent.click(screen.getByRole("button", { name: "Copy details" }));

    expect(writeText).toHaveBeenCalledWith("Reference: trace-01JAB7K9QX\nShipment: S-104882");
    expect(await screen.findByRole("button", { name: "Copied" })).toBeInTheDocument();
    expect(titles()).toEqual(["Rate confirmation didn't send"]);
  });

  it("announces errors assertively and other news politely", async () => {
    const { container } = render(withMotion(<Toaster />));
    const polite = container.querySelector("[aria-live='polite']");
    const assertive = container.querySelector("[aria-live='assertive']");

    act(() => {
      toast.success("Aging report is ready", { description: "1,284 invoices" });
    });
    await waitFor(() => expect(polite).toHaveTextContent("Aging report is ready. 1,284 invoices"));

    act(() => {
      toast.error("Couldn't send");
    });
    await waitFor(() => expect(assertive).toHaveTextContent("Couldn't send"));
  });

  it("moves focus to the newest toast on Alt+T", () => {
    render(withMotion(<Toaster />));
    act(() => {
      toast("Older");
      toast("Newest");
    });

    fireEvent.keyDown(document, { key: "t", code: "KeyT", altKey: true });

    expect(document.activeElement).toBe(cardOf("Newest"));
  });

  it("shows a named Toaster's toasts there, and in the app's Toaster once it is gone", () => {
    render(withMotion(<Toaster />));
    const named = render(withMotion(<Toaster id="user-settings" position="bottom-center" />));
    const [appRegion, namedRegion] = screen.getAllByRole("region", { name: "Notifications" });

    act(() => {
      toast.success("Saved", { toasterId: "user-settings" });
    });
    expect(within(namedRegion).getByText("Saved")).toBeInTheDocument();
    expect(within(appRegion).queryByText("Saved")).not.toBeInTheDocument();

    named.unmount();

    expect(within(appRegion).getByText("Saved")).toBeInTheDocument();
  });
});
