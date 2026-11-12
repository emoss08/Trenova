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
      progressLabel="4 of 10 cleared today"
      done={4}
      total={10}
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

  it("picks a lens by its chip and by the keys 1 to 3", () => {
    const onLens = renderHead();

    fireEvent.click(screen.getByRole("tab", { name: "Watchtower 722" }));
    press("3");
    press("1");

    expect(onLens.mock.calls).toEqual([["Watchtower"], ["Decisions"], ["Everything"]]);
  });

  it("shows only the lenses the person may open and keeps each lens on its own key", () => {
    const onLens = renderHead({ lens: "Decisions", lenses: ["Decisions"] });

    expect(screen.getAllByRole("tab")).toHaveLength(1);
    expect(screen.getByRole("tab", { name: "Decisions 253" })).toHaveAttribute(
      "title",
      "Press 3",
    );
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

  it("offers to clear the filters only while the list is filtered", () => {
    const onClearFilters = vi.fn();
    renderHead({ onClearFilters });

    fireEvent.click(screen.getByRole("button", { name: /Filtered/ }));

    expect(onClearFilters).toHaveBeenCalledTimes(1);
  });
});
