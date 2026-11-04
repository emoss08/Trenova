import { act, fireEvent, render, screen } from "@testing-library/react";
import { withNuqsTestingAdapter, type UrlUpdateEvent } from "nuqs/adapters/testing";
import { describe, expect, it } from "vitest";
import { ViewSwitch } from "../toolbar/board-controls";

function renderSwitch(searchParams: string) {
  const updates: UrlUpdateEvent[] = [];
  render(<ViewSwitch />, {
    wrapper: withNuqsTestingAdapter({
      searchParams,
      onUrlUpdate: (event) => updates.push(event),
    }),
  });
  return updates;
}

async function flushUrlUpdates() {
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 100));
  });
}

describe("ViewSwitch", () => {
  it("offers every view and checks the one in the URL", () => {
    renderSwitch("?view=timeline");

    expect(screen.getByRole("radiogroup", { name: "View" })).toBeTruthy();
    expect(
      screen
        .getAllByRole("radio")
        .map((option) => [option.textContent, option.getAttribute("aria-checked")]),
    ).toEqual([
      ["Table", "false"],
      ["Timeline", "true"],
      ["Map", "false"],
    ]);
  });

  it("writes the picked view to the URL", async () => {
    const updates = renderSwitch("");

    fireEvent.click(screen.getByRole("radio", { name: "Map" }));
    await flushUrlUpdates();

    expect(updates.at(-1)?.searchParams.get("view")).toBe("map");
    expect(screen.getByRole("radio", { name: "Map" }).getAttribute("aria-checked")).toBe("true");
  });

  it("moves to the next view with an arrow key", async () => {
    const updates = renderSwitch("?view=timeline");

    fireEvent.keyDown(screen.getByRole("radio", { name: "Timeline" }), { key: "ArrowRight" });
    await flushUrlUpdates();

    expect(updates.at(-1)?.searchParams.get("view")).toBe("map");
  });
});
