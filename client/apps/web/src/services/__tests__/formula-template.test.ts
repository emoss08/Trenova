import { beforeEach, describe, expect, it, vi } from "vitest";
import { FormulaLibraryService } from "../formula-library";
import { FormulaTemplateNotFoundError, FormulaTemplateService } from "../formula-template";

const requestGraphQL = vi.fn();

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: (...args: unknown[]) => requestGraphQL(...args),
}));

vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: { error: vi.fn() },
}));

type SentRequest = {
  operationName: string;
  variables?: Record<string, unknown>;
  signal?: AbortSignal;
};

function sent(index = 0): SentRequest {
  return requestGraphQL.mock.calls[index]?.[0] as SentRequest;
}

const rateInput = {
  name: "rate",
  doc: "What the contract pays per loaded mile.",
  type: "Money/mi",
  descriptor: { kind: "quantity", dimension: "money/distance", unit: "$/mi" },
  required: true,
  default: null,
  hasDefault: false,
  constraint: null,
};

const minimumInput = {
  name: "minimumCharge",
  doc: null,
  type: "Money",
  descriptor: { kind: "money" },
  required: false,
  default: { $q: "0", u: "$" },
  hasDefault: true,
  constraint: null,
};

const template = {
  id: "ft_01",
  organizationId: "org_01",
  businessUnitId: "bu_01",
  name: "Per Mile",
  description: "Linehaul per loaded mile",
  source: 'tariff "Per Mile"\n',
  status: "Draft",
  inputs: [rateInput, minimumInput],
  imports: [],
  tables: [],
  testCount: 1,
  submittedById: null,
  submittedAt: null,
  approvedById: null,
  approvedAt: null,
  reviewComment: "",
  version: 3,
  sourceTemplateId: null,
  sourceVersionNumber: null,
  currentVersionNumber: 2,
  createdAt: 1767225600,
  updatedAt: 1767225600,
  usageCount: 0,
  billing: null,
};

const thread = {
  thread: {
    id: "athr_01",
    businessUnitId: "bu_01",
    organizationId: "org_01",
    userId: "usr_01",
    agentDefinitionId: "agd_01",
    title: "",
    status: "Active",
    lastMessageAt: 0,
    lastReadAt: 0,
    preferredProviderId: null,
    origin: "Formula",
    pinned: false,
    subjectType: "FormulaTemplate",
    subjectId: "ft_01",
    canContinue: true,
    cannotContinueReason: null,
    taintedAt: null,
    version: 1,
    createdAt: 1767225600,
    updatedAt: 1767225600,
  },
  agent: {
    id: "agd_01",
    name: "Formula assistant",
    description: "",
    template: null,
    icon: "",
    accent: "",
    systemKey: "formula_assistant",
    toolNames: [],
    starters: [],
  },
};

beforeEach(() => {
  requestGraphQL.mockReset();
});

