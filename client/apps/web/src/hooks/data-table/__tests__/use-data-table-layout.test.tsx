import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { useTable } from "@tanstack/react-table";
import { dataTableFeatures, selectDataTableViewState } from "@trenova/shared/lib/table-features";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import type { TableDensity, TableFormatRule } from "@/types/table-configuration";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useDataTableLayout } from "../use-data-table-layout";

const { saveMock, resetMock, requestMock } = vi.hoisted(() => ({
  saveMock: vi.fn(),
  resetMock: vi.fn(),
  requestMock: vi.fn(),
}));

vi.mock("@/lib/graphql/table-layout", () => ({
  saveMyTableLayout: saveMock,
  resetMyTableLayout: resetMock,
}));

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: requestMock,
}));

vi.mock("sonner", () => ({ toast: { error: vi.fn() } }));

type Row = { id: string; name: string; status: string };

const columns: ColumnDef<Row>[] = [
  { id: "name", accessorKey: "name" },
  { id: "status", accessorKey: "status" },
];
const NO_ROWS: Row[] = [];
const NO_RULES: TableFormatRule[] = [];

type Props = {
  density: TableDensity;
  formatRules: TableFormatRule[];
  activeViewId: string | null;
  pinnedRowsCollapsed?: boolean;
  hideChangesSinceLastVisit?: boolean;
  hideTotals?: boolean;
  virtualized?: boolean;
};

function setup(savedLayout: unknown) {
  requestMock.mockResolvedValue({
    myTableLayout: savedLayout
      ? { resource: "shipment", layout: savedLayout, version: 1, updatedAt: 1 }
      : null,
  });
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );

  return renderHook(
    ({
      density,
      formatRules,
      activeViewId,
      pinnedRowsCollapsed = false,
      hideChangesSinceLastVisit = false,
      hideTotals = false,
      virtualized = false,
    }: Props) => {
      const table = useTable(
        {
          features: dataTableFeatures,
          data: NO_ROWS,
          columns,
          getRowId: (row) => row.id,
        },
        selectDataTableViewState,
      );
      const layout = useDataTableLayout({
        resource: "shipment",
        table,
        density,
        formatRules,
        activeViewId,
        pinnedRowsCollapsed,
        hideChangesSinceLastVisit,
        hideTotals,
        virtualized,
      });
      return { table, layout };
    },
    {
      wrapper,
      initialProps: { density: "comfortable", formatRules: NO_RULES, activeViewId: null },
    },
  );
}

async function settle(hook: ReturnType<typeof setup>) {
  await waitFor(() => expect(hook.result.current.layout.settled).toBe(true));
}

async function arm(hook: ReturnType<typeof setup>) {
  act(() => hook.result.current.layout.arm());
  await act(async () => {
    await vi.advanceTimersByTimeAsync(0);
  });
}

