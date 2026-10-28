import { assistantChapterSchema } from "@/types/assistant";
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { resetStreamedTitles } from "../chapter-title-stream";
import type { ChapterEntry } from "../chapter-work";
import { DeskChapterHeading, type ChapterActions } from "../desk-chapter-heading";

vi.mock("motion/react", async (importOriginal) => ({
  ...(await importOriginal<typeof import("motion/react")>()),
  useReducedMotion: () => false,
}));

afterEach(() => {
  cleanup();
  vi.useRealTimers();
  resetStreamedTitles();
});

function entry(overrides: Partial<ChapterEntry["chapter"]> = {}, number = 2): ChapterEntry {
  return {
    chapter: assistantChapterSchema.parse({
      id: "achp_2",
      threadId: "athr_1",
      turnId: "atrn_2",
      title: "Acme invoice dispute",
      titleSource: "Model",
      version: 3,
      createdAt: 1_761_600_000,
      updatedAt: 1_761_600_000,
      startSequence: 8,
      startedAt: 1_761_600_000,
      ...overrides,
    }),
    number,
    work: { artifacts: 0, decisions: 0, waits: 0 },
    live: false,
  };
}

function actions(overrides: Partial<ChapterActions> = {}): ChapterActions {
  return {
    rename: vi.fn(),
    merge: vi.fn(),
    agents: [],
    busy: false,
    ...overrides,
  };
}

function renderHeading(value: ChapterEntry, given: ChapterActions) {
  return render(<DeskChapterHeading entry={value} actions={given} timezone="America/Chicago" />);
}

describe("DeskChapterHeading", () => {
  it("shows the chapter's number and title as a heading", () => {
    renderHeading(entry(), actions());

    expect(screen.getByRole("heading", { level: 2 })).toHaveTextContent("Acme invoice dispute");
    expect(screen.getByText("02")).toBeInTheDocument();
  });

  it("turns into a field on a double-click and keeps the new title on Enter", () => {
    const given = actions();
    const value = entry();
    renderHeading(value, given);

    fireEvent.doubleClick(screen.getByRole("button", { name: "Acme invoice dispute" }));
    const field = screen.getByRole("textbox", { name: "Chapter title" });
    fireEvent.change(field, { target: { value: "  Acme   short-pay on 4471  " } });
    fireEvent.keyDown(field, { key: "Enter" });

    expect(given.rename).toHaveBeenCalledWith(value, "Acme short-pay on 4471");
    expect(screen.getByRole("heading", { level: 2 })).toHaveTextContent("Acme short-pay on 4471");
  });

  it("keeps the old title on Escape, and sends nothing for a blank or unchanged one", () => {
    const given = actions();
    renderHeading(entry(), given);

    fireEvent.keyDown(screen.getByRole("button", { name: "Acme invoice dispute" }), { key: "F2" });
    const field = screen.getByRole("textbox", { name: "Chapter title" });
    fireEvent.change(field, { target: { value: "Something else" } });
    fireEvent.keyDown(field, { key: "Escape" });

    expect(screen.getByRole("heading", { level: 2 })).toHaveTextContent("Acme invoice dispute");

    fireEvent.keyDown(screen.getByRole("button", { name: "Acme invoice dispute" }), {
      key: "Enter",
    });
    fireEvent.change(screen.getByRole("textbox", { name: "Chapter title" }), {
      target: { value: "   " },
    });
    fireEvent.blur(screen.getByRole("textbox", { name: "Chapter title" }));

    expect(given.rename).not.toHaveBeenCalled();
  });

  it("types in a title the model has just written, as the briefing does", () => {
    vi.useFakeTimers();
    renderHeading(
      entry({ title: "Acme invoice dispute", updatedAt: Math.floor(Date.now() / 1000) }),
      actions(),
    );

    const heading = screen.getByRole("heading", { level: 2 });
    expect(heading).not.toHaveTextContent("Acme invoice dispute");
    expect(screen.getByRole("button", { name: "Acme invoice dispute" })).toBeInTheDocument();

    act(() => {
      vi.advanceTimersByTime(1_000);
    });

    expect(heading).toHaveTextContent("Acme invoice dispute");
  });

  // The heading is drawn under the live turn, then again among the saved
  // messages once the turn is saved, moments after the model named it. Typing
  // the title in again there read as the title being written afresh.
  it("types a title in once, and shows it whole when the heading is drawn again", () => {
    vi.useFakeTimers();
    const fresh = entry({
      title: "Acme invoice dispute",
      updatedAt: Math.floor(Date.now() / 1000),
    });
    const first = renderHeading(fresh, actions());
    act(() => {
      vi.advanceTimersByTime(1_000);
    });
    first.unmount();

    renderHeading(fresh, actions());

    expect(screen.getByRole("heading", { level: 2 })).toHaveTextContent("Acme invoice dispute");
  });

  it("types in a new title the model gives a chapter already on screen", () => {
    vi.useFakeTimers();
    const given = actions();
    const view = renderHeading(entry({ title: "Acme invoice dispute" }), given);
    expect(screen.getByRole("heading", { level: 2 })).toHaveTextContent("Acme invoice dispute");

    view.rerender(
      <DeskChapterHeading
        entry={entry({ title: "Acme short-pay on INV-1042", version: 4 })}
        actions={given}
        timezone="America/Chicago"
      />,
    );

    const heading = screen.getByRole("heading", { level: 2 });
    expect(heading).not.toHaveTextContent("Acme short-pay on INV-1042");
    act(() => {
      vi.advanceTimersByTime(1_000);
    });
    expect(heading).toHaveTextContent("Acme short-pay on INV-1042");
  });

  it("shows a title from before the page opened whole at once", () => {
    vi.useFakeTimers();
    renderHeading(entry({ title: "Acme invoice dispute" }), actions());

    expect(screen.getByRole("heading", { level: 2 })).toHaveTextContent("Acme invoice dispute");
  });

  it("says when a scheduled run opened the chapter", () => {
    renderHeading(entry({ titleSource: "Schedule", title: "AR aging over 60 days" }), actions());

    expect(screen.getByText("Oct 27")).toBeInTheDocument();
  });
});
