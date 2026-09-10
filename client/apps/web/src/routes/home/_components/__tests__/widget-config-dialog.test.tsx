import type { HomeWidget, HomeWidgetOption } from "@/lib/graphql/home-layout";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { WidgetConfigDialog } from "../widget-config-dialog";

const mocks = vi.hoisted(() => ({
  useReportCatalog: vi.fn(),
  useReportDashboards: vi.fn(),
  useTileReport: vi.fn(),
  buildCatalogIndex: vi.fn(),
  outputColumnChoices: vi.fn(),
  onSourceChange: vi.fn(),
}));

vi.mock("@/hooks/use-reports", () => ({
  useReportCatalog: mocks.useReportCatalog,
  useReportDashboards: mocks.useReportDashboards,
}));

vi.mock("@/routes/reports/dashboards/_components/use-tile-report", () => ({
  useTileReport: mocks.useTileReport,
}));

vi.mock("@/routes/reports/builder/_components/builder-state", () => ({
  buildCatalogIndex: mocks.buildCatalogIndex,
  outputColumnChoices: mocks.outputColumnChoices,
}));

// The picker has its own tests; here it only has to report a chosen source.
vi.mock("@/components/reports/report-source-picker", () => ({
  ReportSourcePicker: ({
    value,
    onChange,
  }: {
    value: { definitionId: string | null; cannedKey: string | null };
    onChange: (next: { definitionId: string | null; cannedKey: string | null }) => void;
  }) => (
    <button
      type="button"
      data-chosen={value.definitionId ?? value.cannedKey ?? ""}
      onClick={() => onChange({ definitionId: "rdef_1", cannedKey: null })}
    >
      Choose a report
    </button>
  ),
}));

function widget(overrides: Partial<HomeWidget> = {}): HomeWidget {
  return {
    id: "widget_1",
    key: "kpi",
    title: null,
    w: 3,
    h: 2,
    config: {
      metric: null,
      metrics: null,
      definitionId: null,
      cannedKey: null,
      chartId: null,
      columnId: null,
      dashboardId: null,
      text: null,
      limit: null,
      windowDays: null,
    },
    ...overrides,
  } as HomeWidget;
}

function option(configKind: string, label = "Widget"): HomeWidgetOption {
  return {
    key: "kpi",
    label,
    description: "",
    category: "pulse",
    configKind,
    analyticsInclude: null,
    defaultW: 3,
    defaultH: 2,
    minW: 2,
    minH: 2,
    maxW: 6,
    maxH: 4,
  } as unknown as HomeWidgetOption;
}

const METRICS = [
  { key: "activeShipments", label: "Active Shipments" },
  { key: "revenueToday", label: "Revenue Today" },
];

function renderDialog(
  configKind: string,
  overrides: { widget?: HomeWidget; metrics?: typeof METRICS } = {},
) {
  const onSave = vi.fn();
  const onCancel = vi.fn();

  render(
    <WidgetConfigDialog
      widget={overrides.widget ?? widget()}
      option={option(configKind)}
      metrics={overrides.metrics ?? METRICS}
      onSave={onSave}
      onCancel={onCancel}
    />,
  );

  return { onSave, onCancel };
}

function saveButton() {
  return screen.getByRole("button", { name: "Save" });
}

beforeEach(() => {
  vi.clearAllMocks();
  mocks.useReportCatalog.mockReturnValue({ data: { entities: [] } });
  mocks.useReportDashboards.mockReturnValue({ data: [], isLoading: false });
  mocks.buildCatalogIndex.mockReturnValue({ entities: new Map() });
  mocks.outputColumnChoices.mockReturnValue([]);
  mocks.useTileReport.mockReturnValue({ ir: null, name: "", loading: false });
});

