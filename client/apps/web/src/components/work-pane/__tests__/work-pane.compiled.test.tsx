import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Button } from "@trenova/shared/components/ui/button";
import { Activity, useState } from "react";
import { useForm } from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  WorkPane,
  WorkPaneBody,
  WorkPaneCancel,
  WorkPaneDirtyTracker,
  WorkPaneFooter,
  WorkPaneHost,
  WorkPaneProvider,
  useWorkPane,
} from "..";

// Runs under the React Compiler, as the app is built: the pane's dirty guard follows a form
// through a tracker that reports into a ref, which memoization could freeze.

type Pane = "credential" | "event" | null;

function CredentialForm({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const form = useForm({ defaultValues: { number: "" } });
  return (
    <WorkPane
      open={open}
      onOpenChange={onOpenChange}
      title="Add credential"
      description="Record a license, card, endorsement or certificate."
    >
      <WorkPaneDirtyTracker form={form} />
      <form>
        <WorkPaneBody>
          <input aria-label="Number" {...form.register("number")} />
        </WorkPaneBody>
        <WorkPaneFooter
          leading={
            <Button type="button" variant="ghost" size="sm">
              Clear
            </Button>
          }
        >
          <WorkPaneCancel />
          <Button type="submit" size="sm">
            Add credential
          </Button>
        </WorkPaneFooter>
      </form>
    </WorkPane>
  );
}

function EventDetails({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <WorkPane open={open} onOpenChange={onOpenChange} title="Speeding, Nov 4">
      <WorkPaneBody>
        <p>Recorded by Dana Whitfield</p>
      </WorkPaneBody>
      <WorkPaneFooter>
        <Button type="button" size="sm" variant="secondary">
          Close event
        </Button>
      </WorkPaneFooter>
    </WorkPane>
  );
}

function Openers({ setPane }: { setPane: (pane: Pane) => void }) {
  const pane = useWorkPane();
  return (
    <>
      <button type="button" onClick={() => pane.closeThen(() => setPane("credential"))}>
        Open credential
      </button>
      <button type="button" onClick={() => pane.closeThen(() => setPane("event"))}>
        Open event
      </button>
      <span data-testid="state">{`${pane.isOpen ? "open" : "closed"} ${pane.isDirty ? "dirty" : "clean"}`}</span>
    </>
  );
}

function Page({
  initial = null,
  onDirtyChange,
  onClosed,
  tabShown = true,
}: {
  initial?: Pane;
  onDirtyChange?: (dirty: boolean) => void;
  onClosed?: (pane: Pane) => void;
  tabShown?: boolean;
}) {
  const [pane, setPane] = useState<Pane>(initial);
  const close = (which: Pane) => (open: boolean) => {
    if (!open) {
      setPane((current) => (current === which ? null : current));
      onClosed?.(which);
    }
  };
  return (
    <WorkPaneProvider onDirtyChange={onDirtyChange}>
      <Openers setPane={setPane} />
      <WorkPaneHost>
        <p>Credentials list</p>
        <Activity mode={tabShown ? "visible" : "hidden"}>
          <CredentialForm open={pane === "credential"} onOpenChange={close("credential")} />
          <EventDetails open={pane === "event"} onOpenChange={close("event")} />
        </Activity>
      </WorkPaneHost>
    </WorkPaneProvider>
  );
}

function row(): HTMLElement {
  const element = document.querySelector<HTMLElement>('[data-slot="work-pane-row"]');
  if (!element) throw new Error("no work pane row");
  return element;
}

function content(): HTMLElement {
  const element = document.querySelector<HTMLElement>('[data-slot="work-pane-content"]');
  if (!element) throw new Error("no work pane content box");
  return element;
}

/** happy-dom lays nothing out, so the row is given the width a browser would measure. */
function widthOfRow(width: number) {
  vi.spyOn(HTMLElement.prototype, "clientWidth", "get").mockImplementation(function (
    this: HTMLElement,
  ) {
    return this.dataset.slot === "work-pane-row" ? width : 0;
  });
}

function workPane(): HTMLElement {
  const element = document.querySelector<HTMLElement>('[data-slot="work-pane"]');
  if (!element) throw new Error("no work pane host");
  return element;
}

describe("WorkPane", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("opens in the host beside the content, titled, with focus on its first field", async () => {
    const user = userEvent.setup();
    render(<Page />);
    expect(workPane()).toHaveAttribute("aria-hidden", "true");

    await user.click(screen.getByRole("button", { name: "Open credential" }));

    const dialog = await screen.findByRole("dialog", { name: "Add credential" });
    expect(dialog).toHaveAccessibleDescription(
      "Record a license, card, endorsement or certificate.",
    );
    expect(workPane()).toContainElement(dialog);
    expect(workPane()).toHaveAttribute("aria-hidden", "false");
    expect(workPane()).not.toHaveAttribute("inert");
    await waitFor(() => expect(screen.getByLabelText("Number")).toHaveFocus());
    expect(screen.getByTestId("state")).toHaveTextContent("open clean");
  });

  it("closes on Escape when nothing has changed and gives focus back to the opener", async () => {
    const user = userEvent.setup();
    const onClosed = vi.fn();
    render(<Page onClosed={onClosed} />);
    const opener = screen.getByRole("button", { name: "Open credential" });

    await user.click(opener);
    await waitFor(() => expect(screen.getByLabelText("Number")).toHaveFocus());
    await user.keyboard("{Escape}");

    expect(onClosed).toHaveBeenCalledWith("credential");
    expect(workPane()).toHaveAttribute("aria-hidden", "true");
    await waitFor(() => expect(opener).toHaveFocus());
  });

  it("asks before discarding changes on Escape, keeps editing, then discards and closes", async () => {
    const user = userEvent.setup();
    const onClosed = vi.fn();
    const onDirtyChange = vi.fn();
    render(<Page onClosed={onClosed} onDirtyChange={onDirtyChange} />);

    await user.click(screen.getByRole("button", { name: "Open credential" }));
    await user.type(await screen.findByLabelText("Number"), "D1234567");
    await waitFor(() => expect(onDirtyChange).toHaveBeenLastCalledWith(true));
    expect(screen.getByTestId("state")).toHaveTextContent("open dirty");

    await user.keyboard("{Escape}");

    expect(onClosed).not.toHaveBeenCalled();
    expect(screen.getByRole("alert")).toHaveTextContent("Discard 1 unsaved change?");
    expect(screen.queryByRole("button", { name: "Cancel" })).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Keep editing" }));
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    expect(screen.getByLabelText("Number")).toHaveValue("D1234567");

    await user.click(screen.getByRole("button", { name: "Close" }));
    await user.click(screen.getByRole("button", { name: "Discard and close" }));

    expect(onClosed).toHaveBeenCalledWith("credential");
    await waitFor(() => expect(onDirtyChange).toHaveBeenLastCalledWith(false));
  });

  it("asks from Cancel too, and Escape on the question keeps editing", async () => {
    const user = userEvent.setup();
    const onClosed = vi.fn();
    render(<Page onClosed={onClosed} />);

    await user.click(screen.getByRole("button", { name: "Open credential" }));
    await user.type(await screen.findByLabelText("Number"), "7");
    await user.click(screen.getByRole("button", { name: "Cancel" }));
    expect(screen.getByRole("alert")).toHaveTextContent("Discard 1 unsaved change?");

    await user.keyboard("{Escape}");

    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    expect(onClosed).not.toHaveBeenCalled();
  });

  it("asks before opening another pane over a dirty one, and opens it once discarded", async () => {
    const user = userEvent.setup();
    render(<Page />);

    await user.click(screen.getByRole("button", { name: "Open credential" }));
    await user.type(await screen.findByLabelText("Number"), "7");
    await user.click(screen.getByRole("button", { name: "Open event" }));

    expect(screen.getByRole("dialog", { name: "Add credential" })).toBeInTheDocument();
    expect(screen.getByRole("alert")).toHaveTextContent("Discard 1 unsaved change?");

    await user.click(screen.getByRole("button", { name: "Discard and close" }));

    expect(await screen.findByRole("dialog", { name: "Speeding, Nov 4" })).toBeInTheDocument();
    expect(screen.queryByRole("dialog", { name: "Add credential" })).not.toBeInTheDocument();
    expect(screen.getByTestId("state")).toHaveTextContent("open clean");
  });

  it("swaps a clean pane for another without asking", async () => {
    const user = userEvent.setup();
    render(<Page initial="credential" />);
    await screen.findByRole("dialog", { name: "Add credential" });

    await user.click(screen.getByRole("button", { name: "Open event" }));

    expect(await screen.findByRole("dialog", { name: "Speeding, Nov 4" })).toBeInTheDocument();
    expect(screen.getByText("Recorded by Dana Whitfield")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Close event" })).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("lies over the content with a scrim that closes it under 1100px of row width", async () => {
    widthOfRow(1099);
    const user = userEvent.setup();
    const onClosed = vi.fn();
    render(<Page initial="event" onClosed={onClosed} />);
    await screen.findByRole("dialog", { name: "Speeding, Nov 4" });

    expect(row()).toHaveAttribute("data-sheet", "true");
    expect(workPane().style.getPropertyValue("--side-pane-width")).toBe("min(26rem, 1099px)");
    expect(workPane()).toHaveClass("translate-x-0", "ml-[calc(-1*var(--side-pane-width))]");
    const scrim = document.querySelector<HTMLElement>('[data-slot="work-pane-scrim"]');
    expect(scrim).not.toBeNull();

    await user.click(scrim!);

    expect(onClosed).toHaveBeenCalledWith("event");
  });

  it("opens beside the content, with no scrim, from 1100px of row width", async () => {
    widthOfRow(1100);
    render(<Page initial="event" />);
    await screen.findByRole("dialog", { name: "Speeding, Nov 4" });

    expect(row()).not.toHaveAttribute("data-sheet");
    expect(workPane().style.getPropertyValue("--side-pane-width")).toBe("26rem");
    expect(workPane()).not.toHaveClass("translate-x-0");
    expect(document.querySelector('[data-slot="work-pane-scrim"]')).toBeNull();
  });

  it("never traps the fixed-position docks and panels of the content it hosts", async () => {
    widthOfRow(900);
    render(<Page initial="event" />);
    await screen.findByRole("dialog", { name: "Speeding, Nov 4" });

    // A container query, containment, a transform, a filter or a perspective on either box makes
    // it the containing block of every fixed descendant: a tab's save dock would sit at the
    // bottom of the tab instead of on the viewport.
    const trapping =
      /(^|:)@container|(^|:)contain-|(^|:)(-?translate|scale|rotate|skew|transform|perspective|filter|blur|backdrop|will-change)/;
    for (const box of [row(), content()]) {
      const classes = Array.from(box.classList);
      expect(classes.filter((name) => trapping.test(name))).toEqual([]);
      expect(box.getAttribute("style") ?? "").toBe("");
    }
    expect(content()).toHaveTextContent("Credentials list");
  });

  it("gives up the host while its tab is hidden", async () => {
    const { rerender } = render(<Page initial="event" />);
    await screen.findByRole("dialog", { name: "Speeding, Nov 4" });

    rerender(<Page initial="event" tabShown={false} />);

    await waitFor(() => expect(workPane()).toHaveAttribute("aria-hidden", "true"));
    expect(screen.getByTestId("state")).toHaveTextContent("closed clean");

    await act(async () => {
      rerender(<Page initial="event" tabShown />);
    });

    expect(await screen.findByRole("dialog", { name: "Speeding, Nov 4" })).toBeInTheDocument();
  });
});
