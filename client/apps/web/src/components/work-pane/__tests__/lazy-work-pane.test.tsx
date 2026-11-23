import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState, type FunctionComponent } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  WorkPane,
  WorkPaneBody,
  WorkPaneCancel,
  WorkPaneFooter,
  WorkPaneHost,
  WorkPaneProvider,
  lazyWorkPane,
} from "..";

type PaneProps = { open: boolean; onOpenChange: (open: boolean) => void };

function NotePane({ open, onOpenChange }: PaneProps) {
  return (
    <WorkPane open={open} onOpenChange={onOpenChange} title="Add note">
      <WorkPaneBody>
        <input aria-label="Note" />
      </WorkPaneBody>
      <WorkPaneFooter>
        <WorkPaneCancel />
        <button type="submit">Add note</button>
      </WorkPaneFooter>
    </WorkPane>
  );
}

type NoteModule = { NotePane: FunctionComponent<PaneProps> };

/** A module whose arrival the test decides. */
function deferredModule() {
  let resolve: (module: NoteModule) => void = () => undefined;
  let reject: (error: Error) => void = () => undefined;
  const load = vi.fn(
    () =>
      new Promise<NoteModule>((done, fail) => {
        resolve = done;
        reject = fail;
      }),
  );
  return {
    load,
    arrive: () => resolve({ NotePane }),
    fail: (error: Error) => reject(error),
  };
}

function Page({ Pane }: { Pane: FunctionComponent<PaneProps> }) {
  const [open, setOpen] = useState(false);
  return (
    <WorkPaneProvider>
      <button type="button" onClick={() => setOpen(true)}>
        Open
      </button>
      <WorkPaneHost>
        <p>Notes</p>
        <Pane open={open} onOpenChange={setOpen} />
      </WorkPaneHost>
    </WorkPaneProvider>
  );
}

afterEach(() => {
  vi.restoreAllMocks();
});

describe("lazyWorkPane", () => {
  it("does not read the pane's module until the pane first opens", async () => {
    const module = deferredModule();
    const pane = lazyWorkPane(module.load, (loaded) => loaded.NotePane);
    render(<Page Pane={pane.Component} />);

    expect(module.load).not.toHaveBeenCalled();
    await userEvent.click(screen.getByRole("button", { name: "Open" }));
    expect(module.load).toHaveBeenCalledTimes(1);
  });

  it("shows the pane with its title and a form in shimmer while the code loads, then the form", async () => {
    const module = deferredModule();
    const pane = lazyWorkPane(module.load, (loaded) => loaded.NotePane, {
      title: () => "Add note",
    });
    render(<Page Pane={pane.Component} />);

    await userEvent.click(screen.getByRole("button", { name: "Open" }));
    const standIn = screen.getByRole("dialog", { name: "Add note" });
    expect(standIn).toBeInTheDocument();
    expect(screen.getByLabelText("Loading the form")).toHaveAttribute("aria-busy", "true");
    expect(screen.getByRole("button", { name: "Cancel" })).toBeEnabled();

    await act(async () => {
      module.arrive();
    });

    expect(await screen.findByLabelText("Note")).toBeInTheDocument();
    expect(screen.queryByLabelText("Loading the form")).toBeNull();
    expect(screen.getAllByRole("dialog")).toHaveLength(1);
  });

  it("opens at once, with no stand-in, when the code was preloaded", async () => {
    const load = vi.fn(async () => ({ NotePane }));
    const pane = lazyWorkPane(load, (loaded) => loaded.NotePane);
    render(<Page Pane={pane.Component} />);

    await pane.preload();
    await userEvent.click(screen.getByRole("button", { name: "Open" }));

    expect(screen.getByLabelText("Note")).toBeInTheDocument();
    expect(screen.queryByLabelText("Loading the form")).toBeNull();
    expect(load).toHaveBeenCalledTimes(1);
  });

  it("shows a failed read in the pane, and Try again reads it anew", async () => {
    vi.spyOn(console, "error").mockImplementation(() => undefined);
    let failing = true;
    const load = vi.fn(async (): Promise<NoteModule> => {
      if (failing) throw new Error("The pane could not be read");
      return { NotePane };
    });
    const pane = lazyWorkPane(load, (loaded) => loaded.NotePane, { title: () => "Add note" });
    render(<Page Pane={pane.Component} />);

    await userEvent.click(screen.getByRole("button", { name: "Open" }));
    const retry = await screen.findByRole("button", { name: /try again/i });
    expect(screen.getByRole("dialog", { name: "Add note" })).toBeInTheDocument();
    expect(screen.queryByLabelText("Note")).toBeNull();

    failing = false;
    await userEvent.click(retry);

    expect(await screen.findByLabelText("Note")).toBeInTheDocument();
  });

  it("mounts the pane fresh at each opening, without reading the code again", async () => {
    const load = vi.fn(async () => ({ NotePane }));
    const pane = lazyWorkPane(load, (loaded) => loaded.NotePane);
    render(<Page Pane={pane.Component} />);

    await userEvent.click(screen.getByRole("button", { name: "Open" }));
    await userEvent.type(await screen.findByLabelText("Note"), "Called the shop");
    await userEvent.click(screen.getByRole("button", { name: "Cancel" }));
    await waitFor(() => expect(screen.queryByLabelText("Note")).toBeNull());

    await userEvent.click(screen.getByRole("button", { name: "Open" }));

    expect(((await screen.findByLabelText("Note")) as HTMLInputElement).value).toBe("");
    expect(load).toHaveBeenCalledTimes(1);
  });
});