describe("WidgetConfigDialog", () => {
  // The server rejects a metric widget with no metric. Letting the dialog close
  // on an unsavable draft turns that into a toast about a form nobody can see.
  it("will not save a metric widget until a metric is chosen", async () => {
    const user = userEvent.setup();
    const { onSave } = renderDialog("metric");

    expect(saveButton()).toBeDisabled();
    expect(screen.getByText("Choose the metric this tile shows.")).toBeInTheDocument();

    await user.click(screen.getByRole("checkbox", { name: "Revenue Today" }));

    expect(saveButton()).toBeEnabled();
    await user.click(saveButton());
    expect(onSave.mock.calls[0][0].config).toMatchObject({ metric: "revenueToday" });
  });

  it("swaps rather than refuses when a second metric is picked for a single-metric tile", async () => {
    const user = userEvent.setup();
    const { onSave } = renderDialog("metric");

    await user.click(screen.getByRole("checkbox", { name: "Active Shipments" }));
    await user.click(screen.getByRole("checkbox", { name: "Revenue Today" }));
    await user.click(saveButton());

    expect(onSave.mock.calls[0][0].config.metric).toBe("revenueToday");
  });

  it("will not save an announcement that is only whitespace", async () => {
    const user = userEvent.setup();
    renderDialog("text");

    expect(saveButton()).toBeDisabled();
    await user.type(screen.getByLabelText("Announcement"), "   ");
    expect(saveButton()).toBeDisabled();

    await user.type(screen.getByLabelText("Announcement"), "Yard closes at 4pm Friday.");
    expect(saveButton()).toBeEnabled();
  });

  it("will not save a report widget until a report is chosen", async () => {
    const user = userEvent.setup();
    renderDialog("report");

    expect(saveButton()).toBeDisabled();
    expect(screen.getByText("Choose the report this tile shows.")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Choose a report" }));
    expect(saveButton()).toBeEnabled();
  });

  // chartId and columnId have always been part of the stored config, but Home
  // never offered a way to set them, so a report with charts could only land as
  // a table.
  it("offers chart and single-number only once the report actually has them", async () => {
    const user = userEvent.setup();
    renderDialog("report");
    await user.click(screen.getByRole("button", { name: "Choose a report" }));

    expect(screen.getByRole("radio", { name: "Chart" })).toBeDisabled();
    expect(screen.getByRole("radio", { name: "Single number" })).toBeDisabled();
    expect(screen.getByRole("radio", { name: "Table" })).toHaveAttribute("aria-checked", "true");
  });

  it("saves the chart the report defines when the tile is drawn as a chart", async () => {
    mocks.useTileReport.mockReturnValue({
      ir: { charts: [{ id: "chart_1", type: "bar", title: "Revenue by Lane" }] },
      name: "Lane Revenue",
      loading: false,
    });

    const user = userEvent.setup();
    const { onSave } = renderDialog("report");
    await user.click(screen.getByRole("button", { name: "Choose a report" }));
    await user.click(screen.getByRole("radio", { name: "Chart" }));
    await user.click(saveButton());

    expect(onSave.mock.calls[0][0].config).toMatchObject({
      definitionId: "rdef_1",
      chartId: "chart_1",
      columnId: null,
    });
  });

  it("saves a measure, not a chart, when the tile is drawn as a single number", async () => {
    mocks.useTileReport.mockReturnValue({
      ir: { charts: [{ id: "chart_1", type: "bar", title: "Revenue by Lane" }] },
      name: "Lane Revenue",
      loading: false,
    });
    mocks.outputColumnChoices.mockReturnValue([
      { id: "lane", label: "Lane", isDim: true },
      { id: "revenue", label: "Revenue", isDim: false },
    ]);

    const user = userEvent.setup();
    const { onSave } = renderDialog("report");
    await user.click(screen.getByRole("button", { name: "Choose a report" }));
    await user.click(screen.getByRole("radio", { name: "Single number" }));
    await user.click(saveButton());

    expect(onSave.mock.calls[0][0].config).toMatchObject({
      columnId: "revenue",
      chartId: null,
    });
  });

  // A chart id means nothing outside the report that defined it.
  it("drops the chosen chart when the report is swapped", async () => {
    mocks.useTileReport.mockReturnValue({
      ir: { charts: [{ id: "chart_1", type: "bar", title: "Revenue by Lane" }] },
      name: "Lane Revenue",
      loading: false,
    });

    const user = userEvent.setup();
    const { onSave } = renderDialog("report", {
      widget: widget({
        config: { ...widget().config, cannedKey: "on_time", chartId: "chart_from_elsewhere" },
      }),
    });

    await user.click(screen.getByRole("button", { name: "Choose a report" }));
    await user.click(saveButton());

    expect(onSave.mock.calls[0][0].config).toMatchObject({
      definitionId: "rdef_1",
      cannedKey: null,
      chartId: null,
    });
  });

  it("leaves a widget with nothing to configure saveable straight away", () => {
    const { onSave } = renderDialog("none");
    expect(saveButton()).toBeEnabled();
    expect(onSave).not.toHaveBeenCalled();
    expect(screen.getByText(/This widget draws itself/)).toBeInTheDocument();
  });
});
