import { act, fireEvent, render, renderHook, screen } from "@testing-library/react";
import { withNuqsTestingAdapter, type UrlUpdateEvent } from "nuqs/adapters/testing";
import { describe, expect, it } from "vitest";
import { GroupMenu } from "../toolbar/board-controls";
import { useShipmentBoardUrl } from "../url-state";

function renderMenu(searchParams: string) {
  const updates: UrlUpdateEvent[] = [];
  render(<GroupMenu />, {
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

describe("GroupMenu", () => {
  it("names the current grouping and offers every grouping in a popover", async () => {
    renderMenu("?group=deliveryDate");

    const trigger = screen.getByRole("button", { name: "Group by" });
    expect(trigger.textContent).toContain("Delivery date");
    expect(trigger.getAttribute("data-grouped")).toBe("true");

    fireEvent.click(trigger);

    const options = await screen.findAllByRole("radio");
    expect(options.map((option) => option.textContent)).toEqual([
      "Status",
      "Ship date",
      "Delivery date",
      "Customer",
      "Owner",
      "No grouping",
    ]);
    expect(screen.getByRole("radio", { name: "Delivery date" }).getAttribute("aria-checked")).toBe(
      "true",
    );
  });

  it("switches the grouping and opens every group again", async () => {
    const updates = renderMenu("?group=stage&collapsed=2,5");

    fireEvent.click(screen.getByRole("button", { name: "Group by" }));
    fireEvent.click(await screen.findByRole("radio", { name: "Customer" }));
    await flushUrlUpdates();

    const last = updates.at(-1);
    expect(last?.searchParams.get("group")).toBe("customer");
    expect(last?.searchParams.has("collapsed")).toBe(false);
    expect(screen.queryByRole("radio")).toBeNull();
  });

  it("reads plainly when the board is not grouped, and hides off the table view", () => {
    renderMenu("?group=none");
    const trigger = screen.getByRole("button", { name: "Group by" });
    expect(trigger.textContent).toContain("Group");
    expect(trigger.getAttribute("data-grouped")).toBe("false");
  });

  it("is not offered on the timeline or map", () => {
    renderMenu("?view=map");
    expect(screen.queryByRole("button", { name: "Group by" })).toBeNull();
  });
});

describe("shipment board grouping in the URL", () => {
  function groupFrom(searchParams: string) {
    const { result } = renderHook(() => useShipmentBoardUrl(), {
      wrapper: withNuqsTestingAdapter({ searchParams }),
    });
    return result.current[0];
  }

  it("still opens links shared before there was more than one grouping", () => {
    expect(groupFrom("?group=true").group).toBe("stage");
    expect(groupFrom("?group=false").group).toBe("none");
  });

  it("falls back to status for a grouping it does not know, and keeps collapsed keys as text", () => {
    expect(groupFrom("?group=weather").group).toBe("stage");
    expect(groupFrom("").group).toBe("stage");
    expect(groupFrom("?group=owner&collapsed=,usr_1").collapsed).toEqual(["", "usr_1"]);
  });
});
