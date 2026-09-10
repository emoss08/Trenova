import type { HomeWidgetCatalog, HomeWidgetOption } from "@/lib/graphql/home-layout";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { AddWidgetDialog } from "../add-widget-dialog";

function option(overrides: Partial<HomeWidgetOption> & { key: string }): HomeWidgetOption {
  return {
    label: overrides.key,
    description: "",
    category: "work",
    configKind: "none",
    analyticsInclude: null,
    defaultW: 4,
    defaultH: 4,
    minW: 2,
    minH: 2,
    maxW: 12,
    maxH: 8,
    ...overrides,
  } as HomeWidgetOption;
}

const CATALOG = {
  gridColumns: 12,
  maxWidgets: 24,
  densities: ["comfortable", "compact"],
  metrics: [],
  categories: [
    { key: "work", label: "Work", description: "Queues that need someone to act" },
    { key: "pulse", label: "Pulse", description: "How the operation is running right now" },
  ],
  widgets: [
    option({
      key: "unassigned",
      label: "Unassigned Loads",
      description: "Active shipments with no usable assignment",
      category: "work",
      configKind: "queue",
    }),
    option({
      key: "billing-queue",
      label: "Billing Queue",
      description: "Delivered loads waiting to be billed",
      category: "work",
      configKind: "queue",
    }),
    option({
      key: "kpi",
      label: "Metric",
      description: "One number with its trend and target",
      category: "pulse",
      configKind: "metric",
    }),
  ],
} as unknown as HomeWidgetCatalog;

function renderGallery(
  overrides: {
    used?: number;
    max?: number;
    usedCounts?: Map<string, number>;
    loading?: boolean;
    catalog?: HomeWidgetCatalog | undefined;
  } = {},
) {
  const onAdd = vi.fn();
  const onOpenChange = vi.fn();

  render(
    <AddWidgetDialog
      open
      onOpenChange={onOpenChange}
      catalog={"catalog" in overrides ? overrides.catalog : CATALOG}
      loading={overrides.loading ?? false}
      usedCounts={overrides.usedCounts ?? new Map()}
      used={overrides.used ?? 0}
      max={overrides.max ?? 24}
      onAdd={onAdd}
    />,
  );

  return { onAdd, onOpenChange };
}

function cardNames(): string[] {
  return screen
    .getAllByRole("button")
    .filter((element) => element.hasAttribute("data-widget-card"))
    .map((element) => within(element).getAllByText(/.+/)[0]?.textContent ?? "");
}

describe("AddWidgetDialog", () => {
  it("groups the catalog under the categories the server publishes", () => {
    renderGallery();

    expect(screen.getByRole("heading", { name: "Work" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Pulse" })).toBeInTheDocument();
    expect(cardNames()).toEqual(["Unassigned Loads", "Billing Queue", "Metric"]);
  });

  it("searches the description, not just the name", async () => {
    const user = userEvent.setup();
    renderGallery();

    await user.type(screen.getByLabelText("Search widgets"), "billed");

    expect(cardNames()).toEqual(["Billing Queue"]);
    expect(screen.getByText("1 match")).toBeInTheDocument();
  });

  it("narrows to one category from the rail", async () => {
    const user = userEvent.setup();
    renderGallery();

    await user.click(screen.getByRole("button", { name: /^Pulse/ }));

    expect(cardNames()).toEqual(["Metric"]);
    expect(screen.queryByRole("heading", { name: "Work" })).toBeNull();
  });

  // A search that empties the open category otherwise reads as an empty
  // catalog; widening puts the matches back in front of the person.
  it("widens back to everything when the open category has no match", async () => {
    const user = userEvent.setup();
    renderGallery();

    await user.click(screen.getByRole("button", { name: /^Pulse/ }));
    await user.type(screen.getByLabelText("Search widgets"), "billed");

    expect(cardNames()).toEqual(["Billing Queue"]);
  });

  it("returns to the chosen category once the search is cleared", async () => {
    const user = userEvent.setup();
    renderGallery();

    await user.click(screen.getByRole("button", { name: /^Pulse/ }));
    await user.type(screen.getByLabelText("Search widgets"), "billed");
    await user.click(screen.getByRole("button", { name: "Clear search" }));

    expect(cardNames()).toEqual(["Metric"]);
  });

  it("hands the chosen widget back to the canvas", async () => {
    const user = userEvent.setup();
    const { onAdd } = renderGallery();

    await user.click(screen.getByRole("button", { name: /Billing Queue/ }));

    expect(onAdd).toHaveBeenCalledTimes(1);
    expect(onAdd.mock.calls[0][0]).toMatchObject({ key: "billing-queue" });
  });

  it("says how many of a widget are already on the canvas", () => {
    renderGallery({ usedCounts: new Map([["unassigned", 2]]) });

    expect(screen.getByText("2× on canvas")).toBeInTheDocument();
  });

  // The widget limit is the server's. Reaching it has to stop the click, not
  // just explain itself after the save comes back rejected.
  it("refuses to add anything once the home screen is full", async () => {
    const user = userEvent.setup();
    const { onAdd } = renderGallery({ used: 24, max: 24 });

    expect(screen.getByText(/This home screen is full/)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /Billing Queue/ }));
    expect(onAdd).not.toHaveBeenCalled();
  });

  it("says how much room is left while there is room", () => {
    renderGallery({ used: 20, max: 24 });

    expect(screen.getByText(/Room for 4 more/)).toBeInTheDocument();
  });

  it("offers a way out of a search that matches nothing", async () => {
    const user = userEvent.setup();
    renderGallery();

    await user.type(screen.getByLabelText("Search widgets"), "zzzz");

    expect(screen.getByText("Nothing matches")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Clear filters" }));
    expect(cardNames()).toEqual(["Unassigned Loads", "Billing Queue", "Metric"]);
  });

  it("draws placeholders rather than an empty gallery while the catalog loads", () => {
    renderGallery({ loading: true, catalog: undefined });

    expect(cardNames()).toEqual([]);
    expect(screen.queryByText("Nothing matches")).toBeNull();
  });
});
