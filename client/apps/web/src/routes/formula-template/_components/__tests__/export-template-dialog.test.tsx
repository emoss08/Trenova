import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { FormulaTemplate } from "@trenova/shared/types/formula-template";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ExportTemplateDialog } from "../export-template-dialog";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  listVersions: vi.fn(),
  buildTemplateExport: vi.fn(),
  getExportFilename: vi.fn(),
  downloadJson: vi.fn(),
}));

vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: { error: vi.fn(), success: vi.fn() },
}));

vi.mock("@/services/api", () => ({
  apiService: {
    formulaTemplateService: {
      get: mocks.get,
      listVersions: mocks.listVersions,
    },
  },
}));

vi.mock("@/lib/formula-template-export", () => ({
  buildTemplateExport: mocks.buildTemplateExport,
  getExportFilename: mocks.getExportFilename,
  downloadJson: mocks.downloadJson,
}));

// The list row is a projection: it carries no source, so an export built from it would have
// nothing to import.
const listRow = { id: "ft_01", name: "Per mile" };

const fullTemplate = {
  id: "ft_01",
  name: "Per mile",
  description: "Rates by loaded mile",
  source: 'tariff "Per mile"\n\ninput rate: Money/mi\n\ncharge "Linehaul" (quantity: shipment.distance, rate: rate)\n',
  status: "Active",
  inputs: [],
  imports: [],
  testCount: 0,
} as unknown as FormulaTemplate;

describe("ExportTemplateDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.get.mockResolvedValue(fullTemplate);
    mocks.listVersions.mockResolvedValue({ results: [] });
    mocks.buildTemplateExport.mockReturnValue({ exportVersion: "2.0" });
    mocks.getExportFilename.mockReturnValue("per-mile.formula-template.json");
  });

  it("exports the full template fetched by id, not the list row", async () => {
    render(<ExportTemplateDialog open onOpenChange={vi.fn()} template={listRow} />);

    fireEvent.click(screen.getByRole("button", { name: "Export" }));

    await waitFor(() => expect(mocks.downloadJson).toHaveBeenCalledTimes(1));
    expect(mocks.get).toHaveBeenCalledWith("ft_01");
    expect(mocks.buildTemplateExport).toHaveBeenCalledWith(fullTemplate, { versions: undefined });
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
    expect(mocks.buildTemplateExport).toHaveBeenCalledWith(fullTemplate, { versions });
    expect(mocks.getExportFilename).toHaveBeenCalledWith(fullTemplate, true);
  });
});
