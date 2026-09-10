import type { ReportDashboardTile } from "@/types/report";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { TileEditorDialog } from "../tile-editor-dialog";

/**
 * The tile's report was chosen from a <Select> fed by a fixed first hundred
 * definitions with no search, so a dashboard could not point at a report past
 * that hundred at all. It now uses the same paged, searched picker the home
 * canvas does.
 */

const mocks = vi.hoisted(() => ({
  useReportCatalog: vi.fn(),
  useTileReport: vi.fn(),
  buildCatalogIndex: vi.fn(),
  outputColumnChoices: vi.fn(),
  pickerProps: vi.fn(),
}));

vi.mock("@/hooks/use-reports", () => ({
  useReportCatalog: mocks.useReportCatalog,
}));

vi.mock("../use-tile-report", () => ({
  useTileReport: mocks.useTileReport,
}));

vi.mock("../../../builder/_components/builder-state", () => ({
  buildCatalogIndex: mocks.buildCatalogIndex,
  outputColumnChoices: mocks.outputColumnChoices,
}));

vi.mock("@/components/reports/report-source-picker", () => ({
  ReportSourcePicker: (props: {
    value: { definitionId: string | null; cannedKey: string | null };
    onChange: (next: { definitionId: string | null; cannedKey: string | null }) => void;
  }) => {
    mocks.pickerProps(props.value);
    return (
      <button
        type="button"
        onClick={() => props.onChange({ definitionId: "rdef_9000", cannedKey: null })}
      >
        Pick a far-off report
      </button>
    );
  },
}));

function tile(overrides: Partial<ReportDashboardTile> = {}): ReportDashboardTile {
  return {
    id: "tile_1",
    kind: "table",
    x: 0,
    y: 0,
    w: 6,
    h: 5,
    ...overrides,
  } as ReportDashboardTile;
}

function renderEditor(target: ReportDashboardTile = tile()) {
  const onSave = vi.fn();
  render(
    <TileEditorDialog
      open
      onOpenChange={vi.fn()}
      tile={target}
      dashboardParams={[]}
      onSave={onSave}
    />,
  );
  return { onSave };
}

beforeEach(() => {
  vi.clearAllMocks();
  mocks.useReportCatalog.mockReturnValue({ data: { entities: [] } });
  mocks.buildCatalogIndex.mockReturnValue({ entities: new Map() });
  mocks.outputColumnChoices.mockReturnValue([]);
  mocks.useTileReport.mockReturnValue({ ir: null, name: "", loading: false });
});

describe("TileEditorDialog", () => {
  it("chooses the report through the paged picker", async () => {
    const user = userEvent.setup();
    const { onSave } = renderEditor();

    await user.click(screen.getByRole("button", { name: "Pick a far-off report" }));
    await user.click(screen.getByRole("button", { name: "Save Tile" }));

    expect(onSave.mock.calls[0][0]).toMatchObject({
      definitionId: "rdef_9000",
      cannedKey: undefined,
    });
  });

  it("shows the picker what the tile already reads from", () => {
    renderEditor(tile({ cannedKey: "on_time" }));

    expect(mocks.pickerProps).toHaveBeenCalledWith({
      definitionId: null,
      cannedKey: "on_time",
    });
  });

  // A chart or measure id means nothing outside the report that defined it.
  it("drops the chart and measure carried over from the previous report", async () => {
    const user = userEvent.setup();
    const { onSave } = renderEditor(
      tile({ kind: "table", cannedKey: "on_time", chartId: "chart_a", columnId: "revenue" }),
    );

    await user.click(screen.getByRole("button", { name: "Pick a far-off report" }));
    await user.click(screen.getByRole("button", { name: "Save Tile" }));

    expect(onSave.mock.calls[0][0]).toMatchObject({
      chartId: undefined,
      columnId: undefined,
    });
  });

  it("will not save a report tile with no report behind it", () => {
    renderEditor();

    expect(screen.getByRole("button", { name: "Save Tile" })).toBeDisabled();
  });

  it("needs no report for a text tile", () => {
    renderEditor(tile({ kind: "text", text: "Yard closes at 4pm" }));

    expect(screen.queryByRole("button", { name: "Pick a far-off report" })).toBeNull();
    expect(screen.getByRole("button", { name: "Save Tile" })).toBeEnabled();
  });
});
