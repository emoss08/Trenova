import { assistantChapterSchema } from "@/types/assistant";
import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { ChapterEntry } from "../chapter-work";
import { DeskChapterRail, RAIL_TICKS, tickWindow } from "../desk-chapter-rail";

afterEach(cleanup);

function entry(number: number, title: string, work: Partial<ChapterEntry["work"]> = {}): ChapterEntry {
  return {
    chapter: assistantChapterSchema.parse({
      id: `achp_${number}`,
      threadId: "athr_1",
      turnId: `atrn_${number}`,
      title,
      titleSource: "Model",
      version: 1,
      createdAt: number * 100,
      updatedAt: number * 100,
      startSequence: number * 4,
      startedAt: number * 100,
    }),
    number,
    work: { artifacts: 0, decisions: 0, waits: 0, ...work },
    live: false,
  };
}

const chapters = [
  entry(1, "Web test failures on five files", { artifacts: 2 }),
  entry(2, "Stale embedding fixture cleanup", { decisions: 1, waits: 1 }),
  entry(3, "Driver hours for Tuesday"),
];

describe("DeskChapterRail", () => {
  it("draws nothing for a conversation of one chapter", () => {
    const { container } = render(
      <DeskChapterRail entries={[chapters[0]]} nav={{ current: null, reveal: vi.fn() }} />,
    );

    expect(container).toBeEmptyDOMElement();
  });

  it("folds to a tick per chapter, the one in view longer and one waiting on the person marked", () => {
    const { container } = render(
      <DeskChapterRail entries={chapters} nav={{ current: "atrn_3", reveal: vi.fn() }} />,
    );

    const ticks = container.querySelectorAll(".dk-chap-tick");
    expect(ticks).toHaveLength(3);
    expect(ticks[2]).toHaveClass("dk-chap-tick-on");
    expect(ticks[1]).toHaveClass("dk-chap-tick-waits");
    expect(screen.getByRole("button", { name: "3 chapters" })).toHaveAttribute(
      "aria-expanded",
      "false",
    );
  });

  it("opens into the list, says what each chapter holds, and scrolls to the one picked", () => {
    const reveal = vi.fn();
    render(<DeskChapterRail entries={chapters} nav={{ current: "atrn_1", reveal }} />);

    fireEvent.click(screen.getByRole("button", { name: "3 chapters" }));
    const list = screen.getByRole("list");
    const rows = within(list).getAllByRole("button");

    expect(rows.map((row) => row.textContent)).toEqual([
      "1Web test failures on five files2 made",
      "2Stale embedding fixture cleanup1 to decide1 waiting",
      "3Driver hours for Tuesday",
    ]);
    expect(rows[0]).toHaveAttribute("aria-current", "location");

    fireEvent.click(rows[1]);

    expect(reveal).toHaveBeenCalledWith("atrn_2");
    expect(screen.getByRole("button", { name: "3 chapters" })).toHaveAttribute(
      "aria-expanded",
      "false",
    );
  });

  it("folds again on Escape", () => {
    render(<DeskChapterRail entries={chapters} nav={{ current: null, reveal: vi.fn() }} />);

    const toggle = screen.getByRole("button", { name: "3 chapters" });
    expect(screen.queryByRole("list")).toBeNull();
    fireEvent.click(toggle);
    fireEvent.keyDown(screen.getByRole("list"), { key: "Escape" });

    expect(toggle).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByRole("list")).toBeNull();
  });
});

describe("tickWindow", () => {
  it("draws every chapter while they fit, and a window around the one in view after", () => {
    const many = Array.from({ length: 30 }, (_, idx) => entry(idx + 1, `Chapter ${idx + 1}`));

    expect(tickWindow(chapters, 0)).toHaveLength(3);
    expect(tickWindow(many, 0).map((item) => item.number)).toEqual(
      Array.from({ length: RAIL_TICKS }, (_, idx) => idx + 1),
    );
    expect(tickWindow(many, 15).map((item) => item.number)[0]).toBe(10);
    expect(tickWindow(many, 29).map((item) => item.number).at(-1)).toBe(30);
  });
});