describe("useDataTableLayout", () => {
  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    saveMock.mockImplementation(async (resource: string, layout: unknown) => ({
      resource,
      layout,
      version: 2,
      updatedAt: 2,
    }));
    resetMock.mockResolvedValue(true);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.clearAllMocks();
  });

  it("reads back the saved layout once it has loaded", async () => {
    const hook = setup({ columnOrder: ["status", "name"], density: "compact" });
    await settle(hook);

    expect(hook.result.current.layout.hasSavedLayout).toBe(true);
    expect(hook.result.current.layout.readSaved()).toMatchObject({
      columnOrder: ["status", "name"],
      density: "compact",
      columnVisibility: {},
      formatRules: [],
      activeViewId: null,
    });
  });

  it("says there is no saved layout when the person never changed the table", async () => {
    const hook = setup(null);
    await settle(hook);

    expect(hook.result.current.layout.hasSavedLayout).toBe(false);
    expect(hook.result.current.layout.readSaved()).toBeNull();
  });

  it("saves nothing while the table is being restored, before it is armed", async () => {
    const hook = setup(null);
    await settle(hook);

    act(() => hook.result.current.table.setColumnOrder(["status", "name"]));
    await act(async () => {
      await vi.advanceTimersByTimeAsync(5000);
    });

    expect(saveMock).not.toHaveBeenCalled();
  });

  it("saves the arrangement once, a second after the last change", async () => {
    const hook = setup(null);
    await settle(hook);
    await arm(hook);

    act(() => hook.result.current.table.setColumnOrder(["status", "name"]));
    await act(async () => {
      await vi.advanceTimersByTimeAsync(500);
    });
    act(() => hook.result.current.table.setColumnVisibility({ name: false }));
    await act(async () => {
      await vi.advanceTimersByTimeAsync(999);
    });
    expect(saveMock).not.toHaveBeenCalled();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1);
    });

    expect(saveMock).toHaveBeenCalledTimes(1);
    expect(saveMock).toHaveBeenCalledWith(
      "shipment",
      expect.objectContaining({
        columnOrder: ["status", "name"],
        columnVisibility: { name: false },
        density: "comfortable",
      }),
    );
  });

  it("does not save when a change is undone before the table settles", async () => {
    const hook = setup(null);
    await settle(hook);
    await arm(hook);

    act(() => hook.result.current.table.setColumnVisibility({ name: false }));
    act(() => hook.result.current.table.setColumnVisibility({}));
    await act(async () => {
      await vi.advanceTimersByTimeAsync(2000);
    });

    expect(saveMock).not.toHaveBeenCalled();
  });

  it("saves a density change made outside the table's own state", async () => {
    const hook = setup(null);
    await settle(hook);
    await arm(hook);

    hook.rerender({ density: "compact", formatRules: NO_RULES, activeViewId: null });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000);
    });

    expect(saveMock).toHaveBeenCalledWith(
      "shipment",
      expect.objectContaining({ density: "compact" }),
    );
  });

  it("keeps widths inside what the server accepts", async () => {
    const hook = setup(null);
    await settle(hook);
    await arm(hook);

    act(() => hook.result.current.table.setColumnSizing({ name: 3, status: 9000 }));
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000);
    });

    expect(saveMock).toHaveBeenCalledWith(
      "shipment",
      expect.objectContaining({ columnSizing: { name: 24, status: 2000 } }),
    );
  });

  it("drops a pending save on reset, restores the defaults, and does not save them back", async () => {
    const hook = setup({ columnOrder: ["status", "name"] });
    await settle(hook);
    await arm(hook);

    act(() => hook.result.current.table.setColumnVisibility({ name: false }));
    const restoreDefaults = vi.fn(() => {
      hook.result.current.table.setColumnVisibility({});
      hook.result.current.table.setColumnOrder([]);
    });
    await act(async () => {
      await hook.result.current.layout.reset(restoreDefaults);
      await vi.advanceTimersByTimeAsync(2000);
    });

    expect(resetMock).toHaveBeenCalledWith("shipment");
    expect(restoreDefaults).toHaveBeenCalledTimes(1);
    expect(saveMock).not.toHaveBeenCalled();
    expect(hook.result.current.layout.hasSavedLayout).toBe(false);
  });

  it("keeps the rows a person pinned and whether their section is folded", async () => {
    const hook = setup(null);
    await settle(hook);
    await arm(hook);

    act(() => hook.result.current.table.setRowPinning({ top: ["b", "a"], bottom: [] }));
    hook.rerender({
      density: "comfortable",
      formatRules: NO_RULES,
      activeViewId: null,
      pinnedRowsCollapsed: true,
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000);
    });

    expect(saveMock).toHaveBeenCalledWith(
      "shipment",
      expect.objectContaining({ pinnedRowIds: ["b", "a"], pinnedRowsCollapsed: true }),
    );
  });

  it("keeps whether the person draws only the rows in view, and reads it back", async () => {
    const hook = setup(null);
    await settle(hook);
    await arm(hook);

    hook.rerender({
      density: "comfortable",
      formatRules: NO_RULES,
      activeViewId: null,
      virtualized: true,
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000);
    });

    expect(saveMock).toHaveBeenCalledWith(
      "shipment",
      expect.objectContaining({ virtualized: true }),
    );

    const restored = setup({ virtualized: true });
    await settle(restored);
    expect(restored.result.current.layout.readSaved()).toMatchObject({ virtualized: true });
  });

  it("reads a layout saved before the choice existed as drawing every row", async () => {
    const hook = setup({ density: "compact" });
    await settle(hook);

    expect(hook.result.current.layout.readSaved()).toMatchObject({ virtualized: false });
  });

  it("reads back pinned rows saved earlier", async () => {
    const hook = setup({ pinnedRowIds: ["shp_1"], pinnedRowsCollapsed: true });
    await settle(hook);

    expect(hook.result.current.layout.readSaved()).toMatchObject({
      pinnedRowIds: ["shp_1"],
      pinnedRowsCollapsed: true,
    });
  });

  it("keeps the last visit as it was while the table is open", async () => {
    const hook = setup({ lastSeenAt: 1000 });
    await settle(hook);
    hook.result.current.layout.readSaved();
    await arm(hook);

    act(() => hook.result.current.table.setColumnOrder(["status", "name"]));
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000);
    });

    expect(saveMock).toHaveBeenCalledWith("shipment", expect.objectContaining({ lastSeenAt: 1000 }));
  });

  it("moves the last visit to the moment the person leaves the table", async () => {
    vi.setSystemTime(new Date(5_000_000));
    const hook = setup({ lastSeenAt: 1000 });
    await settle(hook);
    hook.result.current.layout.readSaved();
    await arm(hook);

    hook.unmount();
    // The last save queues behind any still in flight, so it starts a tick later.
    await act(async () => {
      await Promise.resolve();
    });

    expect(saveMock).toHaveBeenCalledWith("shipment", expect.objectContaining({ lastSeenAt: 5000 }));
  });

  it("sends a save that was waiting when the table goes away", async () => {
    const hook = setup(null);
    await settle(hook);
    await arm(hook);

    act(() => hook.result.current.table.setColumnOrder(["status", "name"]));
    hook.unmount();
    await act(async () => {
      await Promise.resolve();
    });

    expect(saveMock).toHaveBeenCalledTimes(1);
  });
});
