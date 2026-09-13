import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import type { TableConfig } from "@/types/table-configuration";
import type { Shipment } from "@trenova/shared/types/shipment";
import type { User } from "@trenova/shared/types/user";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { CommandCenterTable } from "../command-center-table";

const page = vi.hoisted(() => ({
  current: {
    results: [] as unknown[],
    count: 0,
    pageInfo: { endCursor: null as string | null, hasNextPage: false },
  },
}));

vi.mock("@/lib/graphql/shipment", () => ({
  listShipmentsGraphQL: async () => page.current,
}));

vi.mock("../use-view-counts", () => ({
  useSavedViewCounts: () => ({}),
}));

vi.mock("../expanded-row", () => ({
  ExpandedRow: () => <div data-testid="expanded-row-panel" />,
}));

const savedViews = vi.hoisted(() => ({
  defaultConfig: null as { tableConfig: Partial<TableConfig> } | null,
  lastSavedConfig: null as TableConfig | null,
}));

vi.mock("@/lib/queries", () => ({
  queries: {
    tableConfiguration: {
      default: () => ({
        queryKey: ["table-config-default"],
        queryFn: async () => savedViews.defaultConfig,
      }),
    },
  },
}));

vi.mock("@/components/data-table/data-table-save-config-dialog", () => ({
  DataTableSaveConfigDialog: ({ currentConfig }: { currentConfig: TableConfig }) => {
    savedViews.lastSavedConfig = currentConfig;
    return null;
  },
}));

function signIn() {
  useAuthStore.setState({
    user: {
      currentOrganizationId: "org_01",
      memberships: [
        {
          userId: "usr_01",
          organizationId: "org_01",
          isDefault: true,
          organization: {
            id: "org_01",
            name: "Hybrid Co",
            brokerageEnabled: true,
            assetOperationsEnabled: true,
          },
        },
      ],
    } as unknown as User,
    isAuthenticated: true,
  });
}

function column(
  id: string,
  size: number,
  bounds: { minSize: number; maxSize: number } = { minSize: 40, maxSize: 600 },
): ColumnDef<Shipment> {
  return {
    id,
    header: id,
    accessorFn: () => null,
    cell: ({ row }: { row: { original: Shipment } }) => `${id}:${row.original.proNumber}`,
    size,
    ...bounds,
  } as unknown as ColumnDef<Shipment>;
}

const columns: ColumnDef<Shipment>[] = [
  column("status", 160),
  column("actions", 56, { minSize: 56, maxSize: 56 }),
  column("customer", 220),
  column("lane", 280),
  column("proBol", 160),
];

function renderTable(searchParams = "?mode=table") {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <NuqsTestingAdapter searchParams={searchParams}>
      <QueryClientProvider client={queryClient}>
        <CommandCenterTable
          columns={columns}
          rowActions={[]}
          mandatoryFieldFilters={[]}
          onUploadDocument={vi.fn()}
        />
      </QueryClientProvider>
    </NuqsTestingAdapter>,
  );
}

function withRows(rows: Partial<Shipment>[]) {
  page.current = {
    results: rows,
    count: rows.length,
    pageInfo: { endCursor: null, hasNextPage: false },
  };
}

function styleOf(element: Element) {
  return element.getAttribute("style") ?? "";
}

