import { act, fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { NeedsLens } from "../lens";
import { NeedsHead } from "../needs-head";

const ALL: readonly NeedsLens[] = ["Everything", "Watchtower", "Decisions"];
const COUNTS = { Everything: 987, Watchtower: 722, Decisions: 253 };

function renderHead(props: Partial<Parameters<typeof NeedsHead>[0]> = {}) {
  const onLens = vi.fn<(lens: NeedsLens) => void>();
  render(
    <NeedsHead
      lens="Everything"
      lenses={ALL}
      onLens={onLens}
      counts={COUNTS}
      onOpenFilter={vi.fn()}
      progress={{ label: "4 of 10 cleared today", done: 4, total: 10 }}
      whose={null}
      search={{ value: "", onCommit: vi.fn() }}
      filter={null}
      tokens={[]}
      onClearFilters={vi.fn()}
      {...props}
    />,
  );
  return onLens;
}

const press = (key: string, target: Element = document.body) =>
  act(() => void fireEvent.keyDown(target, { key }));

describe("NeedsHead, the head every lens of Needs you shares", () => {
  it("names each lens with what waits in it and marks the one in front", () => {
    renderHead({ lens: "Decisions" });

    expect(screen.getByRole("tab", { name: "Everything 987" })).toHaveAttribute(
      "aria-selected",
      "false",
    );
    expect(screen.getByRole("tab", { name: "Watchtower 722" })).toHaveAttribute(
      "aria-selected",
      "false",
    );
    expect(screen.getByRole("tab", { name: "Decisions 253" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
  });

  it("says how much of today's work is done, as a progress bar", () => {
    renderHead();

    const bar = screen.getByRole("progressbar", { name: "4 of 10 cleared today" });
    expect(bar).toHaveAttribute("aria-valuenow", "4");
    expect(bar).toHaveAttribute("aria-valuemax", "10");
  });

  it("picks a lens by its tab and by the keys 1 to 3", () => {
    const onLens = renderHead();

    fireEvent.click(screen.getByRole("tab", { name: "Watchtower 722" }));
    press("3");
    press("1");

    expect(onLens.mock.calls).toEqual([["Watchtower"], ["Decisions"], ["Everything"]]);
  });

  it("shows only the lenses the person may open and keeps each lens on its own key", () => {
    const onLens = renderHead({ lens: "Decisions", lenses: ["Decisions"] });

    expect(screen.getAllByRole("tab")).toHaveLength(1);
    expect(screen.getByRole("tab", { name: "Decisions 253" })).toHaveAttribute("title", "Press 3");
    press("1");
    press("2");
    expect(onLens).not.toHaveBeenCalled();
    press("3");
    expect(onLens.mock.calls).toEqual([["Decisions"]]);
  });

  it("leaves the keys alone while someone types or a dialog is open", () => {
    const onLens = renderHead();
    const input = document.createElement("input");
    const dialog = document.createElement("div");
    dialog.setAttribute("role", "dialog");
    const inside = document.createElement("button");
    dialog.append(inside);
    document.body.append(input, dialog);

    press("2", input);
    press("2", inside);
    expect(onLens).not.toHaveBeenCalled();

    input.remove();
    dialog.remove();
  });

  it("ignores the keys when the page turns them off", () => {
    const onLens = renderHead({ keysEnabled: false });

    press("2");

    expect(onLens).not.toHaveBeenCalled();
  });

  it("draws a token for each filter that is on, each taken off on its own, and all at once", () => {
    const onClearFilters = vi.fn();
    const removeCritical = vi.fn();
    const removeDetention = vi.fn();
    renderHead({
      onClearFilters,
      tokens: [
        { key: "severity:Critical", label: "Critical", onRemove: removeCritical },
        { key: "kind:DetentionOccurrence", label: "Detention", onRemove: removeDetention },
      ],
    });

    const filters = screen.getByRole("group", { name: "Filters" });
    expect(filters).toHaveTextContent("CriticalDetentionClear");
    fireEvent.click(screen.getByRole("button", { name: "Remove the filter Detention" }));
    expect(removeDetention).toHaveBeenCalledTimes(1);
    expect(removeCritical).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole("button", { name: "Clear" }));
    expect(onClearFilters).toHaveBeenCalledTimes(1);
  });

  it("draws no filter row while nothing is filtered", () => {
    renderHead({ onClearFilters: vi.fn(), tokens: [] });

    expect(screen.queryByRole("group", { name: "Filters" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Clear" })).toBeNull();
  });

  it("hands the page what is typed once typing stops, and goes to the search on /", async () => {
    vi.useFakeTimers();
    const onCommit = vi.fn<(value: string) => void>();
    renderHead({ search: { value: "", onCommit } });

    const box = screen.getByRole("textbox", { name: "Search what needs you" });
    press("/");
    expect(box).toHaveFocus();

    fireEvent.change(box, { target: { value: "SEED-DET" } });
    expect(onCommit).not.toHaveBeenCalled();
    await act(() => vi.advanceTimersByTimeAsync(300));
    expect(onCommit.mock.calls).toEqual([["SEED-DET"]]);
    vi.useRealTimers();
  });

  it("commits what is typed at once on Enter and gives the keys back to the list", () => {
    const onCommit = vi.fn<(value: string) => void>();
    renderHead({ search: { value: "", onCommit } });

    const box = screen.getByRole("textbox", { name: "Search what needs you" });
    box.focus();
    fireEvent.change(box, { target: { value: "phoenix" } });
    fireEvent.keyDown(box, { key: "Enter" });

    expect(onCommit.mock.calls).toEqual([["phoenix"]]);
    expect(box).not.toHaveFocus();
  });

  it("searches under every lens, Decisions with the rest", () => {
    renderHead({ lens: "Decisions", search: { value: "acme", onCommit: vi.fn() } });

    const box = screen.getByRole("textbox", { name: "Search what needs you" });
    expect(box).toHaveValue("acme");
    press("/");
    expect(box).toHaveFocus();
  });

  it("tells the page which lens the pointer or the focus is heading for", () => {
    const onLensIntent = vi.fn<(lens: NeedsLens) => void>();
    renderHead({ onLensIntent });

    fireEvent.pointerEnter(screen.getByRole("tab", { name: "Decisions 253" }));
    fireEvent.focus(screen.getByRole("tab", { name: "Watchtower 722" }));

    expect(onLensIntent.mock.calls).toEqual([["Decisions"], ["Watchtower"]]);
  });

  it("names a lens without a number until what waits in it has been counted", () => {
    renderHead({ counts: { Everything: 12 }, progress: null });

    expect(screen.getByRole("tab", { name: "Everything 12" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Watchtower" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Decisions" })).toBeInTheDocument();
    expect(screen.getByRole("progressbar", { name: "Reading what needs you" })).toHaveAttribute(
      "aria-valuenow",
      "0",
    );
  });

  it("draws the lens's filter beside the lenses and opens it on F", () => {
    const onOpenFilter = vi.fn();
    renderHead({ filter: <button type="button">Filter</button>, onOpenFilter });

    expect(screen.getByRole("button", { name: "Filter" })).toBeInTheDocument();
    press("f");
    press("F");

    expect(onOpenFilter).toHaveBeenCalledTimes(2);
  });

  it("offers whose cases to list only to someone who may see everyone's", () => {
    const onWhose = vi.fn<(whose: "mine" | "everyone") => void>();
    renderHead({ whose: "mine", onWhose });

    const group = screen.getByRole("group", { name: "Whose cases" });
    expect(group).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Mine" })).toHaveAttribute("aria-pressed", "true");
    fireEvent.click(screen.getByRole("button", { name: "Everyone" }));
    expect(onWhose.mock.calls).toEqual([["everyone"]]);
  });

  it("draws no choice of whose cases for a reader who only sees their own", () => {
    renderHead({ whose: null });

    expect(screen.queryByRole("group", { name: "Whose cases" })).toBeNull();
  });
});
