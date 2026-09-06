import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { FormulaTemplate } from "@trenova/shared/types/formula-template";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ExportTemplateDialog } from "../export-template-dialog";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  listVersions: vi.fn(),
  listTestCases: vi.fn(),
  buildTemplateExport: vi.fn(),
  getExportFilename: vi.fn(),
  downloadJson: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }));

vi.mock("@/services/api", () => ({
  apiService: {
    formulaTemplateService: {
      get: mocks.get,
      listVersions: mocks.listVersions,
      listTestCases: mocks.listTestCases,
    },
  },
}));

vi.mock("@/lib/formula-template-export", () => ({
  buildTemplateExport: mocks.buildTemplateExport,
  getExportFilename: mocks.getExportFilename,
  downloadJson: mocks.downloadJson,
}));

// The list row is a projection: it carries neither `metadata` nor the typed
// variable definitions, so an export built from it would silently drop them.
const listRow = { id: "ft_01", name: "Per Mile" };

const fullTemplate = {
  id: "ft_01",
  name: "Per Mile",
  description: "Rates by loaded mile",
  type: "FreightCharge",
  expression: "distance * rate",
  status: "Active",
  schemaId: "shipment",
  variableDefinitions: [
    { name: "rate", type: "Number", description: "", required: true, defaultValue: 1 },
  ],
  breakdownDefinitions: [],
  minCharge: "0",
  maxCharge: "0",
  roundingMode: "HalfUp",
  roundingPrecision: 2,
  metadata: { category: "standard" },
} as unknown as FormulaTemplate;

describe("ExportTemplateDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.get.mockResolvedValue(fullTemplate);
    mocks.listTestCases.mockResolvedValue([]);
    mocks.listVersions.mockResolvedValue({ results: [] });
    mocks.buildTemplateExport.mockReturnValue({ exportVersion: "1.3" });
    mocks.getExportFilename.mockReturnValue("per-mile.formula-template.json");
  });

  it("exports the full template fetched by id, not the list row", async () => {
    render(<ExportTemplateDialog open onOpenChange={vi.fn()} template={listRow} />);

    fireEvent.click(screen.getByRole("button", { name: "Export" }));

    await waitFor(() => expect(mocks.downloadJson).toHaveBeenCalledTimes(1));
    expect(mocks.get).toHaveBeenCalledWith("ft_01");
    expect(mocks.buildTemplateExport).toHaveBeenCalledWith(fullTemplate, {
      versions: undefined,
      testCases: [],
    });
    expect(mocks.getExportFilename).toHaveBeenCalledWith(fullTemplate, false);
    expect(mocks.listVersions).not.toHaveBeenCalled();
  });

  it("includes version history when asked", async () => {
    const versions = [{ versionNumber: 1 }];
    mocks.listVersions.mockResolvedValue({ results: versions });
    const user = userEvent.setup();
    render(<ExportTemplateDialog open onOpenChange={vi.fn()} template={listRow} />);

    await user.click(screen.getByRole("checkbox"));
    fireEvent.click(screen.getByRole("button", { name: "Export" }));

    await waitFor(() => expect(mocks.downloadJson).toHaveBeenCalledTimes(1));
    expect(mocks.listVersions).toHaveBeenCalledWith("ft_01", { limit: 1000 });
    expect(mocks.buildTemplateExport).toHaveBeenCalledWith(fullTemplate, {
      versions,
      testCases: [],
    });
    expect(mocks.getExportFilename).toHaveBeenCalledWith(fullTemplate, true);
  });
});