describe("command center table horizontal scroll", () => {
  beforeEach(() => {
    signIn();
    withRows([
      { id: "shp_01", proNumber: "S-100" },
      { id: "shp_02", proNumber: "S-200" },
    ]);
  });

  afterEach(() => {
    cleanup();
    useAuthStore.setState({ user: null, isAuthenticated: false });
    savedViews.defaultConfig = null;
    savedViews.lastSavedConfig = null;
  });

  it("lays the table out at its columns' declared widths instead of squeezing it into the container", async () => {
    renderTable();
    await screen.findByText("lane:S-100");

    const table = screen.getByTestId("command-center-table");
    expect(table).toHaveClass("table-fixed");
    expect(table.style.width).toBe("876px");
    expect(table.style.minWidth).toBe("100%");
  });

  it("keeps the lane at the start and the row actions at the end regardless of declared order", async () => {
    renderTable();
    await screen.findByText("lane:S-100");

    const table = screen.getByTestId("command-center-table");
    const headers = within(table)
      .getAllByRole("columnheader")
      .map((th) => th.textContent);
    expect(headers).toEqual(["lane", "status", "customer", "proBol", "actions"]);

    const cols = table.querySelectorAll("col");
    expect(Array.from(cols, (col) => col.style.width)).toEqual([
      "280px",
      "160px",
      "220px",
      "160px",
      "56px",
    ]);

    const [firstCell, ...rest] = within(screen.getByText("lane:S-100").closest("tr")!).getAllByRole(
      "cell",
    );
    const lastCell = rest.at(-1)!;
    expect(firstCell).toHaveTextContent("lane:S-100");
    expect(firstCell).toHaveClass("sticky");
    expect(styleOf(firstCell)).toContain("inset-inline-start: var(--col-lane-start)");
    expect(lastCell).toHaveTextContent("actions:S-100");
    expect(lastCell).toHaveClass("sticky");
    expect(styleOf(lastCell)).toContain("inset-inline-end: var(--col-actions-end)");
    for (const cell of rest.slice(0, -1)) {
      expect(cell).not.toHaveClass("sticky");
    }

    expect(table.style.getPropertyValue("--col-lane-start")).toBe("0px");
    expect(table.style.getPropertyValue("--col-actions-end")).toBe("0px");
  });

  it("gives pinned cells an opaque surface so scrolled columns never show through", async () => {
    renderTable();
    await screen.findByText("lane:S-100");

    const laneCell = screen.getByText("lane:S-100").closest("td")!;
    expect(laneCell).toHaveClass("bg-card");
    expect(laneCell.className).not.toMatch(/(^|\s)bg-background(\s|$)/);
  });

  it("draws the expanded row's outline on its pinned cells, which cover the row outline", async () => {
    renderTable("?mode=table&expanded=shp_01");
    await screen.findByTestId("expanded-row-panel");

    const expandedCells = within(screen.getByText("lane:S-100").closest("tr")!).getAllByRole(
      "cell",
    );
    const lane = expandedCells[0];
    const actions = expandedCells.at(-1)!;
    expect(styleOf(lane)).toContain("inset 1px 0 0 0 var(--brand)");
    expect(styleOf(lane)).toContain("inset 0 1px 0 0 var(--brand)");
    expect(styleOf(actions)).toContain("inset -1px 0 0 0 var(--brand)");
    expect(styleOf(actions)).toContain("inset 0 -1px 0 0 var(--brand)");

    const collapsedLane = screen.getByText("lane:S-200").closest("td")!;
    expect(styleOf(collapsedLane)).not.toContain("var(--brand)");
  });

  it("pins the expanded panel to the visible viewport rather than the full table width", async () => {
    renderTable("?mode=table&expanded=shp_01");

    const panel = await screen.findByTestId("expanded-row-panel");
    const stickyFrame = panel.closest("td")!.firstElementChild!;
    expect(stickyFrame).toHaveClass("sticky", "left-0", "w-[100cqw]");
    expect(screen.getByTestId("command-center-table").closest(".\\@container")).not.toBeNull();
  });

  it("pins the empty state message to the visible viewport", async () => {
    withRows([]);
    renderTable();

    const message = await screen.findByText("No shipments match the current view.");
    expect(message).toHaveClass("sticky", "left-0", "w-[100cqw]");
  });

  describe("column resizing", () => {
    function colWidths() {
      return Array.from(
        screen.getByTestId("command-center-table").querySelectorAll("col"),
        (col) => col.style.width,
      );
    }

    function drag(handle: HTMLElement, from: number, to: number) {
      fireEvent.mouseDown(handle, { clientX: from });
      fireEvent.mouseMove(document, { clientX: to });
      fireEvent.mouseUp(document, { clientX: to });
    }

    function gripOf(handle: HTMLElement) {
      return handle.querySelector<HTMLElement>('[data-slot="column-resize-grip"]')!;
    }

    it("shows a visible grip on every resizable header without needing a hover", async () => {
      renderTable();
      await screen.findByText("lane:S-100");

      for (const id of ["lane", "status", "customer", "proBol"]) {
        const handle = screen.getByRole("separator", { name: `Resize ${id} column` });
        expect(handle).toHaveClass("cursor-col-resize");
        const grip = gripOf(handle);
        expect(grip).not.toBeNull();
        expect(grip).toHaveClass("bg-muted-foreground/40", "h-4", "w-px");
        expect(grip).not.toHaveClass("bg-border");
        expect(grip.className).not.toMatch(/(^|\s)(opacity-0|invisible|hidden)(\s|$)/);
      }
    });

    it("offers no handle on a column whose size cannot change", async () => {
      renderTable();
      await screen.findByText("lane:S-100");

      expect(screen.queryByRole("separator", { name: "Resize actions column" })).toBeNull();
    });

    it("highlights the grip for the column being dragged until the drag ends", async () => {
      renderTable();
      await screen.findByText("lane:S-100");

      const handle = screen.getByRole("separator", { name: "Resize customer column" });
      fireEvent.mouseDown(handle, { clientX: 500 });
      fireEvent.mouseMove(document, { clientX: 520 });

      await waitFor(() => expect(gripOf(handle)).toHaveClass("bg-primary", "h-full"));
      expect(handle).toHaveAttribute("data-resizing", "true");
      expect(
        gripOf(screen.getByRole("separator", { name: "Resize status column" })),
      ).not.toHaveClass("bg-primary");

      fireEvent.mouseUp(document, { clientX: 520 });

      await waitFor(() => expect(handle).not.toHaveAttribute("data-resizing"));
      expect(gripOf(handle)).not.toHaveClass("bg-primary");
    });

    it("widens a column and the table as the handle is dragged", async () => {
      renderTable();
      await screen.findByText("lane:S-100");

      drag(screen.getByRole("separator", { name: "Resize customer column" }), 500, 580);

      await waitFor(() =>
        expect(colWidths()).toEqual(["280px", "160px", "300px", "160px", "56px"]),
      );
      const table = screen.getByTestId("command-center-table");
      expect(table.style.width).toBe("956px");
      expect(screen.getByRole("columnheader", { name: /customer/ }).style.width).toBe("300px");
    });

    it("resizes a pinned column like any other", async () => {
      renderTable();
      await screen.findByText("lane:S-100");

      drag(screen.getByRole("separator", { name: "Resize lane column" }), 280, 240);

      await waitFor(() => expect(colWidths()[0]).toBe("240px"));
      expect(screen.getByTestId("command-center-table").style.width).toBe("836px");
    });

    it("never shrinks a column below its minimum size", async () => {
      renderTable();
      await screen.findByText("lane:S-100");

      drag(screen.getByRole("separator", { name: "Resize status column" }), 400, 0);

      await waitFor(() => expect(colWidths()[1]).toBe("40px"));
    });

    it("restores a column's declared width on double-click", async () => {
      renderTable();
      await screen.findByText("lane:S-100");

      const handle = screen.getByRole("separator", { name: "Resize customer column" });
      drag(handle, 500, 560);
      await waitFor(() => expect(colWidths()[2]).toBe("280px"));

      fireEvent.doubleClick(handle);
      await waitFor(() => expect(colWidths()[2]).toBe("220px"));
    });

    it("saves the resized widths into the view being saved", async () => {
      renderTable();
      await screen.findByText("lane:S-100");

      drag(screen.getByRole("separator", { name: "Resize customer column" }), 500, 550);

      await waitFor(() =>
        expect(savedViews.lastSavedConfig?.columnSizing).toEqual({ customer: 270 }),
      );
    });

    it("applies the widths stored on the default view", async () => {
      savedViews.defaultConfig = {
        tableConfig: { columnSizing: { lane: 320, proBol: 200 } },
      };
      renderTable();
      await screen.findByText("lane:S-100");

      await waitFor(() =>
        expect(colWidths()).toEqual(["320px", "160px", "220px", "200px", "56px"]),
      );
      expect(screen.getByTestId("command-center-table").style.width).toBe("956px");
    });
  });
});
