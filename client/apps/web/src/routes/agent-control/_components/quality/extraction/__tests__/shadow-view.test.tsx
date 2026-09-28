import type {
  ExtractionShadowReport,
  ExtractionShadowResultDetail,
  ExtractionShadowSettings,
} from "@/lib/graphql/extraction-shadow";
import { stubLayout } from "@/test/layout";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ColumnDef, DataTablePanelProps } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { ComponentType, ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const permissions = vi.hoisted(() => ({ denied: new Set<string>() }));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (resource: string, operation: string) => ({
    allowed: !permissions.denied.has(`${resource}:${operation}`),
    isLoading: false,
  }),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

type StubTableProps = {
  name: string;
  columns: ColumnDef<Record<string, unknown>>[];
  TablePanel?: ComponentType<DataTablePanelProps<Record<string, unknown>>>;
};

const table = vi.hoisted(() => ({
  rows: [] as Record<string, unknown>[],
  openRow: null as Record<string, unknown> | null,
}));

vi.mock("@/components/data-table/data-table", () => ({
  DataTable: (props: StubTableProps) => {
    const { TablePanel } = props;
    return (
      <section aria-label={`${props.name} table`}>
        <table>
          <tbody>
            {table.rows.map((row, rowIndex) => (
              <tr key={rowIndex}>
                {props.columns.map((column, index) => (
                  <td key={index}>
                    {typeof column.cell === "function"
                      ? (column.cell as (context: unknown) => ReactNode)({ row: { original: row } })
                      : null}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
        {TablePanel && table.openRow ? (
          <TablePanel open onOpenChange={() => {}} mode="edit" row={table.openRow} />
        ) : null}
      </section>
    );
  },
}));

const fetchSettings = vi.fn<() => Promise<ExtractionShadowSettings>>();
const fetchReport = vi.fn<() => Promise<ExtractionShadowReport>>();
const fetchResult = vi.fn<() => Promise<ExtractionShadowResultDetail | null>>();
const updateSettings = vi.fn();

vi.mock("@/lib/graphql/extraction-shadow", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/graphql/extraction-shadow")>();
  return {
    ...actual,
    fetchExtractionShadowSettings: () => fetchSettings(),
    fetchExtractionShadowReport: () => fetchReport(),
    fetchExtractionShadowResult: () => fetchResult(),
    updateExtractionShadowSettings: (input: unknown) => updateSettings(input),
  };
});

const providers = [
  {
    id: "aip_candidate",
    name: "Fine-tuned Qwen",
    model: "trenova-extract",
    enabled: true,
    tasks: ["DocumentExtraction"],
  },
  {
    id: "aip_chat",
    name: "Chat only",
    model: "chat-large",
    enabled: true,
    tasks: ["AgentChat"],
  },
];

vi.mock("@/lib/graphql/ai-provider", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/graphql/ai-provider")>();
  return { ...actual, fetchAIProviders: () => Promise.resolve(providers) };
});

const { ShadowView } = await import("../shadow-view");

function settings(overrides: Partial<ExtractionShadowSettings> = {}): ExtractionShadowSettings {
  return {
    enabled: true,
    providerId: "aip_candidate",
    samplePercent: 10,
    dailyLimit: 200,
    updatedById: null,
    version: 3,
    updatedAt: 1_790_000_000,
    ...overrides,
  } as ExtractionShadowSettings;
}

function side(scored: number, correct: number) {
  return { scored, correct, corrected: scored - correct, missed: 0, accuracy: correct / scored };
}

function report(overrides: Partial<ExtractionShadowReport> = {}): ExtractionShadowReport {
  return {
    windowDays: 30,
    since: 1_787_400_000,
    providerId: "aip_candidate",
    providerName: "Fine-tuned Qwen",
    sampled: 12,
    pending: 1,
    completed: 9,
    failed: 1,
    skipped: 1,
    scored: 8,
    truncated: false,
    better: 3,
    worse: 1,
    same: 4,
    candidate: side(40, 37),
    production: side(40, 34),
    fields: [
      {
        key: "rate",
        candidateScored: 8,
        candidateCorrect: 8,
        candidateAccuracy: 1,
        productionScored: 8,
        productionCorrect: 6,
        productionAccuracy: 0.75,
      },
    ],
    costUsd: "0.420000",
    avgLatencyMs: 2100,
    ...overrides,
  } as ExtractionShadowReport;
}

let restoreLayout = () => {};

beforeEach(() => {
  restoreLayout = stubLayout();
  permissions.denied.clear();
  fetchSettings.mockReset();
  fetchReport.mockReset();
  fetchResult.mockReset();
  updateSettings.mockReset();
  table.rows = [];
  table.openRow = null;
});

afterEach(() => {
  cleanup();
  restoreLayout();
});

function renderView() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ShadowView />
    </QueryClientProvider>,
  );
}

describe("ShadowView", () => {
  it("shows the candidate and both sides of the comparison", async () => {
    fetchSettings.mockResolvedValue(settings());
    fetchReport.mockResolvedValue(report());
    renderView();

    expect(await screen.findByText("Fine-tuned Qwen · trenova-extract")).toBeInTheDocument();
    expect(screen.getByText("10%")).toBeInTheDocument();
    expect(await screen.findByText("93%")).toBeInTheDocument();
    expect(screen.getByText("85%")).toBeInTheDocument();
    expect(screen.getByText("3 better, 1 worse")).toBeInTheDocument();
    expect(screen.getByText("8 pts above production")).toBeInTheDocument();
    expect(screen.getByText("+25 pts")).toBeInTheDocument();
  });

  it("asks for a candidate before there is anything to compare", async () => {
    fetchSettings.mockResolvedValue(settings({ enabled: false, providerId: null }));
    fetchReport.mockResolvedValue(report({ providerId: null, providerName: "" }));
    renderView();

    expect(
      await screen.findByText(
        "Choose a candidate provider to start shadowing production extraction.",
      ),
    ).toBeInTheDocument();
    expect(screen.queryByText("Candidate accuracy")).not.toBeInTheDocument();
  });

  it("hides the settings from someone who cannot change them", async () => {
    permissions.denied.add(`${Resource.AgentEvalSuite}:${Operation.Update}`);
    fetchSettings.mockResolvedValue(settings());
    fetchReport.mockResolvedValue(report());
    renderView();

    await screen.findByText("Fine-tuned Qwen · trenova-extract");
    expect(screen.queryByRole("button", { name: "Edit settings" })).not.toBeInTheDocument();
  });

  it("saves the settings with the version it read", async () => {
    const user = userEvent.setup();
    fetchSettings.mockResolvedValue(settings());
    fetchReport.mockResolvedValue(report());
    updateSettings.mockResolvedValue(settings({ samplePercent: 25, version: 4 }));
    renderView();

    await user.click(await screen.findByRole("button", { name: "Edit settings" }));
    const dialog = await screen.findByRole("dialog");
    const share = within(dialog).getByLabelText("Share of extractions (%)");
    await user.clear(share);
    await user.type(share, "25");
    await user.click(within(dialog).getByRole("button", { name: "Save" }));

    await waitFor(() => expect(updateSettings).toHaveBeenCalledTimes(1));
    expect(updateSettings).toHaveBeenCalledWith({
      enabled: true,
      providerId: "aip_candidate",
      samplePercent: 25,
      dailyLimit: 200,
      version: 3,
    });
  });

  it("will not save a share or limit the server would refuse", async () => {
    const user = userEvent.setup();
    fetchSettings.mockResolvedValue(settings());
    fetchReport.mockResolvedValue(report());
    renderView();

    await user.click(await screen.findByRole("button", { name: "Edit settings" }));
    const dialog = await screen.findByRole("dialog");
    const share = within(dialog).getByLabelText("Share of extractions (%)");
    await user.clear(share);
    await user.type(share, "0");

    expect(
      within(dialog).getByText("Sample between 1 and 100 percent of extractions"),
    ).toBeInTheDocument();
    expect(within(dialog).getByRole("button", { name: "Save" })).toBeDisabled();

    await user.clear(share);
    await user.type(share, "15");
    const limit = within(dialog).getByLabelText("Most per 24 hours");
    await user.clear(limit);
    await user.type(limit, "9000");
    expect(within(dialog).getByRole("button", { name: "Save" })).toBeDisabled();
    expect(updateSettings).not.toHaveBeenCalled();
  });

  it("will not turn the shadow on without a candidate", async () => {
    const user = userEvent.setup();
    fetchSettings.mockResolvedValue(settings({ enabled: false, providerId: null }));
    fetchReport.mockResolvedValue(report({ providerId: null }));
    renderView();

    await user.click(await screen.findByRole("button", { name: "Edit settings" }));
    const dialog = await screen.findByRole("dialog");
    await user.click(within(dialog).getByRole("switch", { name: "Shadow production extraction" }));

    expect(
      within(dialog).getByText("Choose the AI provider to shadow production with"),
    ).toBeInTheDocument();
    expect(within(dialog).getByRole("button", { name: "Save" })).toBeDisabled();
  });

  it("puts each field's two readings beside what the person confirmed", async () => {
    fetchSettings.mockResolvedValue(settings());
    fetchReport.mockResolvedValue(report());
    const row = {
      id: "exsr_1",
      status: "Completed",
      verdict: "Better",
      providerName: "Fine-tuned Qwen",
      servedModel: "trenova-extract",
      scoredCount: 2,
      accuracy: 1,
      baselineScoredCount: 2,
      baselineAccuracy: 0.5,
      latencyMs: 1800,
      costUsd: "0.004000",
      createdAt: 1_790_000_000,
    };
    table.rows = [row];
    table.openRow = row;
    fetchResult.mockResolvedValue({
      ...row,
      documentId: "doc_1",
      extractedAt: 1_789_999_000,
      statusReason: "",
      providerId: "aip_candidate",
      productionModel: "frontier-large",
      accepted: true,
      rejectionReason: "",
      correctionId: "aicr_1",
      scoredAt: 1_790_000_100,
      correctCount: 2,
      correctedCount: 0,
      missedCount: 0,
      baselineCorrectCount: 1,
      baselineCorrectedCount: 1,
      baselineMissedCount: 0,
      completedAt: 1_790_000_000,
      updatedAt: 1_790_000_100,
      predicted: null,
      fieldResults: [
        {
          key: "rate",
          predicted: "2563.12",
          confirmed: "2563.12",
          outcome: "Correct",
          source: "ai",
          confidence: 0.9,
        },
        {
          key: "referenceNumber",
          predicted: "OUG-8393964",
          confirmed: "OUG-8393964",
          outcome: "Correct",
          source: "ai",
          confidence: 0.9,
        },
      ],
      baselineFieldResults: [
        {
          key: "rate",
          predicted: "2500.00",
          confirmed: "2563.12",
          outcome: "Corrected",
          source: "ai",
          confidence: 0.9,
        },
        {
          key: "referenceNumber",
          predicted: "OUG-8393964",
          confirmed: "OUG-8393964",
          outcome: "Correct",
          source: "ai",
          confidence: 0.9,
        },
      ],
    } as unknown as ExtractionShadowResultDetail);
    renderView();

    const comparison = await screen.findByRole("region", { name: "Field by field" });
    expect(within(comparison).getByText("1 field differs")).toBeInTheDocument();
    const rate = within(comparison).getByText("Rate").closest("tr");
    expect(rate).not.toBeNull();
    const cells = within(rate as HTMLElement).getAllByRole("cell");
    expect(cells[1]).toHaveTextContent("2563.12");
    expect(cells[2]).toHaveTextContent("2563.12");
    expect(cells[2]).toHaveTextContent("Correct");
    expect(cells[3]).toHaveTextContent("2500.00");
    expect(cells[3]).toHaveTextContent("Corrected");
  });
});