describe("FormulaTemplateService", () => {
  it("reads a template whole and keeps a default only where the source declares one", async () => {
    requestGraphQL.mockResolvedValueOnce({ formulaTemplate: template });
    const controller = new AbortController();

    const result = await new FormulaTemplateService().get("ft_01", { signal: controller.signal });

    expect(sent()).toMatchObject({
      operationName: "FormulaTemplateRecord",
      variables: { id: "ft_01" },
      signal: controller.signal,
    });
    expect(result.inputs[0]).not.toHaveProperty("default");
    expect(result.inputs[0]).not.toHaveProperty("hasDefault");
    expect(result.inputs[1]?.default).toEqual({ $q: "0", u: "$" });
  });

  it("raises a not-found error when the template is not the organisation's", async () => {
    requestGraphQL.mockResolvedValueOnce({ formulaTemplate: null });

    await expect(new FormulaTemplateService().get("ft_missing")).rejects.toBeInstanceOf(
      FormulaTemplateNotFoundError,
    );
  });

  it("reads the declared inputs of the version that bills today", async () => {
    requestGraphQL
      .mockResolvedValueOnce({
        formulaTemplate: {
          id: "ft_01",
          inputs: [rateInput],
          billing: { versionNumber: 4, effectiveFrom: 1767225600, pending: false },
        },
      })
      .mockResolvedValueOnce({
        formulaTemplateVersion: { id: "ftv_04", inputs: [minimumInput] },
      });

    const inputs = await new FormulaTemplateService().getDeclaredInputs("ft_01", null);

    expect(sent(0).operationName).toBe("FormulaTemplateDeclaredInputs");
    expect(sent(1)).toMatchObject({
      operationName: "FormulaTemplateVersionInputs",
      variables: { templateId: "ft_01", versionNumber: 4 },
    });
    expect(inputs.map((input) => input.name)).toEqual(["minimumCharge"]);
  });

  it("imports templates with the conflict rule as the schema's enum", async () => {
    requestGraphQL.mockResolvedValueOnce({
      importFormulaTemplates: { created: [template], renamed: { "Per Mile": "Per Mile (2)" } },
    });

    const result = await new FormulaTemplateService().importTemplates({
      exportVersion: "2.0",
      onConflict: "rename",
      templates: [{ name: "Per Mile", source: 'tariff "Per Mile"\n', sourceVersionNumber: 2 }],
    });

    expect(sent().variables).toEqual({
      input: {
        exportVersion: "2.0",
        onConflict: "Rename",
        templates: [
          {
            name: "Per Mile",
            description: null,
            source: 'tariff "Per Mile"\n',
            sourceTemplateId: null,
            sourceVersionNumber: 2,
          },
        ],
      },
    });
    expect(result.renamed).toEqual({ "Per Mile": "Per Mile (2)" });
    expect(result.created).toHaveLength(1);
  });

  it("pages versions with the server's total", async () => {
    requestGraphQL.mockResolvedValueOnce({
      formulaTemplateVersions: { results: [], totalCount: 7, totalCountExact: true },
    });

    const page = await new FormulaTemplateService().listVersions("ft_01", { limit: 5 });

    expect(sent().variables).toEqual({ templateId: "ft_01", limit: 5, offset: null });
    expect(page).toMatchObject({ results: [], count: 7, totalCount: 7, totalCountExact: true });
  });

  it("reads a candidate's refusal apart from a failure in a backtest", async () => {
    const summary = {
      shipmentCount: 2,
      evaluatedCount: 2,
      changedCount: 0,
      increasedCount: 0,
      decreasedCount: 0,
      errorCount: 1,
      currentErrorCount: 0,
      candidateErrorCount: 1,
      refusedCount: 1,
      adjustedCount: 0,
      currentTotal: "100",
      candidateTotal: "0",
      totalDelta: "-100",
      totalDeltaPct: "-100",
      maxIncrease: "0",
      maxDecrease: "0",
    };
    const row = {
      proNumber: "PRO-1",
      currentSource: "Billed",
      currentAmount: "100",
      candidateAmount: "0",
      delta: "0",
      deltaPct: "0",
      currentError: null,
      adjusted: false,
    };
    requestGraphQL.mockResolvedValueOnce({
      formulaTemplateBacktest: {
        results: [
          {
            ...row,
            shipmentId: "shp_1",
            candidateError: "No rate for TX to OK",
            candidateRefused: true,
          },
          {
            ...row,
            shipmentId: "shp_2",
            candidateError: "division by zero",
            candidateRefused: false,
          },
        ],
        summary,
        truncated: false,
      },
    });

    const result = await new FormulaTemplateService().backtest("ft_01", { versionNumber: 2 });

    expect(sent().variables).toEqual({
      templateId: "ft_01",
      input: { source: null, versionNumber: 2, limit: null },
    });
    expect(result.results[0]?.candidateRefusal).toBe("No rate for TX to OK");
    expect(result.results[0]?.candidateError).toBeUndefined();
    expect(result.results[0]?.currentAmount).toBe(100);
    expect(result.results[1]?.candidateRefusal).toBeUndefined();
    expect(result.results[1]?.candidateError).toBe("division by zero");
  });

  it("asks Nova about the draft with an empty template id sent as null", async () => {
    requestGraphQL.mockResolvedValueOnce({
      explainFormulaTemplate: { preview: null, items: [] },
    });

    await new FormulaTemplateService().explainFormula(
      { templateId: "", name: "Per Mile", description: "", source: 'tariff "Per Mile"\n' },
      { shipmentId: "shp_01" },
    );

    expect(sent()).toMatchObject({
      operationName: "ExplainFormulaTemplate",
      variables: {
        input: {
          draft: {
            templateId: null,
            name: "Per Mile",
            description: "",
            source: 'tariff "Per Mile"\n',
          },
          inputs: null,
          shipmentId: "shp_01",
        },
      },
    });
  });

  it("opens the formula assistant's conversation for an unsaved template", async () => {
    requestGraphQL.mockResolvedValueOnce({ openFormulaTemplateAssistantThread: thread });

    const opened = await new FormulaTemplateService().openAssistantThread(null);

    expect(sent()).toMatchObject({
      operationName: "OpenFormulaTemplateAssistantThread",
      variables: { templateId: null },
    });
    expect(opened.thread.id).toBe("athr_01");
    expect(opened.agent.name).toBe("Formula assistant");
  });
});

describe("FormulaLibraryService", () => {
  it("archives a library with the person's confirmation", async () => {
    requestGraphQL.mockResolvedValueOnce({
      archiveFormulaLibrary: {
        id: "fl_01",
        name: "accessorials",
        description: "",
        source: 'library "org.accessorials"\n',
        imports: [],
        status: "Inactive",
        testCount: 0,
      },
    });

    const library = await new FormulaLibraryService().archive("fl_01", { confirmInUse: true });

    expect(sent()).toMatchObject({
      operationName: "ArchiveFormulaLibrary",
      variables: { id: "fl_01", confirmInUse: true },
    });
    expect(library.status).toBe("Inactive");
  });

  it("submits for review with an unpicked reviewer sent as null", async () => {
    requestGraphQL.mockResolvedValueOnce({
      submitFormulaLibrary: {
        id: "fl_01",
        name: "accessorials",
        source: 'library "org.accessorials"\n',
        status: "InReview",
      },
    });

    await new FormulaLibraryService().submit("fl_01", { comment: "Ready", reviewerId: "" });

    expect(sent().variables).toEqual({
      id: "fl_01",
      input: { comment: "Ready", reviewerId: null, reviewerNote: null },
    });
  });
});
